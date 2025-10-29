package user

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

var gameSetting = menuItem{
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
	ID:        2,
	Type:      "link",
	Section:   "",
	Title:     "upload",
	To:        "/upload",
	Component: "upload/index",
	Icon:      "ri-contacts-book-upload-line",
	Links:     nil,
}
