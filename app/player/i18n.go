package player

import (
	"dst-management-platform-api/i18n"
)

var message = i18n.NewExtendedI18n(map[string]i18n.Text{
	"chat message fail": {ZH: "玩家聊天信息获取失败", EN: "get chat message fail"},
})
