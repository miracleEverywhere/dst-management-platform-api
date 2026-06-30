package webhook

// 事件类型常量
const (
	EventRoomCreated          = "room_created"
	EventRoomDeleted          = "room_deleted"
	EventRoomSettingsUpdated  = "room_settings_updated"
	EventRoomActivated        = "room_activated"
	EventRoomDeactivated      = "room_deactivated"
	EventGameBackup           = "game_backup"
	EventGameReset            = "game_reset"
	EventGameStart            = "game_start"
	EventGameStop             = "game_stop"
	EventGameUpdate           = "game_update"
	EventKeepaliveTriggered   = "keepalive_triggered"
	EventPlayerManage         = "player_manage"
	EventGlobalSettingUpdated = "global_setting_updated"
	EventWebsocketConnected   = "websocket_connected"
	EventOnlinePlayerUpdated  = "online_player_updated"
)

// AllEventTypes 可选事件类型列表，供前端渲染 webhook 配置表单
var AllEventTypes = []EventInfo{
	{Type: EventRoomCreated, ZH: "房间创建", EN: "Room Created"},
	{Type: EventRoomDeleted, ZH: "房间删除", EN: "Room Deleted"},
	{Type: EventRoomSettingsUpdated, ZH: "房间修改", EN: "Room Settings Updated"},
	{Type: EventRoomActivated, ZH: "房间激活", EN: "Room Activated"},
	{Type: EventRoomDeactivated, ZH: "房间关闭", EN: "Room Deactivated"},
	{Type: EventGameBackup, ZH: "游戏备份", EN: "Room Backup"},
	{Type: EventGameReset, ZH: "游戏重置", EN: "Room Reset"},
	{Type: EventGameStart, ZH: "游戏启动", EN: "Room Start"},
	{Type: EventGameStop, ZH: "游戏关闭", EN: "Room Stop"},
	{Type: EventGameUpdate, ZH: "游戏更新", EN: "Game Update"},
	{Type: EventKeepaliveTriggered, ZH: "自动保活", EN: "Keepalive Triggered"},
	{Type: EventPlayerManage, ZH: "玩家管理", EN: "Player Manage"},
	{Type: EventGlobalSettingUpdated, ZH: "平台设置修改", EN: "Platform Settings Updated"},
	{Type: EventWebsocketConnected, ZH: "虚拟终端连接", EN: "Websocket Connected"},
	{Type: EventOnlinePlayerUpdated, ZH: "在线玩家变化", EN: "Online Player Updated"},
}

type EventInfo struct {
	Type string `json:"type"`
	ZH   string `json:"zh"`
	EN   string `json:"en"`
}

// WebhookItem 房间级 webhook 配置项（存储在 RoomSetting.WebhookSetting）
type WebhookItem struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	URL     string   `json:"url"`
	Events  []string `json:"events"`
	Enabled bool     `json:"enabled"`
	Secret  string   `json:"secret,omitempty"`
}

// GlobalWebhookItem 全局级 webhook 配置项（存储在 GlobalSetting.WebhookSetting）
type GlobalWebhookItem struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	URL     string   `json:"url"`
	Events  []string `json:"events"`
	Enabled bool     `json:"enabled"`
	Secret  string   `json:"secret,omitempty"`
	RoomIDs []int    `json:"roomIds"` // 空数组 = 所有房间
}

// Payload 发送给 webhook 接收端的 JSON 结构
type Payload struct {
	Event     EventInfo   `json:"event"`
	Timestamp int64       `json:"timestamp"`
	RoomID    int         `json:"roomId,omitempty"`
	RoomName  string      `json:"roomName,omitempty"`
	Name      string      `json:"name,omitempty"`
	Data      interface{} `json:"data,omitempty"`
}
