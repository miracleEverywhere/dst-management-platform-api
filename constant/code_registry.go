package constant

// Level 描述错误码的性质，供前端决定提示方式并用于文档分类。
// 注意：Level 不参与响应，响应里仍然只有 code/message/data。

type Level int

const (
	// LevelWarn 业务错误，用户通常可以自行修正
	LevelWarn Level = iota
	// LevelAuth 登录态或权限问题，需要重新登录或提权
	LevelAuth
	// LevelFatal 服务端或外部依赖故障，需要查看运行日志
	LevelFatal
)

// Entry 一个业务错误码的元数据。
//
//	Code    错误码，唯一标识一个错误原因
//	Const   对应的 Go 常量名，便于文档与排查时定位
//	Module  所属模块
//	TextKey 对应的 i18n 词条 key，文案由 i18n 包翻译（多个码可以共用一个词条）
//	Hint    排查建议，用于生成 docs/error-codes.md
//	Level   错误性质

type Entry struct {
	Code    int
	Const   string
	Module  string
	TextKey string
	Hint    string
	Level   Level
}

// Registry 全部业务错误码，按码升序排列。
var Registry = []Entry{
	// framework
	{Code: 1, Const: "CodeFrameworkBadRequest", Module: "framework", TextKey: "bad request", Level: LevelWarn, Hint: "请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。"},
	{Code: 2, Const: "CodeFrameworkNoPermission", Module: "framework", TextKey: "permission needed", Level: LevelAuth, Hint: "当前账号没有该房间或该操作的管理员权限。让管理员在用户管理里授予对应房间权限。"},
	{Code: 420, Const: "CodeAuthTokenInvalid", Module: "framework", TextKey: "token fail", Level: LevelAuth, Hint: "登录态缺失、过期或签名不匹配。重新登录；若反复出现，检查服务端 jwt-secret 是否被重置。"},
	{Code: 429, Const: "CodeTooManyRequests", Module: "framework", TextKey: "too many requests", Level: LevelWarn, Hint: "同一账号短时间内请求过于频繁，触发了登录限流。等待约 1 秒后重试。"},

	// user
	{Code: 1001, Const: "CodeUserBadRequest", Module: "user", TextKey: "bad request", Level: LevelWarn, Hint: "请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。"},
	{Code: 1021, Const: "CodeUserNoPermission", Module: "user", TextKey: "permission needed", Level: LevelAuth, Hint: "当前账号没有该房间或该操作的管理员权限。让管理员在用户管理里授予对应房间权限。"},
	{Code: 1041, Const: "CodeUserNotExist", Module: "user", TextKey: "user not exist", Level: LevelWarn, Hint: "用户不存在或已被删除。刷新用户列表后重试。"},
	{Code: 1061, Const: "CodeUserExist", Module: "user", TextKey: "user exist", Level: LevelWarn, Hint: "用户名已被占用，换一个用户名。"},
	{Code: 1062, Const: "CodeUserWrongPassword", Module: "user", TextKey: "wrong password", Level: LevelWarn, Hint: "密码错误。确认大小写与输入法；多次失败会被限流。"},
	{Code: 1063, Const: "CodeUserDisabled", Module: "user", TextKey: "disabled", Level: LevelWarn, Hint: "账号已被禁用，需要管理员在用户管理里解除禁用。"},
	{Code: 1064, Const: "CodeUserDeleteAllForbidden", Module: "user", TextKey: "delete all users", Level: LevelWarn, Hint: "这是最后一个管理员账号，系统拒绝删除以免把自己锁在门外。先创建另一个管理员再删除。"},
	{Code: 1065, Const: "CodeUserUpdateFail", Module: "user", TextKey: "update fail", Level: LevelWarn, Hint: "更新写入失败。确认提交内容合法，并查看 logs/runtime.log 中的具体错误。"},
	{Code: 1091, Const: "CodeUserDBError", Module: "user", TextKey: "database error", Level: LevelFatal, Hint: "数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。"},
	{Code: 1092, Const: "CodeUserCreateFail", Module: "user", TextKey: "create fail", Level: LevelWarn, Hint: "创建失败，常见原因是必填字段缺失或数据库写入异常。查看响应 message 与运行日志。"},
	{Code: 1093, Const: "CodeUserLoginFail", Module: "user", TextKey: "login fail", Level: LevelWarn, Hint: "登录流程失败，常见于签发 token 或写入登录信息时出错。查看 logs/runtime.log。"},
	{Code: 1094, Const: "CodeUserRevokeFail", Module: "user", TextKey: "revoke fail", Level: LevelWarn, Hint: "撤销 token 失败，通常是数据库写入异常。查看 logs/runtime.log。"},
	{Code: 1095, Const: "CodeUserDeleteFail", Module: "user", TextKey: "delete fail", Level: LevelWarn, Hint: "删除失败，可能是资源仍被占用（进程未退出、文件被锁）或数据库写入异常。"},

	// room
	{Code: 2001, Const: "CodeRoomBadRequest", Module: "room", TextKey: "bad request", Level: LevelWarn, Hint: "请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。"},
	{Code: 2002, Const: "CodeRoomInvalidURL", Module: "room", TextKey: "invalid url", Level: LevelWarn, Hint: "提交的 URL 格式不合法（Webhook 等）。必须以 http:// 或 https:// 开头且可解析。"},
	{Code: 2003, Const: "CodeRoomInvalidWorldName", Module: "room", TextKey: "invalid world name", Level: LevelWarn, Hint: "世界名会被拼进命令和文件路径，只允许字母、数字、下划线、中划线和点，且不能以点或中划线开头。修改世界名后重试。"},
	{Code: 2004, Const: "CodeRoomDuplicateWorldName", Module: "room", TextKey: "duplicate world name", Level: LevelWarn, Hint: "同一房间内的世界名必须唯一，否则多个世界会写入同一个存档目录。改成不同的名字。"},
	{Code: 2005, Const: "CodeRoomFavoriteInvalidGameMode", Module: "room", TextKey: "favorite invalid game mode", Level: LevelWarn, Hint: "收藏里的游戏模式不是合法取值（可能来自旧版本数据）。删除该收藏后重新保存。"},
	{Code: 2006, Const: "CodeRoomFavoriteInvalidWebhook", Module: "room", TextKey: "favorite invalid webhook", Level: LevelWarn, Hint: "收藏里的 Webhook 配置不是合法 JSON，或 URL 非法。修正后重新收藏。"},
	{Code: 2021, Const: "CodeRoomNoPermission", Module: "room", TextKey: "permission needed", Level: LevelAuth, Hint: "当前账号没有该房间或该操作的管理员权限。让管理员在用户管理里授予对应房间权限。"},
	{Code: 2041, Const: "CodeRoomNotFound", Module: "room", TextKey: "room not found", Level: LevelWarn, Hint: "房间不存在或已被删除。刷新房间列表后重试。"},
	{Code: 2042, Const: "CodeRoomFavoriteNotFound", Module: "room", TextKey: "favorite not found", Level: LevelWarn, Hint: "收藏不存在、已被删除，或不属于当前账号。刷新收藏列表后重试。"},
	{Code: 2061, Const: "CodeRoomPortConflict", Module: "room", TextKey: "port conflict", Level: LevelWarn, Hint: "端口与已有房间/世界冲突。换一个端口，或先停用占用该端口的房间。"},
	{Code: 2062, Const: "CodeRoomWriteFileFail", Module: "room", TextKey: "write file fail", Level: LevelFatal, Hint: "配置文件写入磁盘失败。检查 DST 集群目录权限与磁盘空间，以及是否有异常的世界名。"},
	{Code: 2063, Const: "CodeRoomFavoriteNoWorlds", Module: "room", TextKey: "favorite no worlds", Level: LevelWarn, Hint: "房间下没有任何世界，无法生成收藏快照。先给房间添加至少一个世界。"},
	{Code: 2064, Const: "CodeRoomFavoriteDataInvalid", Module: "room", TextKey: "favorite data invalid", Level: LevelWarn, Hint: "收藏数据损坏或结构不兼容（旧版本数据）。删除该收藏重新保存。"},
	{Code: 2065, Const: "CodeRoomFavoriteNameRequired", Module: "room", TextKey: "favorite room name required", Level: LevelWarn, Hint: "未提供房间名，且收藏里也没有可用名称。在弹窗里填写一个房间名。"},
	{Code: 2066, Const: "CodeRoomActivateFail", Module: "room", TextKey: "activate fail", Level: LevelWarn, Hint: "启用房间失败，常见于定时通知配置 JSON 解析失败。检查房间设置里的通知配置。"},
	{Code: 2067, Const: "CodeRoomUpdateFail", Module: "room", TextKey: "update fail", Level: LevelWarn, Hint: "更新写入失败。确认提交内容合法，并查看 logs/runtime.log 中的具体错误。"},
	{Code: 2068, Const: "CodeRoomDeleteFail", Module: "room", TextKey: "delete fail", Level: LevelWarn, Hint: "删除失败，可能是资源仍被占用（进程未退出、文件被锁）或数据库写入异常。"},
	{Code: 2069, Const: "CodeRoomWebhookTestFail", Module: "room", TextKey: "webhook test fail", Level: LevelWarn, Hint: "Webhook 测试请求发送失败。检查目标地址可达性、证书与签名密钥。"},
	{Code: 2070, Const: "CodeRoomImportNoWorlds", Module: "room", TextKey: "no available worlds found", Level: LevelWarn, Hint: "上传的存档里没有解析出任何世界目录。确认 zip 内是完整的 Cluster 目录结构。"},
	{Code: 2071, Const: "CodeRoomImportWorldCountMismatch", Module: "room", TextKey: "number of worlds does not match", Level: LevelWarn, Hint: "上传存档的世界数量与当前房间不一致，无法覆盖导入。删除多余世界或改用新建房间导入。"},
	{Code: 2072, Const: "CodeRoomUploadSaveFail", Module: "room", TextKey: "upload save fail", Level: LevelWarn, Hint: "上传文件落盘或解压失败。检查临时目录权限与磁盘空间，确认上传的是合法 zip。"},
	{Code: 2073, Const: "CodeRoomUnzipFail", Module: "room", TextKey: "unzip fail", Level: LevelWarn, Hint: "zip 解压失败。确认文件完整、未加密、未损坏。"},
	{Code: 2074, Const: "CodeRoomClusterDirNotFound", Module: "room", TextKey: "find cluster home fail", Level: LevelWarn, Hint: "解压后找不到包含 cluster.ini 的目录。确认 zip 的目录层级正确。"},
	{Code: 2075, Const: "CodeRoomClusterIniNotFound", Module: "room", TextKey: "cluster.ini file not found", Level: LevelWarn, Hint: "存档根目录缺少 cluster.ini。确认这是 DST 的 Cluster 存档。"},
	{Code: 2076, Const: "CodeRoomClusterIniReadFail", Module: "room", TextKey: "read cluster.ini file fail", Level: LevelWarn, Hint: "cluster.ini 无法读取或编码异常。检查文件权限与内容格式。"},
	{Code: 2077, Const: "CodeRoomClusterNameNotFound", Module: "room", TextKey: "cluster.ini cluster_name not found", Level: LevelWarn, Hint: "cluster.ini 里没有 cluster_name。补上房间名后重新导入。"},
	{Code: 2078, Const: "CodeRoomGameModeNotFound", Module: "room", TextKey: "cluster.ini game_mode not found", Level: LevelWarn, Hint: "cluster.ini 里没有 game_mode。补上游戏模式后重新导入。"},
	{Code: 2079, Const: "CodeRoomGameModeInvalid", Module: "room", TextKey: "cluster.ini game_mode invalid", Level: LevelWarn, Hint: "cluster.ini 的 game_mode 不是受支持的取值（防 XSS 校验拦截）。改成合法模式。"},
	{Code: 2080, Const: "CodeRoomServerIniNotFound", Module: "room", TextKey: "server.ini file not found", Level: LevelWarn, Hint: "世界目录缺少 server.ini。确认每个世界目录结构完整。"},
	{Code: 2081, Const: "CodeRoomServerIniReadFail", Module: "room", TextKey: "read server.ini file fail", Level: LevelWarn, Hint: "server.ini 无法读取。检查文件权限与内容格式。"},
	{Code: 2082, Const: "CodeRoomIsMasterNotFound", Module: "room", TextKey: "server.ini is_master not found", Level: LevelWarn, Hint: "server.ini 里没有 is_master。补上主从标记后重新导入。"},
	{Code: 2083, Const: "CodeRoomIsMasterReadFail", Module: "room", TextKey: "read is_master from server.ini fail", Level: LevelWarn, Hint: "server.ini 的 is_master 不是合法布尔值。改成 true/false。"},
	{Code: 2084, Const: "CodeRoomLevelDataNotFound", Module: "room", TextKey: "level data not found", Level: LevelWarn, Hint: "世界目录缺少 leveldataoverride.lua / worldgenoverride.lua。补上世界配置后重新导入。"},
	{Code: 2085, Const: "CodeRoomWorldsDirInvalid", Module: "room", TextKey: "get worlds path fail", Level: LevelWarn, Hint: "世界目录名非法或目录读取失败。目录名只允许字母数字和斜杠。"},
	{Code: 2086, Const: "CodeRoomDuplicateWorldNameInSave", Module: "room", TextKey: "duplicate world name in save", Level: LevelWarn, Hint: "存档里存在重复的世界名，导入会互相覆盖存档。改名后重新导入。"},
	{Code: 2087, Const: "CodeRoomInvalidWorldNameInSave", Module: "room", TextKey: "invalid world name in save", Level: LevelWarn, Hint: "存档里存在非法世界名（会被拼进命令和路径）。改名后重新导入。"},
	{Code: 2091, Const: "CodeRoomDBError", Module: "room", TextKey: "database error", Level: LevelFatal, Hint: "数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。"},

	// mod
	{Code: 3001, Const: "CodeModBadRequest", Module: "mod", TextKey: "bad request", Level: LevelWarn, Hint: "请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。"},
	{Code: 3021, Const: "CodeModNoPermission", Module: "mod", TextKey: "permission needed", Level: LevelAuth, Hint: "当前账号没有该房间或该操作的管理员权限。让管理员在用户管理里授予对应房间权限。"},
	{Code: 3061, Const: "CodeModSearchFail", Module: "mod", TextKey: "search fail", Level: LevelWarn, Hint: "从 Steam 创意工坊查询模组失败。检查服务器到 Steam API 的网络与 Steam API Key。"},
	{Code: 3062, Const: "CodeModConfigOptionsError", Module: "mod", TextKey: "mod configuration options error", Level: LevelWarn, Hint: "读取模组配置项失败。确认模组已下载完整（modoverrides.lua 可解析）。"},
	{Code: 3063, Const: "CodeModConfigValuesError", Module: "mod", TextKey: "mod configuration values error", Level: LevelWarn, Hint: "读取模组配置当前值失败。同上，检查模组文件是否完整。"},
	{Code: 3064, Const: "CodeModModifyConfigError", Module: "mod", TextKey: "modify mod configuration values error", Level: LevelWarn, Hint: "写入模组配置失败。检查模组目录权限与磁盘空间。"},
	{Code: 3065, Const: "CodeModEnableFail", Module: "mod", TextKey: "mod enable fail", Level: LevelWarn, Hint: "启用模组失败。确认模组已下载完成且 ID 正确。"},
	{Code: 3066, Const: "CodeModDisableFail", Module: "mod", TextKey: "mod disable fail", Level: LevelWarn, Hint: "禁用模组失败。查看 logs/runtime.log 中的具体错误。"},
	{Code: 3067, Const: "CodeModGetEnabledFail", Module: "mod", TextKey: "get enabled mod fail", Level: LevelWarn, Hint: "读取已启用模组列表失败。确认 modoverrides.lua 存在且格式正确。"},
	{Code: 3068, Const: "CodeModDeleteFail", Module: "mod", TextKey: "delete fail", Level: LevelWarn, Hint: "删除失败，可能是资源仍被占用（进程未退出、文件被锁）或数据库写入异常。"},
	{Code: 3069, Const: "CodeModDownloadFail", Module: "mod", TextKey: "download fail", Level: LevelWarn, Hint: "模组下载失败或下载大小与预期不符。检查网络、下载地址与模组文件是否完整，查看 logs/runtime.log 中的具体错误。"},
	{Code: 3091, Const: "CodeModDBError", Module: "mod", TextKey: "database error", Level: LevelFatal, Hint: "数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。"},
	{Code: 3092, Const: "CodeModServerError", Module: "mod", TextKey: "server error", Level: LevelFatal, Hint: "服务端内部异常，通常是被其他请求触发的空指针或未初始化状态。查看 logs/runtime.log。"},

	// dashboard
	{Code: 4001, Const: "CodeDashboardBadRequest", Module: "dashboard", TextKey: "bad request", Level: LevelWarn, Hint: "请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。"},
	{Code: 4021, Const: "CodeDashboardNoPermission", Module: "dashboard", TextKey: "permission needed", Level: LevelAuth, Hint: "当前账号没有该房间或该操作的管理员权限。让管理员在用户管理里授予对应房间权限。"},
	{Code: 4061, Const: "CodeDashboardExecFail", Module: "dashboard", TextKey: "exec fail", Level: LevelWarn, Hint: "执行游戏操作失败。确认对应房间进程存在、screen 会话可用。"},
	{Code: 4062, Const: "CodeDashboardStartupFail", Module: "dashboard", TextKey: "startup game fail", Level: LevelWarn, Hint: "启动游戏失败。检查端口占用、DST 安装完整性、screen 是否可用。"},
	{Code: 4063, Const: "CodeDashboardShutdownFail", Module: "dashboard", TextKey: "shutdown game fail", Level: LevelWarn, Hint: "关闭游戏失败。进程可能已退出或 screen 会话丢失。"},
	{Code: 4064, Const: "CodeDashboardRestartFail", Module: "dashboard", TextKey: "restart game fail", Level: LevelWarn, Hint: "重启游戏失败。查看 logs/runtime.log 与 DST 自身日志。"},
	{Code: 4065, Const: "CodeDashboardResetFail", Module: "dashboard", TextKey: "reset game fail", Level: LevelWarn, Hint: "重置存档失败。确认存档目录存在且可写。"},
	{Code: 4066, Const: "CodeDashboardDeleteGameFail", Module: "dashboard", TextKey: "delete game fail", Level: LevelWarn, Hint: "删除游戏相关文件失败。确认没有进程占用这些文件。"},
	{Code: 4067, Const: "CodeDashboardAnnounceFail", Module: "dashboard", TextKey: "announce fail", Level: LevelWarn, Hint: "定时通知创建失败。检查通知内容的间隔与配置格式。"},
	{Code: 4068, Const: "CodeDashboardSystemMsgFail", Module: "dashboard", TextKey: "system msg fail", Level: LevelWarn, Hint: "发送系统消息失败。确认世界进程正在运行。"},
	{Code: 4069, Const: "CodeDashboardConnectionCodeFail", Module: "dashboard", TextKey: "connection code fail", Level: LevelWarn, Hint: "更新直连信息（IP/端口）失败。检查端口是否被占用、IP 是否可达。"},
	{Code: 4070, Const: "CodeDashboardCheckLobbyFail", Module: "dashboard", TextKey: "check lobby fail", Level: LevelWarn, Hint: "查询 Klei 大厅失败。检查服务器出境网络与 Klei 接口可用性。"},
	{Code: 4071, Const: "CodeDashboardUpdateFail", Module: "dashboard", TextKey: "update fail", Level: LevelWarn, Hint: "更新写入失败。确认提交内容合法，并查看 logs/runtime.log 中的具体错误。"},
	{Code: 4091, Const: "CodeDashboardDBError", Module: "dashboard", TextKey: "database error", Level: LevelFatal, Hint: "数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。"},

	// platform
	{Code: 5001, Const: "CodePlatformBadRequest", Module: "platform", TextKey: "bad request", Level: LevelWarn, Hint: "请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。"},
	{Code: 5002, Const: "CodePlatformInvalidURL", Module: "platform", TextKey: "invalid url", Level: LevelWarn, Hint: "提交的 URL 格式不合法（Webhook 等）。必须以 http:// 或 https:// 开头且可解析。"},
	{Code: 5061, Const: "CodePlatformInstallFail", Module: "platform", TextKey: "install fail", Level: LevelWarn, Hint: "插件安装失败。检查依赖（stex 等）是否安装、目标目录权限与网络。"},
	{Code: 5062, Const: "CodePlatformUninstallFail", Module: "platform", TextKey: "uninstall fail", Level: LevelWarn, Hint: "插件卸载失败。确认插件目录存在且未被占用。"},
	{Code: 5063, Const: "CodePlatformUpdateFail", Module: "platform", TextKey: "update fail", Level: LevelWarn, Hint: "更新写入失败。确认提交内容合法，并查看 logs/runtime.log 中的具体错误。"},
	{Code: 5064, Const: "CodePlatformOSInfoFail", Module: "platform", TextKey: "get os info fail", Level: LevelWarn, Hint: "读取系统信息失败。检查 /proc 或系统命令是否可用（容器环境常见）。"},
	{Code: 5065, Const: "CodePlatformScreensFail", Module: "platform", TextKey: "get screens fail", Level: LevelWarn, Hint: "列出 screen 会话失败。确认已安装 screen 且当前用户可执行。"},
	{Code: 5066, Const: "CodePlatformKillScreenFail", Module: "platform", TextKey: "kill screen fail", Level: LevelWarn, Hint: "结束 screen 会话失败。确认会话存在且属于当前用户。"},
	{Code: 5067, Const: "CodePlatformWebhookTestFail", Module: "platform", TextKey: "webhook test fail", Level: LevelWarn, Hint: "Webhook 测试请求发送失败。检查目标地址可达性、证书与签名密钥。"},
	{Code: 5091, Const: "CodePlatformDBError", Module: "platform", TextKey: "database error", Level: LevelFatal, Hint: "数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。"},

	// logs
	{Code: 6001, Const: "CodeLogsBadRequest", Module: "logs", TextKey: "bad request", Level: LevelWarn, Hint: "请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。"},
	{Code: 6021, Const: "CodeLogsNoPermission", Module: "logs", TextKey: "permission needed", Level: LevelAuth, Hint: "当前账号没有该房间或该操作的管理员权限。让管理员在用户管理里授予对应房间权限。"},
	{Code: 6061, Const: "CodeLogsDownloadFail", Module: "logs", TextKey: "download fail", Level: LevelWarn, Hint: "日志打包或读取失败。检查临时目录 dmp_files/tmp 的权限与磁盘空间。"},
	{Code: 6062, Const: "CodeLogsDeleteFail", Module: "logs", TextKey: "delete fail", Level: LevelWarn, Hint: "删除失败，可能是资源仍被占用（进程未退出、文件被锁）或数据库写入异常。"},
	{Code: 6091, Const: "CodeLogsDBError", Module: "logs", TextKey: "database error", Level: LevelFatal, Hint: "数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。"},

	// tools
	{Code: 7001, Const: "CodeToolsBadRequest", Module: "tools", TextKey: "bad request", Level: LevelWarn, Hint: "请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。"},
	{Code: 7002, Const: "CodeToolsEmbeddingBaseURLInvalid", Module: "tools", TextKey: "embedding base url invalid", Level: LevelWarn, Hint: "向量服务的 Base URL 不合法。填写完整的 http(s) 地址。"},
	{Code: 7003, Const: "CodeToolsEmbeddingNotRunning", Module: "tools", TextKey: "embedding build not running", Level: LevelWarn, Hint: "当前没有正在运行的向量索引构建任务，无需取消。"},
	{Code: 7004, Const: "CodeToolsEmbeddingConfigIncomplete", Module: "tools", TextKey: "embedding config incomplete", Level: LevelWarn, Hint: "向量模型配置不完整（缺少模型名、维度或 API Key）。补全后重试。"},
	{Code: 7005, Const: "CodeToolsEmbeddingConfigInvalid", Module: "tools", TextKey: "embedding config invalid", Level: LevelWarn, Hint: "向量模型配置不合法（Base URL、模型名、维度或 API Key）。按提示补全后重试。"},
	{Code: 7021, Const: "CodeToolsNoPermission", Module: "tools", TextKey: "permission needed", Level: LevelAuth, Hint: "当前账号没有该房间或该操作的管理员权限。让管理员在用户管理里授予对应房间权限。"},
	{Code: 7041, Const: "CodeToolsMapSaveNotFound", Module: "tools", TextKey: "map save not found", Level: LevelWarn, Hint: "地图存档文件不存在。确认该世界已生成过地图存档。"},
	{Code: 7061, Const: "CodeToolsCreateBackupFail", Module: "tools", TextKey: "create backup fail", Level: LevelWarn, Hint: "创建备份失败。检查备份目录权限与磁盘空间。"},
	{Code: 7062, Const: "CodeToolsGetBackupFail", Module: "tools", TextKey: "get backup fail", Level: LevelWarn, Hint: "读取备份列表失败。检查备份目录是否存在。"},
	{Code: 7063, Const: "CodeToolsRestoreFail", Module: "tools", TextKey: "restore fail", Level: LevelWarn, Hint: "恢复备份失败。确认备份文件完整、没有进程占用存档目录。"},
	{Code: 7064, Const: "CodeToolsCreateFail", Module: "tools", TextKey: "create fail", Level: LevelWarn, Hint: "创建失败，常见原因是必填字段缺失或数据库写入异常。查看响应 message 与运行日志。"},
	{Code: 7065, Const: "CodeToolsUpdateFail", Module: "tools", TextKey: "update fail", Level: LevelWarn, Hint: "更新写入失败。确认提交内容合法，并查看 logs/runtime.log 中的具体错误。"},
	{Code: 7066, Const: "CodeToolsDeleteFail", Module: "tools", TextKey: "delete fail", Level: LevelWarn, Hint: "删除失败，可能是资源仍被占用（进程未退出、文件被锁）或数据库写入异常。"},
	{Code: 7067, Const: "CodeToolsExecFail", Module: "tools", TextKey: "exec fail", Level: LevelWarn, Hint: "执行游戏操作失败。确认对应房间进程存在、screen 会话可用。"},
	{Code: 7068, Const: "CodeToolsGetSnapshotFail", Module: "tools", TextKey: "get snapshot fail", Level: LevelWarn, Hint: "读取快照列表失败。检查存档目录与 DST 快照文件。"},
	{Code: 7069, Const: "CodeToolsMapBusy", Module: "tools", TextKey: "map busy", Level: LevelFatal, Hint: "地图正在生成中，请等当前任务结束后再试。"},
	{Code: 7070, Const: "CodeToolsMapParseFail", Module: "tools", TextKey: "map parse fail", Level: LevelWarn, Hint: "解析地图数据失败。存档可能损坏或版本不兼容。"},
	{Code: 7071, Const: "CodeToolsSnapshotStartFail", Module: "tools", TextKey: "snapshot start fail", Level: LevelWarn, Hint: "快照已删除，但房间世界重新启动失败。检查 DST 安装、screen 会话与端口占用，确认世界进程是否恢复运行。"},
	{Code: 7091, Const: "CodeToolsDBError", Module: "tools", TextKey: "database error", Level: LevelFatal, Hint: "数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。"},

	// player
	{Code: 8001, Const: "CodePlayerBadRequest", Module: "player", TextKey: "bad request", Level: LevelWarn, Hint: "请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。"},
	{Code: 8021, Const: "CodePlayerNoPermission", Module: "player", TextKey: "permission needed", Level: LevelAuth, Hint: "当前账号没有该房间或该操作的管理员权限。让管理员在用户管理里授予对应房间权限。"},
	{Code: 8061, Const: "CodePlayerAddFail", Module: "player", TextKey: "add fail", Level: LevelWarn, Hint: "添加失败。确认目标（白名单/黑名单/管理员）内容格式正确。"},
	{Code: 8062, Const: "CodePlayerDeleteFail", Module: "player", TextKey: "delete fail", Level: LevelWarn, Hint: "删除失败，可能是资源仍被占用（进程未退出、文件被锁）或数据库写入异常。"},
	{Code: 8063, Const: "CodePlayerChatMessageFail", Module: "player", TextKey: "chat message fail", Level: LevelWarn, Hint: "读取聊天记录失败。确认聊天日志已生成且可读。"},
	{Code: 8091, Const: "CodePlayerDBError", Module: "player", TextKey: "database error", Level: LevelFatal, Hint: "数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。"},

	// cloud
	{Code: 9001, Const: "CodeCloudBadRequest", Module: "cloud", TextKey: "bad request", Level: LevelWarn, Hint: "请求参数缺失、类型错误或取值非法。核对前端提交的字段名与取值范围，必要时用浏览器开发者工具查看请求体。"},
	{Code: 9002, Const: "CodeCloudArchiveIDInvalid", Module: "cloud", TextKey: "cloud archive id invalid", Level: LevelWarn, Hint: "云存档 ID 格式非法。刷新列表后重试。"},
	{Code: 9021, Const: "CodeCloudArchiveNoPermission", Module: "cloud", TextKey: "cloud archive permission", Level: LevelWarn, Hint: "当前账号没有该房间权限，无法上传云存档。"},
	{Code: 9022, Const: "CodeCloudArchiveCreateNoPermission", Module: "cloud", TextKey: "cloud archive create permission", Level: LevelWarn, Hint: "当前账号没有创建房间的权限，无法导入云存档。"},
	{Code: 9061, Const: "CodeCloudNotLinked", Module: "cloud", TextKey: "cloud not linked", Level: LevelWarn, Hint: "本地账号还没有绑定云账号。先到「云服务」页面完成绑定。"},
	{Code: 9062, Const: "CodeCloudAlreadyLinked", Module: "cloud", TextKey: "cloud already linked", Level: LevelWarn, Hint: "该云账号已经绑定了别的本地账号。先解绑或换一个云账号。"},
	{Code: 9063, Const: "CodeCloudLinkNotFound", Module: "cloud", TextKey: "cloud link not found", Level: LevelWarn, Hint: "云账号绑定不存在或已被解绑。重新绑定后再试。"},
	{Code: 9064, Const: "CodeCloudLocalUserMissing", Module: "cloud", TextKey: "cloud local user missing", Level: LevelWarn, Hint: "云账号对应的本地用户不存在（可能已被删除）。重新绑定或重建本地用户。"},
	{Code: 9065, Const: "CodeCloudLocalUserDisabled", Module: "cloud", TextKey: "cloud local user disabled", Level: LevelWarn, Hint: "云账号对应的本地用户已被禁用。让管理员解除禁用。"},
	{Code: 9066, Const: "CodeCloudPasswordWrong", Module: "cloud", TextKey: "cloud password wrong", Level: LevelWarn, Hint: "云账号或本地密码错误。确认输入；多次失败会被限流。"},
	{Code: 9067, Const: "CodeCloudRelinkRequired", Module: "cloud", TextKey: "cloud relink required", Level: LevelWarn, Hint: "云账号的登录凭据已失效（token 被撤销或密码变更）。重新绑定云账号。"},
	{Code: 9068, Const: "CodeCloudUnavailable", Module: "cloud", TextKey: "cloud unavailable", Level: LevelFatal, Hint: "云服务不可达或返回异常。检查服务器到云服务的网络，稍后重试；本地登录仍可用。"},
	{Code: 9069, Const: "CodeCloudSyncFail", Module: "cloud", TextKey: "cloud sync fail", Level: LevelWarn, Hint: "云同步失败。查看响应 message 与 logs/runtime.log 中的具体原因。"},
	{Code: 9070, Const: "CodeCloudSyncInvalidDirection", Module: "cloud", TextKey: "cloud sync invalid direction", Level: LevelWarn, Hint: "同步方向参数非法，只能是 both/upload/download。"},
	{Code: 9071, Const: "CodeCloudArchiveTooLarge", Module: "cloud", TextKey: "cloud archive too large", Level: LevelWarn, Hint: "云存档超过大小上限。清理日志或快照后重新上传。"},
	{Code: 9072, Const: "CodeCloudArchiveUltimateOnly", Module: "cloud", TextKey: "cloud archive ultimate only", Level: LevelWarn, Hint: "云存档是旗舰套餐功能。升级套餐后重试。"},
	{Code: 9073, Const: "CodeCloudArchiveBuildFail", Module: "cloud", TextKey: "cloud archive build fail", Level: LevelWarn, Hint: "打包云存档失败。检查临时目录权限与磁盘空间。"},
	{Code: 9074, Const: "CodeCloudArchiveImportFail", Module: "cloud", TextKey: "cloud archive import fail", Level: LevelWarn, Hint: "导入云存档失败。存档可能损坏或与当前版本不兼容。"},
	{Code: 9075, Const: "CodeCloudArchiveDownloadFail", Module: "cloud", TextKey: "cloud archive download fail", Level: LevelWarn, Hint: "从云服务下载存档失败。检查网络与云账号状态后重试。"},
	{Code: 9076, Const: "CodeCloudRemoteError", Module: "cloud", TextKey: "cloud remote error", Level: LevelWarn, Hint: "云服务返回了业务错误，具体原因见 message（由云端给出，未经过本地翻译）。"},
	{Code: 9091, Const: "CodeCloudDBError", Module: "cloud", TextKey: "database error", Level: LevelFatal, Hint: "数据库读写失败。检查数据目录（-dbpath）磁盘空间与文件权限，以及 logs/runtime.log 中的 GORM 错误。"},
	{Code: 9092, Const: "CodeCloudLoginFail", Module: "cloud", TextKey: "cloud login fail", Level: LevelWarn, Hint: "云登录失败，常见于签发本地 token 或写入绑定信息时出错。查看 logs/runtime.log。"},
	{Code: 9093, Const: "CodeCloudCreateFail", Module: "cloud", TextKey: "create fail", Level: LevelWarn, Hint: "创建失败，常见原因是必填字段缺失或数据库写入异常。查看响应 message 与运行日志。"},
}

var codeIndex = func() map[int]Entry {
	m := make(map[int]Entry, len(Registry))
	for _, e := range Registry {
		m[e.Code] = e
	}
	return m
}()

var nameIndex = func() map[string]Entry {
	m := make(map[string]Entry, len(Registry))
	for _, e := range Registry {
		m[e.Const] = e
	}
	return m
}()

// LookupCode 按错误码查注册表。
func LookupCode(code int) (Entry, bool) {
	e, ok := codeIndex[code]
	return e, ok
}

// LookupName 按常量名查注册表。
func LookupName(name string) (Entry, bool) {
	e, ok := nameIndex[name]
	return e, ok
}
