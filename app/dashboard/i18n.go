package dashboard

import (
	"dst-management-platform-api/i18n"
)

var message = i18n.NewExtendedI18n(map[string]i18n.Text{
	"startup game fail":     {ZH: "启动失败", EN: "Startup Fail"},
	"startup game success":  {ZH: "启动成功", EN: "Startup Success"},
	"shutdown game fail":    {ZH: "关闭失败", EN: "Shutdown Fail"},
	"shutdown game success": {ZH: "关闭成功", EN: "Shutdown Success"},
	"restart game fail":     {ZH: "重启失败", EN: "Restart Fail"},
	"restart game success":  {ZH: "重启成功", EN: "Restart Success"},
	"updating":              {ZH: "更新中，请耐心等待", EN: "Updating, please wait patiently"},
	"reset game fail":       {ZH: "重置失败", EN: "Reset Fail"},
	"reset game success":    {ZH: "重置成功", EN: "Reset Success"},
	"delete game fail":      {ZH: "清空世界失败", EN: "Delete Fail"},
	"delete game success":   {ZH: "清空世界成功", EN: "Delete Success"},
	"announce fail":         {ZH: "宣告失败", EN: "Announce Fail"},
	"announce success":      {ZH: "宣告成功", EN: "Announce Success"},
	"system msg fail":       {ZH: "通知失败", EN: "System Message Send Fail"},
	"system msg success":    {ZH: "通知成功", EN: "System Message Send Success"},
	"connection code fail":  {ZH: "直连代码获取失败", EN: "Get Connection Code Fail"},
	"check lobby fail":      {ZH: "检查世界失败", EN: "Check Lobby Fail"},
})
