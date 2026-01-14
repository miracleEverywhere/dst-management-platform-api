package user

import "dst-management-platform-api/database/dao"

type Handler struct {
	userDao   *dao.UserDAO
	systemDao *dao.SystemDAO
}

func NewHandler(userDao *dao.UserDAO) *Handler {
	return &Handler{
		userDao: userDao,
	}
}

type menuItem struct {
	ID        int        `json:"id"`
	Type      string     `json:"type"`
	Section   string     `json:"section"`
	Title     string     `json:"title"`
	To        string     `json:"to"`
	Component string     `json:"component"`
	Icon      string     `json:"icon"`
	Links     []menuItem `json:"links"`
}

var rooms = menuItem{
	ID:        1,
	Type:      "link",
	Section:   "",
	Title:     "rooms",
	To:        "/rooms",
	Component: "rooms/index",
	Icon:      "ri-instance-line",
	Links:     nil,
}

var dashboard = menuItem{
	ID:        2,
	Type:      "link",
	Section:   "",
	Title:     "dashboard",
	To:        "/dashboard",
	Component: "dashboard/index",
	Icon:      "ri-function-ai-line",
	Links:     nil,
}

var game = menuItem{
	ID:        3,
	Type:      "group",
	Section:   "",
	Title:     "game",
	To:        "/game",
	Component: "",
	Icon:      "ri-gamepad-line",
	Links: []menuItem{
		{
			ID:        301,
			Type:      "link",
			Section:   "",
			Title:     "gameBase",
			To:        "/game/base",
			Component: "game/base",
			Icon:      "ri-sword-line",
			Links:     nil,
		},
		{
			ID:        302,
			Type:      "link",
			Section:   "",
			Title:     "gameMod",
			To:        "/game/mod",
			Component: "game/mod",
			Icon:      "ri-rocket-2-line",
			Links:     nil,
		},
		{
			ID:        303,
			Type:      "link",
			Section:   "",
			Title:     "gamePlayer",
			To:        "/game/player",
			Component: "game/player",
			Icon:      "ri-ghost-line",
			Links:     nil,
		},
	},
}

var upload = menuItem{
	ID:        4,
	Type:      "link",
	Section:   "",
	Title:     "upload",
	To:        "/upload",
	Component: "upload/index",
	Icon:      "ri-contacts-book-upload-line",
	Links:     nil,
}

var install = menuItem{
	ID:        5,
	Type:      "link",
	Section:   "",
	Title:     "install",
	To:        "/install",
	Component: "install/index",
	Icon:      "ri-import-line",
	Links:     nil,
}

var tools = menuItem{
	ID:        6,
	Type:      "group",
	Section:   "",
	Title:     "tools",
	To:        "/tools",
	Component: "",
	Icon:      "ri-wrench-line",
	Links: []menuItem{
		{
			ID:        601,
			Type:      "link",
			Section:   "",
			Title:     "toolsBackup",
			To:        "/tools/backup",
			Component: "tools/backup",
			Icon:      "ri-save-2-line",
			Links:     nil,
		},
		{
			ID:        602,
			Type:      "link",
			Section:   "",
			Title:     "toolsAnnounce",
			To:        "/tools/announce",
			Component: "tools/announce",
			Icon:      "ri-chat-smile-ai-3-line",
			Links:     nil,
		},
		{
			ID:        603,
			Type:      "link",
			Section:   "",
			Title:     "toolsMap",
			To:        "/tools/map",
			Component: "tools/map",
			Icon:      "ri-road-map-line",
			Links:     nil,
		},
		{
			ID:        604,
			Type:      "link",
			Section:   "",
			Title:     "toolsToken",
			To:        "/tools/token",
			Component: "tools/token",
			Icon:      "ri-coupon-line",
			Links:     nil,
		},
		{
			ID:        605,
			Type:      "link",
			Section:   "",
			Title:     "toolsSnapshot",
			To:        "/tools/snapshot",
			Component: "tools/snapshot",
			Icon:      "ri-flip-horizontal-line",
			Links:     nil,
		},
	},
}

var logs = menuItem{
	ID:        7,
	Type:      "group",
	Section:   "",
	Title:     "logs",
	To:        "/logs",
	Component: "",
	Icon:      "ri-blogger-line",
	Links: []menuItem{
		{
			ID:        701,
			Type:      "link",
			Section:   "",
			Title:     "logsGame",
			To:        "/logs/game",
			Component: "logs/game",
			Icon:      "ri-game-line",
			Links:     nil,
		},
		{
			ID:        702,
			Type:      "link",
			Section:   "",
			Title:     "logsChat",
			To:        "/logs/chat",
			Component: "logs/chat",
			Icon:      "ri-chat-smile-3-line",
			Links:     nil,
		},
		{
			ID:        703,
			Type:      "link",
			Section:   "",
			Title:     "logsDownload",
			To:        "/logs/download",
			Component: "logs/download",
			Icon:      "ri-download-2-line",
			Links:     nil,
		},
		{
			ID:        704,
			Type:      "link",
			Section:   "",
			Title:     "logsSteam",
			To:        "/logs/steam",
			Component: "logs/steam",
			Icon:      "ri-steam-line",
			Links:     nil,
		},
		{
			ID:        705,
			Type:      "link",
			Section:   "",
			Title:     "logsAccess",
			To:        "/logs/access",
			Component: "logs/access",
			Icon:      "ri-code-box-line",
			Links:     nil,
		},
		{
			ID:        706,
			Type:      "link",
			Section:   "",
			Title:     "logsRuntime",
			To:        "/logs/runtime",
			Component: "logs/runtime",
			Icon:      "ri-terminal-box-line",
			Links:     nil,
		},
		{
			ID:        707,
			Type:      "link",
			Section:   "",
			Title:     "logsClean",
			To:        "/logs/clean",
			Component: "logs/clean",
			Icon:      "ri-file-shred-line",
			Links:     nil,
		},
	},
}

var platform = menuItem{
	ID:        8,
	Type:      "link",
	Section:   "",
	Title:     "platform",
	To:        "/platform",
	Component: "platform/index",
	Icon:      "ri-vip-crown-2-line",
	Links:     nil,
}

type Partition struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"pageSize" form:"pageSize"`
}
