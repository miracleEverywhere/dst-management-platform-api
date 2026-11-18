package i18n

// Platform 是平台模块的词条表：基础文案 + 本模块词条。
var Platform = NewExtendedI18n(map[string]Text{
	"get os info fail":     {ZH: "获取系统信息失败", EN: "Get OS Info Fail"},
	"get screens fail":     {ZH: "获取Screens失败", EN: "Get Screens Fail"},
	"kill screen fail":     {ZH: "关闭Screens失败", EN: "Kill Screens Fail"},
	"kill screen success":  {ZH: "关闭Screens成功", EN: "Kill Screens Success"},
	"webhook test fail":    {ZH: "Webhook 测试失败: %s", EN: "Webhook Test Failed: %s"},
	"webhook test success": {ZH: "Webhook 测试成功", EN: "Webhook Test Success"},
	"install fail":         {ZH: "安装失败: %s", EN: "Install Failed: %s"},
	"install success":      {ZH: "安装成功", EN: "Install Success"},
	"uninstall fail":       {ZH: "卸载失败", EN: "Uninstall Fail"},
	"game task busy":       {ZH: "已有游戏安装或更新任务正在进行，请等待其结束", EN: "A game install or update task is already running, please wait for it to finish"},
	"game task start fail": {ZH: "游戏安装或更新任务启动失败", EN: "Failed to start the game install or update task"},
	"game task not running": {
		ZH: "当前没有进行中的游戏安装或更新任务",
		EN: "No game install or update task is running",
	},
})
