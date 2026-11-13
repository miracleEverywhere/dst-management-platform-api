package user

import (
	"dst-management-platform-api/service"
)

// Menu 当前角色可见的菜单
func (s *Service) Menu(actor service.Actor) service.Result {
	var data []menuItem

	if actor.IsAdmin() {
		data = []menuItem{
			rooms,
			dashboard,
			game,
			tools,
			logs,
			upload,
			install,
			platform,
		}
	} else {
		toolsLess := tools
		toolsLess.Links = []menuItem{
			tools.Links[0],
			tools.Links[1],
			tools.Links[2],
			tools.Links[4],
			tools.Links[5],
			tools.Links[6],
		}
		logsLess := logs
		logsLess.Links = []menuItem{
			logs.Links[0],
			logs.Links[1],
			logs.Links[2],
		}

		data = []menuItem{
			rooms,
			dashboard,
			game,
			toolsLess,
			logsLess,
			upload,
		}
	}

	return service.Success(data)
}

// ---- 菜单数据 ----
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
