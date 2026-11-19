package diagnosis

import (
	"dst-management-platform-api/aichat"
	"dst-management-platform-api/service"
	"encoding/json"
	"fmt"
	"sort"
	"sync"
	"time"
)

// Session 保存一次诊断的内存状态与 SSE 事件日志。
type Session struct {
	ID          string
	Scope       string
	RoomID      int
	RoomName    string
	Description string
	Actor       service.Actor
	ChatConfig  aichat.ChatConfig
	Status      string
	CreatedAt   time.Time
	ExpiresAt   time.Time

	Steps    []Step
	Evidence map[string]*Evidence
	Report   *Report
	Actions  map[string]*Action
	Apply    *ApplyState
	Error    string

	mu          sync.Mutex
	events      []SSEEvent
	applyEvents []SSEEvent
	nextEventID int64
	nextApplyID int64
	notify      chan struct{}
	applyNotify chan struct{}
	diagDone    bool
	applyDone   bool
	cancel      func()

	finishedAt *time.Time
}

func newSession(id, scope string, roomID int, roomName, description string, actor service.Actor) *Session {
	now := time.Now()
	return &Session{
		ID:          id,
		Scope:       scope,
		RoomID:      roomID,
		RoomName:    roomName,
		Description: description,
		Actor:       actor,
		Status:      StatusRunning,
		CreatedAt:   now,
		ExpiresAt:   now.Add(15 * time.Minute),
		Evidence:    make(map[string]*Evidence),
		Actions:     make(map[string]*Action),
		notify:      make(chan struct{}),
		applyNotify: make(chan struct{}),
	}
}

func (s *Session) expired() bool {
	return time.Now().After(s.ExpiresAt)
}

func (s *Session) addEvent(eventType string, data any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextEventID++
	s.events = append(s.events, SSEEvent{ID: s.nextEventID, Type: eventType, Data: data})
	close(s.notify)
	s.notify = make(chan struct{})
}

func (s *Session) addApplyEvent(eventType string, data any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nextApplyID++
	s.applyEvents = append(s.applyEvents, SSEEvent{ID: s.nextApplyID, Type: eventType, Data: data})
	close(s.applyNotify)
	s.applyNotify = make(chan struct{})
}

func (s *Session) markDiagDone() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.diagDone = true
	close(s.notify)
	s.notify = make(chan struct{})
}

func (s *Session) markApplyDone() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.applyDone = true
	close(s.applyNotify)
	s.applyNotify = make(chan struct{})
}

func (s *Session) eventsAfter(seq int64) ([]SSEEvent, chan struct{}, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []SSEEvent
	for _, event := range s.events {
		if event.ID > seq {
			out = append(out, event)
		}
	}
	return out, s.notify, s.diagDone
}

func (s *Session) applyEventsAfter(seq int64) ([]SSEEvent, chan struct{}, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []SSEEvent
	for _, event := range s.applyEvents {
		if event.ID > seq {
			out = append(out, event)
		}
	}
	return out, s.applyNotify, s.applyDone
}

func (s *Session) setStatus(status, errMessage string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Status = status
	if errMessage != "" {
		s.Error = errMessage
	}
	if status != StatusRunning {
		now := time.Now()
		s.finishedAt = &now
	}
}

func (s *Session) setReport(report *Report) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Report = report
}

func (s *Session) getReport() *Report {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Report
}

// isCanceled 判断用户是否已取消本次诊断。
func (s *Session) isCanceled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Status == StatusCanceled
}

// finishWithReport 仅在诊断仍在运行时写入最终报告并置为成功。
// 返回 false 表示诊断已被取消或已结束，调用方不应再写成功事件。
func (s *Session) finishWithReport(report *Report) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Status != StatusRunning {
		return false
	}
	s.Report = report
	s.Status = StatusSucceeded
	now := time.Now()
	s.finishedAt = &now
	return true
}

// failIfActive 仅在诊断仍在运行时置为失败并记录错误。
// 返回 false 表示诊断已被取消或已结束，调用方不应再写失败事件。
func (s *Session) failIfActive(message string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Status != StatusRunning {
		return false
	}
	s.Status = StatusFailed
	s.Error = message
	now := time.Now()
	s.finishedAt = &now
	return true
}

func (s *Session) stepCount() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.Steps)
}

func (s *Session) addStep(step Step) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Steps = append(s.Steps, step)
}

func (s *Session) updateStep(index int, status string, duration time.Duration, errMessage string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.Steps {
		if s.Steps[i].Index == index {
			s.Steps[i].Status = status
			s.Steps[i].DurationMs = duration.Milliseconds()
			s.Steps[i].Error = errMessage
			return
		}
	}
}

func (s *Session) nextEvidenceID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("ev_%d", len(s.Evidence)+1)
}

func (s *Session) addEvidence(evidence *Evidence) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Evidence[evidence.EvidenceID] = evidence
}

func (s *Session) getEvidence(id string) *Evidence {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Evidence[id]
}

func (s *Session) evidenceList(ids []string) []*Evidence {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Evidence, 0, len(ids))
	for _, id := range ids {
		if evidence, ok := s.Evidence[id]; ok {
			out = append(out, evidence)
		}
	}
	return out
}

func (s *Session) upsertAction(action *Action) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Actions[action.ActionID] = action
}

func (s *Session) getAction(id string) *Action {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Actions[id]
}

func (s *Session) actionList() []*Action {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.actionListLocked()
}

// actionListLocked 返回会话内动作指针，调用方需持有 s.mu。
// 会话内的 Report.Actions 与这里返回的指针是同一批对象，状态只有一份。
func (s *Session) actionListLocked() []*Action {
	out := make([]*Action, 0, len(s.Actions))
	for _, action := range s.Actions {
		out = append(out, action)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Order == out[j].Order {
			return out[i].ActionID < out[j].ActionID
		}
		return out[i].Order < out[j].Order
	})
	return out
}

// actionsJSON 在锁内序列化动作列表，避免与修复执行时的状态更新并发读写。
func (s *Session) actionsJSON() ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return json.Marshal(s.actionListLocked())
}

func (s *Session) setActionStatus(id, status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if action, ok := s.Actions[id]; ok {
		action.Status = status
	}
}

// setActionParams 覆盖动作参数（用户授权时可以改写参数），与快照读取共用同一把锁。
func (s *Session) setActionParams(id string, params map[string]any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if action, ok := s.Actions[id]; ok {
		action.Params = params
	}
}

func (s *Session) setActionResult(result ActionResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if action, ok := s.Actions[result.ActionID]; ok {
		action.Status = result.Status
		action.ApplyResult = &result
	}
}

func (s *Session) setApply(apply *ApplyState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Apply = apply
}

func (s *Session) getApply() *ApplyState {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Apply
}

func (s *Session) appendApplyResult(result ActionResult) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Apply != nil {
		s.Apply.Results = append(s.Apply.Results, result)
	}
}

func (s *Session) finishApply(status string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Apply != nil {
		s.Apply.Status = status
		now := time.Now()
		s.Apply.FinishedAt = &now
	}
}

func (s *Session) canApply() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Status != StatusSucceeded || s.expired() || len(s.Actions) == 0 {
		return false
	}
	if s.Apply != nil {
		return false
	}
	return true
}

func (s *Session) snapshot() *SessionSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	steps := make([]Step, len(s.Steps))
	copy(steps, s.Steps)
	var report *Report
	if s.Report != nil {
		copyReport := *s.Report
		copyReport.References = make([]Reference, len(s.Report.References))
		copy(copyReport.References, s.Report.References)
		copyReport.Findings = make([]*Finding, 0, len(s.Report.Findings))
		for _, finding := range s.Report.Findings {
			if finding == nil {
				continue
			}
			copyFinding := *finding
			copyFinding.EvidenceIDs = append([]string(nil), finding.EvidenceIDs...)
			copyFinding.Solutions = append([]string(nil), finding.Solutions...)
			copyFinding.Evidence = append([]*Evidence(nil), finding.Evidence...)
			copyReport.Findings = append(copyReport.Findings, &copyFinding)
		}
		copyReport.Actions = make([]*Action, 0, len(s.Actions))
		for _, action := range s.actionListLocked() {
			if action == nil {
				continue
			}
			copyAction := *action
			copyAction.EvidenceIDs = append([]string(nil), action.EvidenceIDs...)
			if action.ApplyResult != nil {
				result := *action.ApplyResult
				copyAction.ApplyResult = &result
			}
			copyReport.Actions = append(copyReport.Actions, &copyAction)
		}
		report = &copyReport
	}
	var apply *ApplyState
	if s.Apply != nil {
		copyApply := *s.Apply
		copyApply.Results = append([]ActionResult(nil), s.Apply.Results...)
		apply = &copyApply
	}
	canApply := s.Status == StatusSucceeded && !s.expired() && len(s.Actions) > 0 && s.Apply == nil
	return &SessionSnapshot{
		DiagnosisID: s.ID,
		Scope:       s.Scope,
		RoomID:      s.RoomID,
		RoomName:    s.RoomName,
		Description: s.Description,
		Status:      s.Status,
		CreatedAt:   s.CreatedAt,
		FinishedAt:  s.finishedAt,
		Steps:       steps,
		Report:      report,
		CanApply:    canApply,
		Apply:       apply,
		Error:       s.Error,
	}
}

// sessionStore 是进程内诊断会话存储。诊断会话短生命周期，不落库。
type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]*Session
	max      int
}

func newSessionStore() *sessionStore {
	return &sessionStore{
		sessions: make(map[string]*Session),
		max:      100,
	}
}

func (st *sessionStore) put(session *Session) {
	st.mu.Lock()
	defer st.mu.Unlock()
	st.cleanupLocked()
	if len(st.sessions) >= st.max {
		var oldest *Session
		for _, s := range st.sessions {
			if oldest == nil || s.CreatedAt.Before(oldest.CreatedAt) {
				oldest = s
			}
		}
		if oldest != nil {
			delete(st.sessions, oldest.ID)
		}
	}
	st.sessions[session.ID] = session
}

func (st *sessionStore) get(id string) *Session {
	st.mu.Lock()
	defer st.mu.Unlock()
	session, ok := st.sessions[id]
	if !ok {
		return nil
	}
	if session.expired() {
		delete(st.sessions, id)
		return nil
	}
	return session
}

func (st *sessionStore) delete(id string) {
	st.mu.Lock()
	defer st.mu.Unlock()
	delete(st.sessions, id)
}

func (st *sessionStore) hasActive(actorUsername string, roomID int, scope string) bool {
	st.mu.Lock()
	defer st.mu.Unlock()
	for _, session := range st.sessions {
		if session.expired() {
			continue
		}
		if session.Actor.Username != actorUsername || session.Scope != scope || session.RoomID != roomID {
			continue
		}
		if session.Status == StatusRunning {
			return true
		}
	}
	return false
}

func (st *sessionStore) cleanupLocked() {
	now := time.Now()
	for id, session := range st.sessions {
		if now.After(session.ExpiresAt) {
			delete(st.sessions, id)
		}
	}
}
