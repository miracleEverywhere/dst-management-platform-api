package diagnosis

import (
	"context"
	"dst-management-platform-api/aichat"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/disk"
)

const (
	maxLogLines        = 200
	maxLogLineRunes    = 500
	maxToolResultBytes = 16 * 1024
)

var (
	secretPattern = regexp.MustCompile(`(?i)(cluster[_-]?token|cluster[_-]?password|password|passwd|api[_-]?key|authorization|secret)\s*[:=]\s*[^\s,;]+`)
	bearerPattern = regexp.MustCompile(`(?i)bearer\s+[A-Za-z0-9._\-]+`)
)

// Scope 是工具和动作执行时的房间上下文。
type Scope struct {
	Actor    service.Actor
	Platform bool
	RoomID   int
	Room     *models.Room
	Worlds   []models.World
	Setting  *models.RoomSetting
	Game     *dst.Game
}

func (s *Service) loadRoomScope(actor service.Actor, roomID int) (*Scope, error) {
	room, worlds, setting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		return nil, err
	}
	return &Scope{
		Actor:   actor,
		RoomID:  roomID,
		Room:    room,
		Worlds:  *worlds,
		Setting: setting,
		Game:    dst.NewGameController(room, worlds, setting, actor.Lang),
	}, nil
}

func (s *Service) toolSpecs() []aichat.Tool {
	return []aichat.Tool{
		tool("get_system_metrics", "获取平台 CPU、内存、磁盘、负载、运行时长等系统指标。", objectSchema(nil)),
		tool("get_room_summary", "获取房间、世界、端口、运行状态与房间设置。平台诊断时可传 roomID=0 获取全部房间。", objectSchema(map[string]any{
			"roomID": map[string]any{"type": "integer", "description": "房间 ID；平台诊断时可选"},
		})),
		tool("get_world_status", "获取单个世界的进程状态、CPU、内存与磁盘占用。", objectSchema(map[string]any{
			"roomID":  map[string]any{"type": "integer", "description": "平台诊断时必填"},
			"worldID": map[string]any{"type": "integer", "description": "世界 ID"},
		}, "worldID")),
		tool("get_logs", "读取日志尾部，可按关键词过滤。game 日志需要房间权限；runtime/steam/access 仅管理员。", objectSchema(map[string]any{
			"scope":   map[string]any{"type": "string", "enum": []string{"game", "runtime", "steam", "access"}},
			"roomID":  map[string]any{"type": "integer"},
			"worldID": map[string]any{"type": "integer", "description": "game 日志可选，省略时读取所有世界"},
			"lines":   map[string]any{"type": "integer", "minimum": 1, "maximum": maxLogLines},
			"keyword": map[string]any{"type": "string", "description": "字面关键词，不区分大小写"},
		}, "scope")),
		tool("get_mods", "获取房间某个世界已启用的模组 ID 列表。", objectSchema(map[string]any{
			"roomID":  map[string]any{"type": "integer"},
			"worldID": map[string]any{"type": "integer"},
		})),
		tool("get_game_version", "获取平台版本、DST 本地/远端版本与是否正在更新。", objectSchema(nil)),
		tool("get_scheduler", "获取当前时间、时区与房间定时任务名称。", objectSchema(map[string]any{
			"roomID": map[string]any{"type": "integer"},
		})),
		tool("get_backups", "获取房间最近的备份文件列表。", objectSchema(map[string]any{
			"roomID": map[string]any{"type": "integer"},
		})),
		tool("search_kb", "检索诊断知识库，返回相关规则、判断依据与处理步骤。", objectSchema(map[string]any{
			"query": map[string]any{"type": "string", "description": "错误关键词或问题描述"},
		}, "query")),
		tool("propose_action", "登记一个候选修复动作，必须引用已获取的证据。动作不会立即执行。", objectSchema(map[string]any{
			"actionID":    map[string]any{"type": "string"},
			"params":      map[string]any{"type": "object"},
			"evidenceIDs": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"reason":      map[string]any{"type": "string"},
		}, "actionID", "evidenceIDs")),
		s.finishToolSpec(),
	}
}

// finishToolSpec 是收尾阶段唯一向模型暴露的工具，主循环与收尾阶段共用同一份定义。
func (s *Service) finishToolSpec() aichat.Tool {
	return tool("finish_diagnosis", "提交最终诊断结论、发现和候选修复动作。只能调用一次。", finishSchema())
}

func tool(name, description string, parameters json.RawMessage) aichat.Tool {
	return aichat.Tool{
		Type: "function",
		Function: aichat.ToolFunction{
			Name:        name,
			Description: description,
			Parameters:  parameters,
		},
	}
}

func objectSchema(properties map[string]any, required ...string) json.RawMessage {
	// 无参工具必须序列化为 {}，为 nil 时 encoding/json 会写成 null，
	// 部分模型服务会因 "null is not of type object" 直接返回 HTTP 400。
	if properties == nil {
		properties = map[string]any{}
	}
	schema := map[string]any{
		"type":       "object",
		"properties": properties,
	}
	if len(required) > 0 {
		schema["required"] = required
	}
	data, _ := json.Marshal(schema)
	return data
}

func finishSchema() json.RawMessage {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"summary":  map[string]any{"type": "string"},
			"severity": map[string]any{"type": "string", "enum": []string{"low", "medium", "high", "critical"}},
			"findings": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"title":       map[string]any{"type": "string"},
						"severity":    map[string]any{"type": "string"},
						"confidence":  map[string]any{"type": "number"},
						"kbID":        map[string]any{"type": "string"},
						"cause":       map[string]any{"type": "string"},
						"solutions":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"evidenceIDs": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					},
					"required": []string{"title", "severity", "evidenceIDs"},
				},
			},
			"actions": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"actionID":    map[string]any{"type": "string"},
						"params":      map[string]any{"type": "object"},
						"evidenceIDs": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
						"reason":      map[string]any{"type": "string"},
					},
					"required": []string{"actionID", "evidenceIDs"},
				},
			},
			"references": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"title": map[string]any{"type": "string"},
						"url":   map[string]any{"type": "string"},
					},
					"required": []string{"title", "url"},
				},
			},
		},
		"required": []string{"summary", "severity", "findings", "actions"},
	}
	data, _ := json.Marshal(schema)
	return data
}

func redact(text string) string {
	text = secretPattern.ReplaceAllString(text, "***")
	text = bearerPattern.ReplaceAllString(text, "***")
	return text
}

func truncateRunes(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit]) + "...(truncated)"
}

func capText(text string) (string, bool) {
	if len(text) <= maxToolResultBytes {
		return text, false
	}
	runes := []rune(text)
	if len(runes) > maxToolResultBytes/3 {
		runes = runes[:maxToolResultBytes/3]
	}
	return string(runes) + "\n...(truncated)", true
}

func toolError(message string) string {
	data, _ := json.Marshal(map[string]any{"ok": false, "error": redact(message)})
	return string(data)
}

// executeTool 执行一次只读工具或动作提案，返回给模型的数据与证据。
func (s *Service) executeTool(ctx context.Context, session *Session, current *Scope, name string, raw json.RawMessage) (any, *Evidence, error) {
	switch name {
	case "get_system_metrics":
		return s.toolSystemMetrics(session)
	case "get_room_summary":
		return s.toolRoomSummary(current, raw)
	case "get_world_status":
		return s.toolWorldStatus(current, raw)
	case "get_logs":
		return s.toolLogs(session, current, raw)
	case "get_mods":
		return s.toolMods(current, raw)
	case "get_game_version":
		return s.toolGameVersion()
	case "get_scheduler":
		return s.toolScheduler(current, raw)
	case "get_backups":
		return s.toolBackups(current, raw)
	case "search_kb":
		var args struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal(raw, &args); err != nil {
			return nil, nil, fmt.Errorf("参数解析失败: %w", err)
		}
		views := s.kb.Search(args.Query, session.Actor.Lang, 3)
		return map[string]any{"entries": views}, &Evidence{
			Tool:    name,
			Source:  "knowledge_base",
			Summary: fmt.Sprintf("检索知识库: %s", truncateRunes(args.Query, 80)),
			Detail:  mustJSON(views),
		}, nil
	case "propose_action":
		return s.toolProposeAction(session, current, raw)
	case "finish_diagnosis":
		return s.toolFinishDiagnosis(session, current, raw)
	default:
		return nil, nil, fmt.Errorf("未知工具: %s", name)
	}
}

func mustJSON(value any) string {
	data, _ := json.MarshalIndent(value, "", "  ")
	return string(data)
}

func (s *Service) toolSystemMetrics(session *Session) (any, *Evidence, error) {
	diskFreeGB := 0.0
	if usage, err := disk.Usage(string(os.PathSeparator)); err == nil {
		diskFreeGB = float64(usage.Free) / (1024 * 1024 * 1024)
	}
	data := map[string]any{
		"cpuPercent":      utils.CpuUsage(),
		"memoryPercent":   utils.MemoryUsage(),
		"diskPercent":     utils.DiskUsage(),
		"diskFreeGB":      diskFreeGB,
		"cpuCores":        runtime.NumCPU(),
		"os":              runtime.GOOS,
		"arch":            runtime.GOARCH,
		"platformVersion": constant.Version,
		"dstUpdating":     cache.DstUpdating,
		"time":            time.Now().Format(time.RFC3339),
		"timezone":        time.Local.String(),
	}
	return data, &Evidence{
		Tool:    "get_system_metrics",
		Source:  "system",
		Summary: "读取系统 CPU、内存、磁盘与运行状态",
		Detail:  mustJSON(data),
	}, nil
}

func (s *Service) toolRoomSummary(current *Scope, raw json.RawMessage) (any, *Evidence, error) {
	var args struct {
		RoomID int `json:"roomID"`
	}
	_ = json.Unmarshal(raw, &args)

	if current.Platform {
		roomsBasic, err := s.roomDao.GetRoomBasic()
		if err != nil {
			return nil, nil, fmt.Errorf("查询房间列表失败: %w", err)
		}
		type roomItem struct {
			RoomID      int    `json:"roomID"`
			GameName    string `json:"gameName"`
			Status      bool   `json:"status"`
			RunningText string `json:"runningText"`
		}
		items := make([]roomItem, 0, len(*roomsBasic))
		for i, room := range *roomsBasic {
			if i >= 20 {
				break
			}
			running := "stopped"
			if room.Status {
				running = "enabled"
			}
			items = append(items, roomItem{RoomID: room.RoomID, GameName: room.RoomName, Status: room.Status, RunningText: running})
		}
		data := map[string]any{"scope": ScopePlatform, "roomCount": len(*roomsBasic), "rooms": items}
		return data, &Evidence{
			Tool:    "get_room_summary",
			Source:  "rooms",
			Summary: fmt.Sprintf("读取平台房间列表，共 %d 个", len(*roomsBasic)),
			Detail:  mustJSON(data),
		}, nil
	}

	scope := current
	if args.RoomID > 0 && args.RoomID != current.RoomID {
		return nil, nil, fmt.Errorf("房间 ID 与当前诊断不匹配")
	}
	worlds := make([]map[string]any, 0, len(scope.Worlds))
	runningWorlds := 0
	masterCount := 0
	for _, world := range scope.Worlds {
		up := scope.Game.WorldUpStatus(world.ID)
		if up {
			runningWorlds++
		}
		if world.IsMaster {
			masterCount++
		}
		perf := scope.Game.WorldPerformanceStatus(world.ID)
		worlds = append(worlds, map[string]any{
			"worldID":            world.ID,
			"worldName":          world.WorldName,
			"isMaster":           world.IsMaster,
			"serverPort":         world.ServerPort,
			"masterServerPort":   world.MasterServerPort,
			"authenticationPort": world.AuthenticationPort,
			"status":             up,
			"lastAliveTime":      world.LastAliveTime,
			"cpuPercent":         perf.CPU,
			"memoryMB":           perf.MemSize,
			"diskBytes":          perf.Disk,
		})
	}
	data := map[string]any{
		"scope":             ScopeRoom,
		"roomID":            scope.Room.ID,
		"gameName":          scope.Room.GameName,
		"gameMode":          scope.Room.GameMode,
		"pvp":               scope.Room.Pvp,
		"maxPlayer":         scope.Room.MaxPlayer,
		"status":            scope.Room.Status,
		"worldCount":        len(scope.Worlds),
		"runningWorldCount": runningWorlds,
		"masterCount":       masterCount,
		"worlds":            worlds,
		"setting": map[string]any{
			"tickRate":                 scope.Setting.TickRate,
			"startType":                scope.Setting.StartType,
			"backupEnable":             scope.Setting.BackupEnable,
			"restartEnable":            scope.Setting.RestartEnable,
			"keepaliveEnable":          scope.Setting.KeepaliveEnable,
			"scheduledStartStopEnable": scope.Setting.ScheduledStartStopEnable,
			"roomMetricsEnable":        scope.Setting.RoomMetricsEnable,
		},
	}
	return data, &Evidence{
		Tool:    "get_room_summary",
		Source:  fmt.Sprintf("room:%d", scope.Room.ID),
		Summary: fmt.Sprintf("读取房间 %s 的世界、端口与运行状态", scope.Room.GameName),
		Detail:  mustJSON(data),
	}, nil
}

func (s *Service) toolWorldStatus(current *Scope, raw json.RawMessage) (any, *Evidence, error) {
	var args struct {
		RoomID  int `json:"roomID"`
		WorldID int `json:"worldID"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, nil, fmt.Errorf("参数解析失败: %w", err)
	}
	scope, err := s.resolveScope(current, args.RoomID)
	if err != nil {
		return nil, nil, err
	}
	var world *models.World
	for i := range scope.Worlds {
		if scope.Worlds[i].ID == args.WorldID {
			world = &scope.Worlds[i]
			break
		}
	}
	if world == nil {
		return nil, nil, fmt.Errorf("世界不存在: %d", args.WorldID)
	}
	perf := scope.Game.WorldPerformanceStatus(world.ID)
	data := map[string]any{
		"roomID":             scope.Room.ID,
		"worldID":            world.ID,
		"worldName":          world.WorldName,
		"isMaster":           world.IsMaster,
		"status":             scope.Game.WorldUpStatus(world.ID),
		"cpuPercent":         perf.CPU,
		"memoryMB":           perf.MemSize,
		"memoryPercent":      perf.Mem,
		"diskBytes":          perf.Disk,
		"lastAliveTime":      world.LastAliveTime,
		"serverPort":         world.ServerPort,
		"masterServerPort":   world.MasterServerPort,
		"authenticationPort": world.AuthenticationPort,
	}
	return data, &Evidence{
		Tool:    "get_world_status",
		Source:  fmt.Sprintf("world:%d", world.ID),
		Summary: fmt.Sprintf("读取世界 %s 的运行状态与资源占用", world.WorldName),
		Detail:  mustJSON(data),
	}, nil
}

func (s *Service) resolveScope(current *Scope, roomID int) (*Scope, error) {
	if !current.Platform {
		if roomID != 0 && roomID != current.RoomID {
			return nil, fmt.Errorf("房间 ID 与当前诊断不匹配")
		}
		return current, nil
	}
	if roomID <= 0 {
		return nil, fmt.Errorf("平台诊断需要提供 roomID")
	}
	return s.loadRoomScope(current.Actor, roomID)
}

func (s *Service) toolLogs(session *Session, current *Scope, raw json.RawMessage) (any, *Evidence, error) {
	var args struct {
		Scope   string `json:"scope"`
		RoomID  int    `json:"roomID"`
		WorldID int    `json:"worldID"`
		Lines   int    `json:"lines"`
		Keyword string `json:"keyword"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, nil, fmt.Errorf("参数解析失败: %w", err)
	}
	if args.Scope == "" {
		args.Scope = "game"
	}
	if args.Lines <= 0 {
		args.Lines = 80
	}
	if args.Lines > maxLogLines {
		args.Lines = maxLogLines
	}

	var (
		lines     []string
		source    string
		truncated bool
	)
	switch args.Scope {
	case "game":
		scope, err := s.resolveScope(current, args.RoomID)
		if err != nil {
			return nil, nil, err
		}
		if args.WorldID > 0 {
			lines = scope.Game.LogContent("game", args.WorldID, args.Lines)
			source = fmt.Sprintf("room:%d/world:%d/server_log.txt", scope.RoomID, args.WorldID)
		} else {
			if len(scope.Worlds) == 0 {
				return map[string]any{"scope": "game", "source": "no_world", "lines": []string{}}, nil, nil
			}
			for _, world := range scope.Worlds {
				part := scope.Game.LogContent("game", world.ID, args.Lines/len(scope.Worlds)+1)
				lines = append(lines, part...)
				source = fmt.Sprintf("room:%d/game_logs", scope.RoomID)
				if len(lines) >= args.Lines {
					break
				}
			}
		}
	case "runtime", "steam", "access":
		if !session.Actor.IsAdmin() {
			return nil, nil, fmt.Errorf("没有读取系统日志的权限")
		}
		path := ""
		switch args.Scope {
		case "runtime":
			path = "logs/runtime.log"
		case "access":
			path = "logs/access.log"
		case "steam":
			if cache.OsType == constant.Darwin {
				path = constant.DarwinSteamLogPath
			} else {
				path = constant.SteamLogPath
			}
		}
		lines = utils.GetFileLastNLines(path, args.Lines)
		source = path
	default:
		return nil, nil, fmt.Errorf("不支持的日志类型: %s", args.Scope)
	}

	filtered := make([]string, 0, len(lines))
	keyword := strings.ToLower(strings.TrimSpace(args.Keyword))
	for _, line := range lines {
		line = redact(truncateRunes(line, maxLogLineRunes))
		if keyword != "" && !strings.Contains(strings.ToLower(line), keyword) {
			continue
		}
		filtered = append(filtered, line)
	}
	if len(filtered) > args.Lines {
		filtered = filtered[len(filtered)-args.Lines:]
	}

	detail := strings.Join(filtered, "\n")
	detail, truncated = capText(detail)
	data := map[string]any{
		"scope":     args.Scope,
		"source":    source,
		"lines":     filtered,
		"truncated": truncated,
	}
	return data, &Evidence{
		Tool:      "get_logs",
		Source:    source,
		Summary:   fmt.Sprintf("读取 %s 日志 %d 行", args.Scope, len(filtered)),
		Detail:    detail,
		Truncated: truncated,
	}, nil
}

func (s *Service) toolMods(current *Scope, raw json.RawMessage) (any, *Evidence, error) {
	var args struct {
		RoomID  int `json:"roomID"`
		WorldID int `json:"worldID"`
	}
	_ = json.Unmarshal(raw, &args)
	scope, err := s.resolveScope(current, args.RoomID)
	if err != nil {
		return nil, nil, err
	}
	worldID := args.WorldID
	if worldID == 0 {
		for _, world := range scope.Worlds {
			if world.IsMaster {
				worldID = world.ID
				break
			}
		}
		if worldID == 0 && len(scope.Worlds) > 0 {
			worldID = scope.Worlds[0].ID
		}
	}
	if worldID == 0 {
		return nil, nil, fmt.Errorf("房间没有世界")
	}
	mods, err := scope.Game.GetEnabledMods(worldID)
	if err != nil {
		return nil, nil, fmt.Errorf("读取模组列表失败: %w", err)
	}
	items := make([]map[string]any, 0, len(mods))
	for _, mod := range mods {
		items = append(items, map[string]any{"modID": mod.ID})
	}
	data := map[string]any{"roomID": scope.RoomID, "worldID": worldID, "count": len(items), "mods": items}
	return data, &Evidence{
		Tool:    "get_mods",
		Source:  fmt.Sprintf("room:%d/world:%d", scope.RoomID, worldID),
		Summary: fmt.Sprintf("读取世界 %d 的启用模组，共 %d 个", worldID, len(items)),
		Detail:  mustJSON(data),
	}, nil
}

func (s *Service) toolGameVersion() (any, *Evidence, error) {
	version := scheduler.GetDSTVersion()
	outdated := version.Local > 0 && version.Server > 0 && version.Local < version.Server
	data := map[string]any{
		"platformVersion":     constant.Version,
		"dstLocalVersion":     version.Local,
		"dstServerVersion":    version.Server,
		"gameVersionOutdated": outdated,
		"dstUpdating":         cache.DstUpdating,
	}
	return data, &Evidence{
		Tool:    "get_game_version",
		Source:  "dst/version.txt",
		Summary: fmt.Sprintf("读取 DST 版本：本地 %d，远端 %d", version.Local, version.Server),
		Detail:  mustJSON(data),
	}, nil
}

func (s *Service) toolScheduler(current *Scope, raw json.RawMessage) (any, *Evidence, error) {
	var args struct {
		RoomID int `json:"roomID"`
	}
	_ = json.Unmarshal(raw, &args)
	roomID := args.RoomID
	if roomID == 0 && !current.Platform {
		roomID = current.RoomID
	}
	jobs := []string{}
	if roomID > 0 {
		jobs = scheduler.GetJobsByRoomID(roomID)
	}
	data := map[string]any{
		"now":      time.Now().Format(time.RFC3339),
		"timezone": time.Local.String(),
		"roomID":   roomID,
		"jobs":     jobs,
	}
	return data, &Evidence{
		Tool:    "get_scheduler",
		Source:  "scheduler",
		Summary: fmt.Sprintf("读取时区与定时任务，房间 %d 共 %d 个任务", roomID, len(jobs)),
		Detail:  mustJSON(data),
	}, nil
}

func (s *Service) toolBackups(current *Scope, raw json.RawMessage) (any, *Evidence, error) {
	var args struct {
		RoomID int `json:"roomID"`
	}
	_ = json.Unmarshal(raw, &args)
	scope, err := s.resolveScope(current, args.RoomID)
	if err != nil {
		return nil, nil, err
	}
	backups, err := scope.Game.GetBackups()
	if err != nil {
		return nil, nil, fmt.Errorf("读取备份列表失败: %w", err)
	}
	if len(backups) > 10 {
		backups = backups[:10]
	}
	data := map[string]any{"roomID": scope.RoomID, "backups": backups}
	return data, &Evidence{
		Tool:    "get_backups",
		Source:  fmt.Sprintf("room:%d/backups", scope.RoomID),
		Summary: fmt.Sprintf("读取最近备份，共 %d 个", len(backups)),
		Detail:  mustJSON(data),
	}, nil
}

func (s *Service) toolProposeAction(session *Session, current *Scope, raw json.RawMessage) (any, *Evidence, error) {
	var args struct {
		ActionID    string         `json:"actionID"`
		Params      map[string]any `json:"params"`
		EvidenceIDs []string       `json:"evidenceIDs"`
		Reason      string         `json:"reason"`
	}
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, nil, fmt.Errorf("参数解析失败: %w", err)
	}
	action, err := s.buildAction(session, current, args.ActionID, args.Params, args.EvidenceIDs, args.Reason)
	if err != nil {
		return nil, nil, err
	}
	session.upsertAction(action)
	return action, nil, nil
}

func (s *Service) toolFinishDiagnosis(session *Session, current *Scope, raw json.RawMessage) (any, *Evidence, error) {
	var args finishArgs
	if err := json.Unmarshal(raw, &args); err != nil {
		return nil, nil, fmt.Errorf("参数解析失败: %w", err)
	}
	report, err := s.buildReport(session, current, &args)
	if err != nil {
		return nil, nil, err
	}
	session.setReport(report)
	return map[string]any{"accepted": true}, nil, nil
}

func (s *Service) scopedRoomID(current *Scope, params map[string]any) int {
	if current != nil && !current.Platform {
		return current.RoomID
	}
	if params == nil {
		return 0
	}
	switch value := params["roomID"].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case json.Number:
		id, _ := strconv.Atoi(value.String())
		return id
	default:
		return 0
	}
}

func (s *Service) actorFromSession(session *Session) service.Actor {
	return session.Actor
}

func (s *Service) debugTool(name string) {
	logger.Logger.Debugf("AI 诊断工具调用: %s", name)
}
