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
	"self update in container": {
		ZH: "容器部署不支持在网页上升级平台，请更新镜像后重建容器",
		EN: "Self update is not available in a container, update the image and recreate the container",
	},
	"self update unsupported": {
		ZH: "当前运行环境不支持在网页上升级平台，请使用 run.sh 更新",
		EN: "Self update is not supported in this environment, please update with run.sh",
	},
	"self update not writable": {
		ZH: "程序所在目录不可写，无法替换二进制",
		EN: "The directory of the executable is not writable, cannot replace the binary",
	},
	"update check fail": {
		ZH: "检查更新失败，请确认服务器能访问 GitHub",
		EN: "Failed to check for updates, please make sure GitHub is reachable",
	},
	"update asset invalid": {
		ZH: "没有找到对应的安装包或缺少校验值，已中止更新",
		EN: "The matching package or its checksum is missing, update aborted",
	},
	"update download fail": {
		ZH: "下载安装包失败，可更换加速节点后重试",
		EN: "Failed to download the package, try another acceleration node",
	},
	"update digest mismatch": {
		ZH: "安装包校验失败，可能与加速节点返回的内容不一致，请更换节点重试",
		EN: "Package checksum mismatch, the acceleration node may return modified content, try another node",
	},
	"update extract fail": {
		ZH: "安装包解压失败或包内缺少可执行文件",
		EN: "Failed to extract the package or the executable is missing",
	},
	"update preflight fail": {
		ZH: "新版本预检失败，安装包与目标版本不一致，已中止更新",
		EN: "Preflight of the new binary failed, the package does not match the target version, update aborted",
	},
	"update replace fail": {
		ZH: "替换程序文件或重启失败，详情见 dmp_files/dmp_update 下的记录",
		EN: "Failed to replace the binary or restart, see dmp_files/dmp_update for details",
	},
	"update busy": {
		ZH: "已有平台更新任务正在进行，请等待其结束",
		EN: "A platform update task is already running, please wait for it to finish",
	},
	"update not running": {
		ZH: "当前没有进行中的平台更新任务",
		EN: "No platform update task is running",
	},
	"update backup fail": {
		ZH: "更新前备份数据库失败，已中止更新",
		EN: "Failed to back up the database, update aborted",
	},
	"update not needed": {
		ZH: "当前版本不低于最新版本，无需更新",
		EN: "The current version is not older than the latest release, no update needed",
	},
	"update started": {
		ZH: "已开始更新平台",
		EN: "Platform update started",
	},
	"update available": {
		ZH: "发现新版本",
		EN: "A new version is available",
	},
	"update up to date": {
		ZH: "已是最新版本",
		EN: "Already up to date",
	},
	"update check failed": {
		ZH: "检查更新失败，请确认服务器能访问 GitHub",
		EN: "Failed to check for updates, please make sure GitHub is reachable",
	},
	"update local newer": {
		ZH: "当前为开发版本，高于线上最新版本",
		EN: "The running build is newer than the latest release",
	},
})
