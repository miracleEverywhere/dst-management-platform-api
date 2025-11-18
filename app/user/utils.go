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
	Component: "tools/backup",
	Icon:      "ri-wrench-line",
	Links: []menuItem{
		{
			ID:        601,
			Type:      "link",
			Section:   "",
			Title:     "toolsBackup",
			To:        "/tools/backup",
			Component: "tools/backup",
			Icon:      "ri-save-line",
			Links:     nil,
		},
	},
}

var platform = menuItem{
	ID:        7,
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
