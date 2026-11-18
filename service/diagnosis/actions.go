package diagnosis

import (
	"context"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/utils"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ActionSpec 描述一个可被 AI 推荐、由用户授权执行的修复动作。
type ActionSpec struct {
	ID           string
	TitleKey     string
	Risk         string
	Reversible   bool
	AdminOnly    bool
	Confirmation string
	Order        int
	DependsOn    []string
	Validate     func(ctx context.Context, s *Service, scope *Scope, params map[string]any) error
	Execute      func(ctx context.Context, s *Service, scope *Scope, params map[string]any) error
	Verify       func(ctx context.Context, s *Service, scope *Scope, params map[string]any) (bool, string)
}

var (
	workshopPattern = regexp.MustCompile(`workshop-(\d+)`)
	sessionPattern  = regexp.MustCompile(`session/([0-9A-Fa-f]+)`)
)

func (s *Service) actionSpec(id string) (*ActionSpec, bool) {
	switch id {
	case "start_world":
		return &ActionSpec{
			ID: "start_world", TitleKey: "diagnosis action start world", Risk: RiskLow, Reversible: true,
			Confirmation: ConfirmNone, Order: 50,
			Validate: s.validateWorldID,
			Execute:  executeStartWorld,
			Verify:   verifyWorldRunning,
		}, true
	case "restart_world":
		return &ActionSpec{
			ID: "restart_world", TitleKey: "diagnosis action restart world", Risk: RiskLow, Reversible: true,
			Confirmation: ConfirmNone, Order: 50,
			Validate: s.validateWorldID,
			Execute:  executeRestartWorld,
			Verify:   verifyWorldRunning,
		}, true
	case "restart_room":
		return &ActionSpec{
			ID: "restart_room", TitleKey: "diagnosis action restart room", Risk: RiskMedium, Reversible: true,
			Confirmation: ConfirmNormal, Order: 60,
			Validate: s.validateRoomOnly,
			Execute:  executeRestartRoom,
			Verify:   verifyRoomRunning,
		}, true
	case "clean_logs":
		return &ActionSpec{
			ID: "clean_logs", TitleKey: "diagnosis action clean logs", Risk: RiskLow, Reversible: false,
			Confirmation: ConfirmNone, Order: 10,
			Validate: s.validateCleanLogs,
			Execute:  executeCleanLogs,
			Verify:   verifyCleanLogs,
		}, true
	case "use_default_token":
		return &ActionSpec{
			ID: "use_default_token", TitleKey: "diagnosis action use default token", Risk: RiskMedium, Reversible: true,
			Confirmation: ConfirmNormal, Order: 15,
			Validate: s.validateUseDefaultToken,
			Execute:  executeUseDefaultToken,
			Verify:   verifyDefaultToken,
		}, true
	case "disable_mod":
		return &ActionSpec{
			ID: "disable_mod", TitleKey: "diagnosis action disable mod", Risk: RiskMedium, Reversible: true,
			Confirmation: ConfirmNormal, Order: 20,
			Validate: s.validateModID,
			Execute:  executeDisableMod,
			Verify:   verifyModDisabled,
		}, true
	case "enable_mod":
		return &ActionSpec{
			ID: "enable_mod", TitleKey: "diagnosis action enable mod", Risk: RiskMedium, Reversible: true,
			Confirmation: ConfirmNormal, Order: 20,
			Validate: s.validateModID,
			Execute:  executeEnableMod,
			Verify:   verifyModEnabled,
		}, true
	case "delete_snapshot":
		return &ActionSpec{
			ID: "delete_snapshot", TitleKey: "diagnosis action delete snapshot", Risk: RiskHigh, Reversible: false,
			Confirmation: ConfirmTyped, Order: 40,
			Validate: s.validateSnapshot,
			Execute:  executeDeleteSnapshot,
			Verify:   verifySnapshotDeleted,
		}, true
	case "restore_backup":
		return &ActionSpec{
			ID: "restore_backup", TitleKey: "diagnosis action restore backup", Risk: RiskHigh, Reversible: false,
			Confirmation: ConfirmTyped, Order: 40,
			Validate: s.validateBackup,
			Execute:  executeRestoreBackup,
			Verify:   verifyRoomRunning,
		}, true
	case "update_game":
		return &ActionSpec{
			ID: "update_game", TitleKey: "diagnosis action update game", Risk: RiskMedium, Reversible: false,
			AdminOnly: true, Confirmation: ConfirmTyped, Order: 30,
			Validate: s.validateUpdateGame,
			Execute:  executeUpdateGame,
			Verify:   verifyGameVersion,
		}, true
	default:
		return nil, false
	}
}

func (s *Service) buildAction(session *Session, current *Scope, actionID string, params map[string]any, evidenceIDs []string, reason string) (*Action, error) {
	spec, ok := s.actionSpec(actionID)
	if !ok {
		return nil, fmt.Errorf("不支持的动作: %s", actionID)
	}
	if params == nil {
		params = map[string]any{}
	}
	if len(evidenceIDs) == 0 {
		return nil, fmt.Errorf("动作 %s 必须引用至少一条证据", actionID)
	}
	for _, id := range evidenceIDs {
		if session.getEvidence(id) == nil {
			return nil, fmt.Errorf("证据不存在: %s", id)
		}
	}

	// 从证据中补全动态参数，避免模型需要精确复制日志内容。
	switch actionID {
	case "disable_mod":
		if _, ok := params["modID"]; !ok {
			for _, id := range evidenceIDs {
				if match := workshopPattern.FindStringSubmatch(session.getEvidence(id).Detail); len(match) == 2 {
					if modID, err := strconv.Atoi(match[1]); err == nil {
						params["modID"] = modID
						break
					}
				}
			}
		}
	case "delete_snapshot":
		if _, ok := params["sessionID"]; !ok {
			for _, id := range evidenceIDs {
				if match := sessionPattern.FindStringSubmatch(session.getEvidence(id).Detail); len(match) == 2 {
					params["sessionID"] = match[1]
					break
				}
			}
		}
	}

	scope, err := s.actionScopeForAction(current, actionID, params)
	if err != nil {
		return nil, err
	}
	if spec.AdminOnly && !session.Actor.IsAdmin() {
		return nil, fmt.Errorf("该动作仅管理员可执行")
	}
	if err = spec.Validate(context.Background(), s, scope, params); err != nil {
		return nil, err
	}
	if err = s.requireEvidence(actionID, session, evidenceIDs, params); err != nil {
		return nil, err
	}

	confirmationHint := ""
	switch spec.Confirmation {
	case ConfirmNormal:
		confirmationHint = "请确认执行该动作"
	case ConfirmTyped:
		if actionID == "update_game" {
			confirmationHint = "请输入 UPDATE 以确认"
		} else {
			confirmationHint = fmt.Sprintf("请输入房间名 %s 以确认", scope.Room.GameName)
		}
	}

	return &Action{
		ActionID:          actionID,
		TitleKey:          spec.TitleKey,
		Risk:              spec.Risk,
		Reversible:        spec.Reversible,
		AdminOnly:         spec.AdminOnly,
		ConfirmationLevel: spec.Confirmation,
		ConfirmationHint:  confirmationHint,
		Params:            params,
		Preview:           s.actionPreview(actionID, session.Actor.Lang, scope, params),
		Reason:            reason,
		EvidenceIDs:       evidenceIDs,
		DependsOn:         spec.DependsOn,
		Order:             spec.Order,
		Available:         true,
		Status:            ActionStatusProposed,
	}, nil
}

func (s *Service) actionScopeForAction(current *Scope, actionID string, params map[string]any) (*Scope, error) {
	if current != nil && current.Platform && actionID == "update_game" {
		return current, nil
	}
	return s.actionScope(current, params)
}

func (s *Service) actionScope(current *Scope, params map[string]any) (*Scope, error) {
	if current == nil {
		return nil, fmt.Errorf("缺少诊断上下文")
	}
	if !current.Platform {
		if current.Room == nil || current.Game == nil {
			return s.loadRoomScope(current.Actor, current.RoomID)
		}
		return current, nil
	}
	roomID := s.scopedRoomID(current, params)
	if roomID <= 0 {
		return nil, fmt.Errorf("平台级动作需要 roomID")
	}
	return s.loadRoomScope(current.Actor, roomID)
}

func (s *Service) requireEvidence(actionID string, session *Session, evidenceIDs []string, params map[string]any) error {
	combined := strings.Builder{}
	for _, id := range evidenceIDs {
		if evidence := session.getEvidence(id); evidence != nil {
			combined.WriteString(evidence.Detail)
			combined.WriteString("\n")
		}
	}
	text := combined.String()
	switch actionID {
	case "disable_mod":
		modID := intParam(params, "modID")
		if modID <= 0 || !strings.Contains(text, fmt.Sprintf("workshop-%d", modID)) {
			return fmt.Errorf("禁用模组需要日志中出现 workshop-%d", modID)
		}
	case "delete_snapshot":
		sessionID := stringParam(params, "sessionID")
		if sessionID == "" || !strings.Contains(text, "session/"+sessionID) {
			return fmt.Errorf("删除存档需要日志中出现 session/%s", sessionID)
		}
	case "update_game":
		if !strings.Contains(text, `"gameVersionOutdated": true`) && !strings.Contains(text, `"dstUpdating": true`) {
			return fmt.Errorf("更新游戏需要版本检查证据")
		}
	case "use_default_token":
		lower := strings.ToLower(text)
		if !strings.Contains(lower, "token") && !strings.Contains(text, "令牌") && !strings.Contains(text, "E_ROWID_EXIST") {
			return fmt.Errorf("使用默认令牌需要令牌失效或 E_ROWID_EXIST 的证据")
		}
	}
	return nil
}

func (s *Service) actionPreview(actionID, lang string, scope *Scope, params map[string]any) string {
	worldName := ""
	if worldID := intParam(params, "worldID"); worldID > 0 && scope != nil {
		for _, world := range scope.Worlds {
			if world.ID == worldID {
				worldName = world.WorldName
				break
			}
		}
	}
	zh := lang != "en"
	switch actionID {
	case "start_world":
		if zh {
			return fmt.Sprintf("将启动世界 %s", worldName)
		}
		return fmt.Sprintf("Start world %s", worldName)
	case "restart_world":
		if zh {
			return fmt.Sprintf("将重启世界 %s", worldName)
		}
		return fmt.Sprintf("Restart world %s", worldName)
	case "restart_room":
		if zh {
			return "将停止并重新启动房间内所有世界"
		}
		return "Stop and restart all worlds in this room"
	case "clean_logs":
		if zh {
			return "将清理所选日志文件"
		}
		return "Clean the selected log files"
	case "use_default_token":
		token := utils.GetDstToken()
		if zh {
			return fmt.Sprintf("将房间令牌替换为平台默认游戏令牌：%s（重启房间后生效）", token)
		}
		return fmt.Sprintf("Replace the room token with the platform default game token: %s (takes effect after restarting the room)", token)
	case "disable_mod":
		if zh {
			return fmt.Sprintf("将禁用模组 workshop-%d", intParam(params, "modID"))
		}
		return fmt.Sprintf("Disable mod workshop-%d", intParam(params, "modID"))
	case "enable_mod":
		if zh {
			return fmt.Sprintf("将启用模组 workshop-%d", intParam(params, "modID"))
		}
		return fmt.Sprintf("Enable mod workshop-%d", intParam(params, "modID"))
	case "delete_snapshot":
		if zh {
			return fmt.Sprintf("将备份并删除存档快照 %s", stringParam(params, "sessionID"))
		}
		return fmt.Sprintf("Back up and delete snapshot %s", stringParam(params, "sessionID"))
	case "restore_backup":
		if zh {
			return fmt.Sprintf("将备份当前存档并恢复到 %s", stringParam(params, "backupName"))
		}
		return fmt.Sprintf("Back up the current save and restore %s", stringParam(params, "backupName"))
	case "update_game":
		if zh {
			return "将通过 SteamCMD 更新游戏，可能需要数分钟"
		}
		return "Update the game with SteamCMD; this may take several minutes"
	default:
		return actionID
	}
}

func intParam(params map[string]any, key string) int {
	if params == nil {
		return 0
	}
	switch value := params[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case int64:
		return int(value)
	case jsonNumber:
		number, _ := strconv.Atoi(string(value))
		return number
	case string:
		number, _ := strconv.Atoi(value)
		return number
	default:
		return 0
	}
}

type jsonNumber string

func stringParam(params map[string]any, key string) string {
	if params == nil {
		return ""
	}
	if value, ok := params[key].(string); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func boolParam(params map[string]any, key string) bool {
	if params == nil {
		return false
	}
	value, _ := params[key].(bool)
	return value
}

func (s *Service) validateWorldID(ctx context.Context, svc *Service, scope *Scope, params map[string]any) error {
	worldID := intParam(params, "worldID")
	if worldID <= 0 {
		return fmt.Errorf("worldID 不能为空")
	}
	for _, world := range scope.Worlds {
		if world.ID == worldID {
			return nil
		}
	}
	return fmt.Errorf("世界不存在: %d", worldID)
}

func (s *Service) validateRoomOnly(ctx context.Context, svc *Service, scope *Scope, params map[string]any) error {
	if scope == nil || scope.Room == nil {
		return fmt.Errorf("缺少房间信息")
	}
	return nil
}

func (s *Service) validateCleanLogs(ctx context.Context, svc *Service, scope *Scope, params map[string]any) error {
	targets, ok := params["targets"].(map[string]any)
	if !ok || len(targets) == 0 {
		return fmt.Errorf("targets 不能为空")
	}
	if !scope.Actor.IsAdmin() {
		for _, system := range []string{"runtime", "access", "steam"} {
			if boolParam(targets, system) {
				return fmt.Errorf("普通用户不能清理系统日志")
			}
		}
	}
	return nil
}

func (s *Service) validateUseDefaultToken(ctx context.Context, svc *Service, scope *Scope, params map[string]any) error {
	if scope == nil || scope.Room == nil || scope.Game == nil {
		return fmt.Errorf("缺少房间信息")
	}
	defaultToken := strings.TrimSpace(utils.GetDstToken())
	if defaultToken == "" {
		return fmt.Errorf("默认游戏令牌不可用")
	}
	if strings.TrimSpace(scope.Room.Token) == defaultToken {
		return fmt.Errorf("房间已使用默认游戏令牌")
	}
	return nil
}

func (s *Service) validateModID(ctx context.Context, svc *Service, scope *Scope, params map[string]any) error {
	if intParam(params, "modID") <= 0 {
		return fmt.Errorf("modID 不能为空")
	}
	return nil
}

func (s *Service) validateSnapshot(ctx context.Context, svc *Service, scope *Scope, params map[string]any) error {
	sessionID := stringParam(params, "sessionID")
	if sessionID == "" || !utils.IsSafeString(sessionID) {
		return fmt.Errorf("sessionID 非法")
	}
	snapshots, err := scope.Game.GetSnapshot()
	if err != nil {
		return fmt.Errorf("读取存档快照失败: %w", err)
	}
	for _, snapshot := range snapshots {
		if snapshot.Name == sessionID {
			return nil
		}
	}
	return fmt.Errorf("存档快照不存在: %s", sessionID)
}

func (s *Service) validateBackup(ctx context.Context, svc *Service, scope *Scope, params map[string]any) error {
	name := stringParam(params, "backupName")
	if name == "" || !utils.IsSafePath(name) {
		return fmt.Errorf("backupName 非法")
	}
	backups, err := scope.Game.GetBackups()
	if err != nil {
		return fmt.Errorf("读取备份失败: %w", err)
	}
	for _, backup := range backups {
		if backup.FileName == name {
			return nil
		}
	}
	return fmt.Errorf("备份不存在: %s", name)
}

func (s *Service) validateUpdateGame(ctx context.Context, svc *Service, scope *Scope, params map[string]any) error {
	if !scope.Actor.IsAdmin() {
		return fmt.Errorf("仅管理员可以更新游戏")
	}
	return nil
}

func executeStartWorld(ctx context.Context, s *Service, scope *Scope, params map[string]any) error {
	worldID := intParam(params, "worldID")
	if scope.Game.WorldUpStatus(worldID) {
		return nil
	}
	return scope.Game.StartWorld(worldID)
}

func executeRestartWorld(ctx context.Context, s *Service, scope *Scope, params map[string]any) error {
	worldID := intParam(params, "worldID")
	_ = scope.Game.StopWorld(worldID)
	time.Sleep(2 * time.Second)
	return scope.Game.StartWorld(worldID)
}

func executeRestartRoom(ctx context.Context, s *Service, scope *Scope, params map[string]any) error {
	_ = scope.Game.StopAllWorld()
	time.Sleep(2 * time.Second)
	return scope.Game.StartAllWorld()
}

func executeCleanLogs(ctx context.Context, s *Service, scope *Scope, params map[string]any) error {
	targets, _ := params["targets"].(map[string]any)
	clean := &dst.CleanLogs{
		Game:    boolParam(targets, "game"),
		Chat:    boolParam(targets, "chat"),
		Runtime: boolParam(targets, "runtime"),
		Access:  boolParam(targets, "access"),
		Steam:   boolParam(targets, "steam"),
	}
	if !scope.Game.LogsClean(clean) {
		return fmt.Errorf("清理日志失败")
	}
	return nil
}

func executeUseDefaultToken(ctx context.Context, s *Service, scope *Scope, params map[string]any) error {
	token := strings.TrimSpace(utils.GetDstToken())
	if token == "" {
		return fmt.Errorf("默认游戏令牌不可用")
	}
	// 先写 cluster_token.txt 再更新数据库：写文件失败时数据库保持旧值，重试仍可通过校验。
	if err := scope.Game.UpdateToken(token); err != nil {
		return fmt.Errorf("写入令牌文件失败: %w", err)
	}
	if err := s.roomDao.UpdateRoomToken(scope.RoomID, token); err != nil {
		return fmt.Errorf("更新房间令牌失败: %w", err)
	}
	return nil
}

func executeDisableMod(ctx context.Context, s *Service, scope *Scope, params map[string]any) error {
	return scope.Game.ModDisable(intParam(params, "modID"))
}

func executeEnableMod(ctx context.Context, s *Service, scope *Scope, params map[string]any) error {
	worldID := 0
	for _, world := range scope.Worlds {
		if world.IsMaster {
			worldID = world.ID
			break
		}
	}
	if worldID == 0 && len(scope.Worlds) > 0 {
		worldID = scope.Worlds[0].ID
	}
	if worldID == 0 {
		return fmt.Errorf("房间没有世界")
	}
	modID := intParam(params, "modID")
	if err := scope.Game.ModEnable(worldID, modID, true); err != nil {
		if err = scope.Game.ModEnable(worldID, modID, false); err != nil {
			return err
		}
	}
	return nil
}

func executeDeleteSnapshot(ctx context.Context, s *Service, scope *Scope, params map[string]any) error {
	if err := scope.Game.Backup(); err != nil {
		return fmt.Errorf("删除前备份失败: %w", err)
	}
	return scope.Game.DeleteSnapshot(stringParam(params, "sessionID"))
}

func executeRestoreBackup(ctx context.Context, s *Service, scope *Scope, params map[string]any) error {
	running := make([]int, 0, len(scope.Worlds))
	for _, world := range scope.Worlds {
		if scope.Game.WorldUpStatus(world.ID) {
			running = append(running, world.ID)
		}
	}
	if err := scope.Game.Backup(); err != nil {
		return fmt.Errorf("恢复前备份失败: %w", err)
	}
	saveData, err := scope.Game.Restore(stringParam(params, "backupName"))
	if err != nil {
		return err
	}
	if err = s.roomDao.UpdateRoom(&saveData.Room); err != nil {
		return fmt.Errorf("更新房间失败: %w", err)
	}
	if err = s.worldDao.UpdateWorlds(&saveData.Worlds); err != nil {
		return fmt.Errorf("更新世界失败: %w", err)
	}
	if err = s.roomSettingDao.UpdateRoomSetting(&saveData.RoomSetting); err != nil {
		return fmt.Errorf("更新房间设置失败: %w", err)
	}
	if err = s.aiManager.Reload(scope.RoomID); err != nil {
		logger.Logger.Warnf("恢复备份后重载 AI 对话失败: %v", err)
	}
	runningNames := make(map[string]bool, len(running))
	for _, world := range scope.Worlds {
		for _, worldID := range running {
			if world.ID == worldID {
				runningNames[world.WorldName] = true
			}
		}
	}
	restoredGame := dst.NewGameController(&saveData.Room, &saveData.Worlds, &saveData.RoomSetting, scope.Actor.Lang)
	for _, world := range saveData.Worlds {
		if runningNames[world.WorldName] {
			if err = restoredGame.StartWorld(world.ID); err != nil {
				logger.Logger.Warnf("恢复备份后启动世界 %s 失败: %v", world.WorldName, err)
			}
		}
	}
	return nil
}

func executeUpdateGame(ctx context.Context, s *Service, scope *Scope, params map[string]any) error {
	if !scope.Actor.IsAdmin() {
		return fmt.Errorf("仅管理员可以更新游戏")
	}
	cache.DstUpdating = true
	defer func() { cache.DstUpdating = false }()
	updateCmd := "cd ~/steamcmd && ./steamcmd.sh +login anonymous +force_install_dir ~/dst +app_update 343050 validate +quit"
	if err := utils.BashCMD(updateCmd); err != nil {
		return fmt.Errorf("更新游戏失败: %w", err)
	}

	var globalSettings models.GlobalSetting
	if err := s.globalSettingDao.GetGlobalSetting(&globalSettings); err == nil && globalSettings.AutoUpdateRestart {
		rooms, err := s.roomDao.GetRoomBasic()
		if err == nil {
			for _, room := range *rooms {
				if !room.Status {
					continue
				}
				roomData, worlds, setting, err := dao.FetchGameInfo(room.RoomID)
				if err != nil {
					continue
				}
				game := dst.NewGameController(roomData, worlds, setting, scope.Actor.Lang)
				_ = game.StopAllWorld()
				_ = game.StartAllWorld()
				time.Sleep(5 * time.Second)
			}
		}
	}
	return nil
}

func verifyWorldRunning(ctx context.Context, s *Service, scope *Scope, params map[string]any) (bool, string) {
	worldID := intParam(params, "worldID")
	for i := 0; i < 10; i++ {
		if scope.Game.WorldUpStatus(worldID) {
			return true, "世界已启动"
		}
		time.Sleep(time.Second)
	}
	return false, "世界未启动"
}

func verifyRoomRunning(ctx context.Context, s *Service, scope *Scope, params map[string]any) (bool, string) {
	for i := 0; i < 15; i++ {
		game := scope.Game
		worlds := scope.Worlds
		if scope.RoomID > 0 {
			room, currentWorlds, setting, err := dao.FetchGameInfo(scope.RoomID)
			if err == nil {
				game = dst.NewGameController(room, currentWorlds, setting, scope.Actor.Lang)
				worlds = *currentWorlds
			}
		}
		allUp := len(worlds) > 0
		for _, world := range worlds {
			if !game.WorldUpStatus(world.ID) {
				allUp = false
				break
			}
		}
		if allUp {
			return true, "所有世界已启动"
		}
		time.Sleep(time.Second)
	}
	return false, "部分世界未启动"
}

func verifyCleanLogs(ctx context.Context, s *Service, scope *Scope, params map[string]any) (bool, string) {
	return true, "日志已清理"
}

func verifyDefaultToken(ctx context.Context, s *Service, scope *Scope, params map[string]any) (bool, string) {
	defaultToken := strings.TrimSpace(utils.GetDstToken())
	room, err := s.roomDao.GetRoomByID(scope.RoomID)
	if err != nil {
		return false, "读取房间令牌失败"
	}
	if strings.TrimSpace(room.Token) != defaultToken {
		return false, "房间令牌未更新"
	}
	fileToken, err := scope.Game.GetToken()
	if err != nil {
		return false, "读取令牌文件失败"
	}
	if strings.TrimSpace(fileToken) != defaultToken {
		return false, "令牌文件未更新"
	}
	return true, "已使用默认游戏令牌，重启房间后生效"
}

func verifyModDisabled(ctx context.Context, s *Service, scope *Scope, params map[string]any) (bool, string) {
	modID := intParam(params, "modID")
	worldID := 0
	for _, world := range scope.Worlds {
		if world.IsMaster {
			worldID = world.ID
			break
		}
	}
	if worldID == 0 && len(scope.Worlds) > 0 {
		worldID = scope.Worlds[0].ID
	}
	mods, err := scope.Game.GetEnabledMods(worldID)
	if err != nil {
		return false, "读取模组列表失败"
	}
	for _, mod := range mods {
		if mod.ID == modID {
			return false, "模组仍处于启用状态"
		}
	}
	return true, "模组已禁用"
}

func verifyModEnabled(ctx context.Context, s *Service, scope *Scope, params map[string]any) (bool, string) {
	modID := intParam(params, "modID")
	worldID := 0
	for _, world := range scope.Worlds {
		if world.IsMaster {
			worldID = world.ID
			break
		}
	}
	if worldID == 0 && len(scope.Worlds) > 0 {
		worldID = scope.Worlds[0].ID
	}
	mods, err := scope.Game.GetEnabledMods(worldID)
	if err != nil {
		return false, "读取模组列表失败"
	}
	for _, mod := range mods {
		if mod.ID == modID {
			return true, "模组已启用"
		}
	}
	return false, "模组未启用"
}

func verifySnapshotDeleted(ctx context.Context, s *Service, scope *Scope, params map[string]any) (bool, string) {
	sessionID := stringParam(params, "sessionID")
	snapshots, err := scope.Game.GetSnapshot()
	if err != nil {
		return false, "读取存档快照失败"
	}
	for _, snapshot := range snapshots {
		if snapshot.Name == sessionID {
			return false, "存档快照仍然存在"
		}
	}
	return true, "存档快照已删除"
}

func verifyGameVersion(ctx context.Context, s *Service, scope *Scope, params map[string]any) (bool, string) {
	version := scheduler.GetDSTVersion()
	if version.Local <= 0 {
		return false, "无法读取本地游戏版本"
	}
	return true, "游戏版本已更新"
}

// finishArgs 是 finish_diagnosis 工具的参数。
type finishArgs struct {
	Summary    string          `json:"summary"`
	Severity   string          `json:"severity"`
	Findings   []finishFinding `json:"findings"`
	Actions    []finishAction  `json:"actions"`
	References []Reference     `json:"references"`
}

type finishFinding struct {
	Title       string   `json:"title"`
	Severity    string   `json:"severity"`
	Confidence  float64  `json:"confidence"`
	KBID        string   `json:"kbID"`
	Cause       string   `json:"cause"`
	Solutions   []string `json:"solutions"`
	EvidenceIDs []string `json:"evidenceIDs"`
}

type finishAction struct {
	ActionID    string         `json:"actionID"`
	Params      map[string]any `json:"params"`
	EvidenceIDs []string       `json:"evidenceIDs"`
	Reason      string         `json:"reason"`
}

func (s *Service) buildReport(session *Session, current *Scope, args *finishArgs) (*Report, error) {
	if args == nil {
		return nil, fmt.Errorf("诊断参数为空")
	}
	severity := args.Severity
	switch severity {
	case "low", "medium", "high", "critical":
	default:
		severity = "medium"
	}

	findings := make([]*Finding, 0, len(args.Findings))
	for i, finding := range args.Findings {
		if strings.TrimSpace(finding.Title) == "" {
			return nil, fmt.Errorf("finding.title 不能为空")
		}
		for _, evidenceID := range finding.EvidenceIDs {
			if session.getEvidence(evidenceID) == nil {
				return nil, fmt.Errorf("证据不存在: %s", evidenceID)
			}
		}
		evidence := session.evidenceList(finding.EvidenceIDs)
		findings = append(findings, &Finding{
			FindingID:   fmt.Sprintf("f_%d", i+1),
			Title:       finding.Title,
			Severity:    finding.Severity,
			Confidence:  finding.Confidence,
			KBID:        finding.KBID,
			Cause:       finding.Cause,
			Solutions:   finding.Solutions,
			EvidenceIDs: finding.EvidenceIDs,
			Evidence:    evidence,
		})
	}

	actions := make([]*Action, 0, len(args.Actions))
	seen := make(map[string]struct{}, len(args.Actions))
	for _, item := range args.Actions {
		if _, ok := seen[item.ActionID]; ok {
			continue
		}
		action, err := s.buildAction(session, current, item.ActionID, item.Params, item.EvidenceIDs, item.Reason)
		if err != nil {
			return nil, fmt.Errorf("动作 %s 无效: %w", item.ActionID, err)
		}
		seen[item.ActionID] = struct{}{}
		session.upsertAction(action)
		actions = append(actions, action)
	}

	references := make([]Reference, 0, len(args.References))
	for _, reference := range args.References {
		if utils.IsValidURL(reference.URL) {
			references = append(references, reference)
		}
	}
	return &Report{
		Summary:    args.Summary,
		Severity:   severity,
		Findings:   findings,
		Actions:    actions,
		References: references,
	}, nil
}

func (s *Service) applyAction(ctx context.Context, session *Session, action *Action) ActionResult {
	start := time.Now()
	result := ActionResult{ActionID: action.ActionID, Status: ActionStatusFailed}
	spec, ok := s.actionSpec(action.ActionID)
	if !ok {
		result.Code = constant.CodeDiagnosisActionInvalid
		result.Message = "动作不受支持"
		return result
	}
	scope, err := s.actionScopeForAction(&Scope{
		Actor:    session.Actor,
		Platform: session.Scope == ScopePlatform,
		RoomID:   session.RoomID,
	}, action.ActionID, action.Params)
	if err != nil {
		result.Code = constant.CodeDiagnosisActionPrecondition
		result.Message = err.Error()
		return result
	}
	if spec.AdminOnly && !session.Actor.IsAdmin() {
		result.Code = constant.CodeDiagnosisNoPermission
		result.Message = "仅管理员可执行该动作"
		return result
	}
	if err = spec.Validate(ctx, s, scope, action.Params); err != nil {
		result.Code = constant.CodeDiagnosisActionPrecondition
		result.Message = err.Error()
		return result
	}
	if err = s.requireEvidence(action.ActionID, session, action.EvidenceIDs, action.Params); err != nil {
		result.Code = constant.CodeDiagnosisActionPrecondition
		result.Message = err.Error()
		return result
	}
	if err = spec.Execute(ctx, s, scope, action.Params); err != nil {
		logger.Logger.Errorf("诊断动作执行失败, action: %s, roomID: %d, err: %v", action.ActionID, session.RoomID, err)
		result.Code = constant.CodeDiagnosisApplyFail
		result.Message = err.Error()
		result.DurationMs = time.Since(start).Milliseconds()
		return result
	}
	verified, verifyMessage := spec.Verify(ctx, s, scope, action.Params)
	result.Status = ActionStatusSucceeded
	result.Verified = verified
	result.VerifyMessage = verifyMessage
	if !verified {
		result.Code = constant.CodeDiagnosisVerifyFail
	}
	result.DurationMs = time.Since(start).Milliseconds()
	return result
}

var _ = errors.Is
