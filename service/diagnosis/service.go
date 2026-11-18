package diagnosis

import (
	"context"
	"dst-management-platform-api/aichat"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"encoding/json"
	"errors"
	"fmt"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// maxDescriptionRunes 限制用户提交的异常描述长度，避免超长文本占满提示词。
const maxDescriptionRunes = 1000

// Service 是 AI 诊断模块的业务实现。
type Service struct {
	userDao          *dao.UserDAO
	roomDao          *dao.RoomDAO
	worldDao         *dao.WorldDAO
	roomSettingDao   *dao.RoomSettingDAO
	systemDao        *dao.SystemDAO
	globalSettingDao *dao.GlobalSettingDAO
	diagnosisDao     *dao.DiagnosisDAO
	aiManager        *aichat.Manager
	chatClient       *aichat.ChatClient

	kb *KnowledgeBase

	store *sessionStore

	startMu   sync.Mutex
	lastStart map[string]time.Time
	roomLocks map[int]*sync.Mutex
}

// New 创建诊断模块 service。
func New(
	userDao *dao.UserDAO,
	roomDao *dao.RoomDAO,
	worldDao *dao.WorldDAO,
	roomSettingDao *dao.RoomSettingDAO,
	systemDao *dao.SystemDAO,
	globalSettingDao *dao.GlobalSettingDAO,
	diagnosisDao *dao.DiagnosisDAO,
	aiManager *aichat.Manager,
) *Service {
	kb, kbErr := loadKnowledgeBase()
	if kbErr != nil {
		logger.Logger.Errorf("加载 AI 诊断知识库失败: %v", kbErr)
	} else {
		logKBLoaded(kb)
	}
	if err := diagnosisDao.MarkInterrupted(); err != nil {
		logger.Logger.Warnf("标记中断的诊断记录失败: %v", err)
	}
	return &Service{
		userDao:          userDao,
		roomDao:          roomDao,
		worldDao:         worldDao,
		roomSettingDao:   roomSettingDao,
		systemDao:        systemDao,
		globalSettingDao: globalSettingDao,
		diagnosisDao:     diagnosisDao,
		aiManager:        aiManager,
		chatClient:       aichat.NewChatClient(),
		kb:               kb,
		store:            newSessionStore(),
		lastStart:        make(map[string]time.Time),
		roomLocks:        make(map[int]*sync.Mutex),
	}
}

func (s *Service) hasPermission(actor service.Actor, roomID int) bool {
	if actor.IsAdmin() {
		return true
	}
	dbUser, err := s.userDao.GetUserByUsername(actor.Username)
	if err != nil {
		logger.Logger.Error("查询数据库失败")
		return false
	}
	for _, id := range strings.Split(dbUser.Rooms, ",") {
		if id == strconv.Itoa(roomID) {
			return true
		}
	}
	return false
}

func (s *Service) allowedRoomIDs(actor service.Actor) ([]int, bool) {
	if actor.IsAdmin() {
		return nil, true
	}
	dbUser, err := s.userDao.GetUserByUsername(actor.Username)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败: %v", err)
		return []int{}, false
	}
	var ids []int
	for _, id := range strings.Split(dbUser.Rooms, ",") {
		roomID, err := strconv.Atoi(strings.TrimSpace(id))
		if err == nil && roomID > 0 {
			ids = append(ids, roomID)
		}
	}
	return ids, true
}

// Start 创建一次诊断会话并异步启动 Agent。
func (s *Service) Start(actor service.Actor, input StartInput) service.Result {
	chatConfig, configCode, err := s.resolveChatConfig()
	if err != nil {
		logger.Logger.Warnf("AI 诊断配置不可用: %v", err)
		switch configCode {
		case constant.CodeDiagnosisAIConfigMissing, constant.CodeDiagnosisDBError:
			return service.Fail(configCode)
		default:
			return service.FailMsg(configCode, err.Error())
		}
	}

	scope := input.Scope
	if scope == "" {
		scope = ScopeRoom
	}
	if scope != ScopeRoom && scope != ScopePlatform {
		return service.Fail(constant.CodeDiagnosisBadRequest)
	}

	description := strings.TrimSpace(input.Description)
	if len([]rune(description)) > maxDescriptionRunes {
		return service.Fail(constant.CodeDiagnosisDescriptionTooLong)
	}

	roomID := input.RoomID
	roomName := ""
	if scope == ScopeRoom {
		if roomID <= 0 {
			return service.Fail(constant.CodeDiagnosisBadRequest)
		}
		if !s.hasPermission(actor, roomID) {
			return service.Fail(constant.CodeDiagnosisNoPermission)
		}
		room, err := s.roomDao.GetRoomByID(roomID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return service.Fail(constant.CodeRoomNotFound)
			}
			logger.Logger.Errorf("查询房间失败: %v", err)
			return service.Fail(constant.CodeDiagnosisDBError)
		}
		roomName = room.GameName
	} else {
		if !actor.IsAdmin() {
			return service.Fail(constant.CodeDiagnosisNoPermission)
		}
		roomID = 0
	}

	if s.store.hasActive(actor.Username, roomID, scope) {
		return service.Fail(constant.CodeDiagnosisBusy)
	}

	// 同一账号同一范围 15 秒内只允许发起一次，避免重复消耗 token。
	rateKey := fmt.Sprintf("%s:%s:%d", actor.Username, scope, roomID)
	s.startMu.Lock()
	if last, ok := s.lastStart[rateKey]; ok && time.Since(last) < 15*time.Second {
		s.startMu.Unlock()
		return service.Fail(constant.CodeDiagnosisBusy)
	}
	s.lastStart[rateKey] = time.Now()
	if len(s.lastStart) > 1024 {
		for key, at := range s.lastStart {
			if time.Since(at) > time.Hour {
				delete(s.lastStart, key)
			}
		}
	}
	s.startMu.Unlock()

	random, err := utils.GenerateRandomKey(12)
	if err != nil {
		logger.Logger.Errorf("生成诊断 ID 失败: %v", err)
		return service.Fail(constant.CodeDiagnosisDBError)
	}
	id := "diag_" + random

	session := newSession(id, scope, roomID, roomName, description, actor)
	session.ChatConfig = chatConfig
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	session.cancel = cancel
	s.store.put(session)

	record := &models.DiagnosisRecord{
		ID:          id,
		Scope:       scope,
		RoomID:      roomID,
		RoomName:    roomName,
		Description: description,
		Username:    actor.Username,
		Status:      StatusRunning,
		CreatedAt:   session.CreatedAt,
	}
	if err := s.diagnosisDao.CreateRecord(record); err != nil {
		logger.Logger.Errorf("创建诊断记录失败: %v", err)
		cancel()
		s.store.delete(id)
		return service.Fail(constant.CodeDiagnosisDBError)
	}

	go s.runDiagnosis(ctx, session)

	return service.Success(StartData{
		DiagnosisID: id,
		Scope:       scope,
		Status:      StatusRunning,
		EventsURL:   fmt.Sprintf("/%s/diagnosis/%s/events", constant.ApiVersion, id),
	})
}

// Get 返回诊断快照；会话过期后回退到数据库记录。
func (s *Service) Get(actor service.Actor, id string) service.Result {
	session := s.store.get(id)
	if session != nil {
		if session.Actor.Username != actor.Username && !actor.IsAdmin() {
			return service.Fail(constant.CodeDiagnosisNoPermission)
		}
		return service.Success(session.snapshot())
	}

	record, err := s.getRecord(actor, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return service.Fail(constant.CodeDiagnosisNotFound)
		}
		return service.Fail(constant.CodeDiagnosisDBError)
	}
	actions, _ := s.diagnosisDao.ListActionRecords(id)
	return service.Success(recordSnapshot(record, actions))
}

// Cancel 取消正在进行的诊断。
func (s *Service) Cancel(actor service.Actor, id string) service.Result {
	session := s.store.get(id)
	if session == nil {
		return service.Fail(constant.CodeDiagnosisNotFound)
	}
	if session.Actor.Username != actor.Username && !actor.IsAdmin() {
		return service.Fail(constant.CodeDiagnosisNoPermission)
	}
	if session.Status == StatusSucceeded || session.Status == StatusFailed || session.Status == StatusCanceled {
		return service.Msg("exec success", nil)
	}
	if apply := session.getApply(); apply != nil && apply.Status == StatusRunning {
		return service.Fail(constant.CodeDiagnosisApplyBusy)
	}
	if session.cancel != nil {
		session.cancel()
	}
	session.setStatus(StatusCanceled, "canceled by user")
	session.addEvent(EventNotice, NoticeEventData{Level: "warning", Message: "diagnosis canceled"})
	session.addEvent(EventDone, map[string]any{"status": StatusCanceled, "finishedAt": time.Now()})
	session.markDiagDone()
	s.saveRecordFromSession(session)
	return service.Msg("exec success", nil)
}

// Stream 返回诊断 SSE 事件通道。
func (s *Service) Stream(ctx context.Context, actor service.Actor, id string, after int64) (<-chan SSEEvent, service.Result) {
	session := s.store.get(id)
	if session == nil {
		return nil, service.Fail(constant.CodeDiagnosisNotFound)
	}
	if session.Actor.Username != actor.Username && !actor.IsAdmin() {
		return nil, service.Fail(constant.CodeDiagnosisNoPermission)
	}
	ch := make(chan SSEEvent, 64)
	go func() {
		// 推送协程 panic 会带走整个进程，这里只记录并关闭通道，SSE 连接随之结束
		defer func() {
			if r := recover(); r != nil {
				logger.Logger.Errorf("诊断事件推送 panic, diagnosisID: %s, err: %v\n%s", session.ID, r, debug.Stack())
			}
			close(ch)
		}()
		last := after
		for {
			events, notify, done := session.eventsAfter(last)
			for _, event := range events {
				last = event.ID
				select {
				case ch <- event:
				case <-ctx.Done():
					return
				}
			}
			if done {
				return
			}
			select {
			case <-notify:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, service.Result{Code: 200}
}

// ApplyStream 返回修复执行 SSE 事件通道。
func (s *Service) ApplyStream(ctx context.Context, actor service.Actor, id string, after int64) (<-chan SSEEvent, service.Result) {
	session := s.store.get(id)
	if session == nil {
		return nil, service.Fail(constant.CodeDiagnosisNotFound)
	}
	if session.Actor.Username != actor.Username && !actor.IsAdmin() {
		return nil, service.Fail(constant.CodeDiagnosisNoPermission)
	}
	ch := make(chan SSEEvent, 64)
	go func() {
		// 推送协程 panic 会带走整个进程，这里只记录并关闭通道，SSE 连接随之结束
		defer func() {
			if r := recover(); r != nil {
				logger.Logger.Errorf("诊断修复事件推送 panic, diagnosisID: %s, err: %v\n%s", session.ID, r, debug.Stack())
			}
			close(ch)
		}()
		last := after
		for {
			events, notify, done := session.applyEventsAfter(last)
			for _, event := range events {
				last = event.ID
				select {
				case ch <- event:
				case <-ctx.Done():
					return
				}
			}
			if done {
				return
			}
			select {
			case <-notify:
			case <-ctx.Done():
				return
			}
		}
	}()
	return ch, service.Result{Code: 200}
}

// History 分页查询诊断历史。
func (s *Service) History(actor service.Actor, scope, status string, page, pageSize int) service.Result {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 10
	}
	roomIDs, ok := s.allowedRoomIDs(actor)
	if !ok {
		return service.Fail(constant.CodeDiagnosisDBError)
	}
	if !actor.IsAdmin() && len(roomIDs) == 0 {
		return service.Success(dao.PaginatedResult[HistoryRow]{
			Data: []HistoryRow{}, Page: page, PageSize: pageSize,
		})
	}
	records, err := s.diagnosisDao.ListRecords(roomIDs, scope, status, page, pageSize)
	if err != nil {
		logger.Logger.Errorf("查询诊断历史失败: %v", err)
		return service.Fail(constant.CodeDiagnosisDBError)
	}

	rows := make([]HistoryRow, 0, len(records.Data))
	recordIDs := make([]string, 0, len(records.Data))
	for i := range records.Data {
		recordIDs = append(recordIDs, records.Data[i].ID)
	}
	appliedCounts, err := s.diagnosisDao.CountActionRecordsByStatus(recordIDs, ActionStatusSucceeded)
	if err != nil {
		logger.Logger.Errorf("查询诊断动作记录失败: %v", err)
		return service.Fail(constant.CodeDiagnosisDBError)
	}
	for i := range records.Data {
		record := &records.Data[i]
		var actions []*Action
		_ = json.Unmarshal([]byte(record.Actions), &actions)
		rows = append(rows, HistoryRow{
			DiagnosisID:  record.ID,
			Scope:        record.Scope,
			RoomID:       record.RoomID,
			RoomName:     record.RoomName,
			Description:  record.Description,
			Username:     record.Username,
			Status:       record.Status,
			Severity:     record.Severity,
			Summary:      record.Summary,
			Model:        record.Model,
			ActionCount:  len(actions),
			AppliedCount: appliedCounts[record.ID],
			CreatedAt:    record.CreatedAt,
			FinishedAt:   record.FinishedAt,
		})
	}

	return service.Success(dao.PaginatedResult[HistoryRow]{
		Data:       rows,
		Page:       records.Page,
		PageSize:   records.PageSize,
		TotalCount: records.TotalCount,
	})
}

// HistoryDetail 返回一条历史诊断的完整内容。
func (s *Service) HistoryDetail(actor service.Actor, id string) service.Result {
	record, err := s.getRecord(actor, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return service.Fail(constant.CodeDiagnosisNotFound)
		}
		return service.Fail(constant.CodeDiagnosisDBError)
	}
	actions, err := s.diagnosisDao.ListActionRecords(id)
	if err != nil {
		return service.Fail(constant.CodeDiagnosisDBError)
	}
	return service.Success(recordSnapshot(record, actions))
}

func (s *Service) getRecord(actor service.Actor, id string) (*models.DiagnosisRecord, error) {
	record, err := s.diagnosisDao.GetRecord(id)
	if err != nil {
		return nil, err
	}
	if record.Scope == ScopePlatform {
		if !actor.IsAdmin() {
			return nil, gorm.ErrRecordNotFound
		}
	} else if !s.hasPermission(actor, record.RoomID) {
		return nil, gorm.ErrRecordNotFound
	}
	return record, nil
}

func recordSnapshot(record *models.DiagnosisRecord, actionRecords []models.DiagnosisActionRecord) *SessionSnapshot {
	var findings []*Finding
	var actions []*Action
	_ = json.Unmarshal([]byte(record.Findings), &findings)
	_ = json.Unmarshal([]byte(record.Actions), &actions)

	report := &Report{
		Summary:    record.Summary,
		Severity:   record.Severity,
		Markdown:   record.Report,
		Findings:   findings,
		Actions:    actions,
		Model:      record.Model,
		References: []Reference{},
		Usage: aichat.ChatUsage{
			PromptTokens:     record.PromptTokens,
			CompletionTokens: record.CompletionTokens,
			TotalTokens:      record.PromptTokens + record.CompletionTokens,
		},
	}
	if report.Findings == nil {
		report.Findings = []*Finding{}
	}
	if report.Actions == nil {
		report.Actions = []*Action{}
	}
	if record.References != "" {
		_ = json.Unmarshal([]byte(record.References), &report.References)
	}
	if report.References == nil {
		report.References = []Reference{}
	}

	var apply *ApplyState
	if len(actionRecords) > 0 {
		apply = &ApplyState{
			ApplyID:   record.ID,
			Status:    ActionStatusSucceeded,
			StartedAt: record.CreatedAt,
			Results:   []ActionResult{},
		}
		for _, actionRecord := range actionRecords {
			if actionRecord.Status != ActionStatusSucceeded {
				apply.Status = ActionStatusFailed
			}
			apply.Results = append(apply.Results, ActionResult{
				ActionID:      actionRecord.ActionID,
				Status:        actionRecord.Status,
				Code:          actionRecord.Code,
				Message:       actionRecord.Message,
				Verified:      actionRecord.Verified,
				VerifyMessage: actionRecord.VerifyMessage,
			})
		}
		if record.FinishedAt != nil {
			apply.FinishedAt = record.FinishedAt
		}
	}

	return &SessionSnapshot{
		DiagnosisID: record.ID,
		Scope:       record.Scope,
		RoomID:      record.RoomID,
		RoomName:    record.RoomName,
		Description: record.Description,
		Status:      record.Status,
		CreatedAt:   record.CreatedAt,
		FinishedAt:  record.FinishedAt,
		Steps:       []Step{},
		Report:      report,
		CanApply:    false,
		Apply:       apply,
		Error:       record.Error,
	}
}

func (s *Service) saveRecordFromSession(session *Session) {
	snapshot := session.snapshot()
	record, err := s.diagnosisDao.GetRecord(session.ID)
	if err != nil {
		return
	}
	record.Status = snapshot.Status
	record.Error = snapshot.Error
	record.Description = snapshot.Description
	if snapshot.Report != nil {
		record.Severity = snapshot.Report.Severity
		record.Summary = snapshot.Report.Summary
		record.Report = snapshot.Report.Markdown
		record.Model = snapshot.Report.Model
		record.PromptTokens = snapshot.Report.Usage.PromptTokens
		record.CompletionTokens = snapshot.Report.Usage.CompletionTokens
		findings, _ := json.Marshal(snapshot.Report.Findings)
		actions, _ := session.actionsJSON()
		references, _ := json.Marshal(snapshot.Report.References)
		record.Findings = string(findings)
		record.Actions = string(actions)
		record.References = string(references)
	}
	record.FinishedAt = snapshot.FinishedAt
	if err := s.diagnosisDao.UpdateRecord(record); err != nil {
		logger.Logger.Warnf("更新诊断记录失败: %v", err)
	}
}

// AISettingData 是诊断 LLM 配置接口的响应。
// Effective 始终是诊断专用配置，与 AI 对话配置无关。
type AISettingData struct {
	Source     string                    `json:"source"`
	Configured bool                      `json:"configured"`
	Setting    models.DiagnosisAISetting `json:"setting"`
	Effective  models.DiagnosisAISetting `json:"effective"`
}

// AISettingGet 返回诊断专用 LLM 配置；未配置时不会回退 AI 对话配置。
func (s *Service) AISettingGet(actor service.Actor) service.Result {
	if !actor.IsAdmin() {
		return service.Fail(constant.CodeDiagnosisNoPermission)
	}
	setting, err := s.systemDao.GetAIDiagnosisSetting()
	if err != nil {
		logger.Logger.Errorf("获取诊断 AI 配置失败: %v", err)
		return service.Fail(constant.CodeDiagnosisDBError)
	}

	normalized := normalizeDiagnosisSetting(*setting)
	data := AISettingData{
		Configured: normalized.Configured(),
		Setting:    normalized,
		Source:     "none",
		Effective:  normalized,
	}
	if data.Configured {
		data.Source = "diagnosis"
	}
	return service.Success(data)
}

// AISettingPut 保存诊断专用 LLM 配置；三项都留空表示清除配置，诊断将要求重新配置。
func (s *Service) AISettingPut(actor service.Actor, setting models.DiagnosisAISetting) service.Result {
	if !actor.IsAdmin() {
		return service.Fail(constant.CodeDiagnosisNoPermission)
	}
	setting = normalizeDiagnosisSetting(setting)
	if setting.Configured() {
		if err := diagnosisChatConfig(setting).ValidateEndpoint(); err != nil {
			return service.FailMsg(constant.CodeDiagnosisAIConfigInvalid, err.Error())
		}
	}
	if err := s.systemDao.UpdateAIDiagnosisSetting(&setting); err != nil {
		logger.Logger.Errorf("更新诊断 AI 配置失败: %v", err)
		return service.Fail(constant.CodeDiagnosisDBError)
	}
	return service.Msg("update success", nil)
}

// resolveChatConfig 只读取诊断专用配置，不回退 AI 对话配置。
// 返回的 code 是失败时应写入业务响应的错误码。
func (s *Service) resolveChatConfig() (aichat.ChatConfig, int, error) {
	setting, err := s.systemDao.GetAIDiagnosisSetting()
	if err != nil {
		return aichat.ChatConfig{}, constant.CodeDiagnosisDBError, err
	}
	if !setting.Configured() {
		return aichat.ChatConfig{}, constant.CodeDiagnosisAIConfigMissing, errors.New("AI 诊断尚未配置专用大模型")
	}
	config := diagnosisChatConfig(*setting)
	if err := config.ValidateEndpoint(); err != nil {
		return config, constant.CodeDiagnosisAIConfigInvalid, err
	}
	return config, 0, nil
}

// diagnosisChatConfig 只保留连接信息；采样参数与超时由调用方的 ChatOptions 统一指定。
func diagnosisChatConfig(setting models.DiagnosisAISetting) aichat.ChatConfig {
	return aichat.ChatConfig{
		BaseURL: setting.ChatBaseURL,
		APIKey:  setting.ChatApiKey,
		Model:   setting.ChatModel,
	}.Normalize()
}

func normalizeDiagnosisSetting(setting models.DiagnosisAISetting) models.DiagnosisAISetting {
	setting.ChatBaseURL = strings.TrimSpace(setting.ChatBaseURL)
	setting.ChatApiKey = strings.TrimSpace(setting.ChatApiKey)
	setting.ChatModel = strings.TrimSpace(setting.ChatModel)
	return setting
}

func (s *Service) roomLock(roomID int) *sync.Mutex {
	s.startMu.Lock()
	defer s.startMu.Unlock()
	lock, ok := s.roomLocks[roomID]
	if !ok {
		lock = &sync.Mutex{}
		s.roomLocks[roomID] = lock
	}
	return lock
}
