package constant

// 业务错误码。按「错误原因」唯一：每个原因一个码，注册表见 code_registry.go。

// 号段：
//
//	0          框架层（token/权限/限流/参数）
//	1000-1999  用户模块
//	2000-2999  房间模块
//	3000-3999  模组模块
//	4000-4999  控制台模块
//	5000-5999  平台模块
//	6000-6999  日志模块
//	7000-7999  工具模块
//	8000-8999  玩家模块
//	9000-9999  云服务模块
//
// 模块内后两位表示类别：01-19 参数校验、21-39 权限、41-59 资源不存在、61-89 业务规则、91-99 数据库/服务端内部。

const (
	// 框架层（token/权限/限流/参数）
	CodeFrameworkBadRequest   = 1
	CodeFrameworkNoPermission = 2
	CodeAuthTokenInvalid      = 420
	CodeTooManyRequests       = 429

	// 用户模块
	CodeUserBadRequest         = 1001
	CodeUserNoPermission       = 1021
	CodeUserNotExist           = 1041
	CodeUserExist              = 1061
	CodeUserWrongPassword      = 1062
	CodeUserDisabled           = 1063
	CodeUserDeleteAllForbidden = 1064
	CodeUserUpdateFail         = 1065
	CodeUserDBError            = 1091
	CodeUserCreateFail         = 1092
	CodeUserLoginFail          = 1093
	CodeUserRevokeFail         = 1094
	CodeUserDeleteFail         = 1095

	// 房间模块
	CodeRoomBadRequest               = 2001
	CodeRoomInvalidURL               = 2002
	CodeRoomInvalidWorldName         = 2003
	CodeRoomDuplicateWorldName       = 2004
	CodeRoomFavoriteInvalidGameMode  = 2005
	CodeRoomFavoriteInvalidWebhook   = 2006
	CodeRoomNoPermission             = 2021
	CodeRoomNotFound                 = 2041
	CodeRoomFavoriteNotFound         = 2042
	CodeRoomPortConflict             = 2061
	CodeRoomWriteFileFail            = 2062
	CodeRoomFavoriteNoWorlds         = 2063
	CodeRoomFavoriteDataInvalid      = 2064
	CodeRoomFavoriteNameRequired     = 2065
	CodeRoomActivateFail             = 2066
	CodeRoomUpdateFail               = 2067
	CodeRoomDeleteFail               = 2068
	CodeRoomWebhookTestFail          = 2069
	CodeRoomImportNoWorlds           = 2070
	CodeRoomImportWorldCountMismatch = 2071
	CodeRoomUploadSaveFail           = 2072
	CodeRoomUnzipFail                = 2073
	CodeRoomClusterDirNotFound       = 2074
	CodeRoomClusterIniNotFound       = 2075
	CodeRoomClusterIniReadFail       = 2076
	CodeRoomClusterNameNotFound      = 2077
	CodeRoomGameModeNotFound         = 2078
	CodeRoomGameModeInvalid          = 2079
	CodeRoomServerIniNotFound        = 2080
	CodeRoomServerIniReadFail        = 2081
	CodeRoomIsMasterNotFound         = 2082
	CodeRoomIsMasterReadFail         = 2083
	CodeRoomLevelDataNotFound        = 2084
	CodeRoomWorldsDirInvalid         = 2085
	CodeRoomDuplicateWorldNameInSave = 2086
	CodeRoomInvalidWorldNameInSave   = 2087
	CodeRoomDBError                  = 2091

	// 模组模块
	CodeModBadRequest         = 3001
	CodeModNoPermission       = 3021
	CodeModSearchFail         = 3061
	CodeModConfigOptionsError = 3062
	CodeModConfigValuesError  = 3063
	CodeModModifyConfigError  = 3064
	CodeModEnableFail         = 3065
	CodeModDisableFail        = 3066
	CodeModGetEnabledFail     = 3067
	CodeModDeleteFail         = 3068
	CodeModDownloadFail       = 3069
	CodeModDBError            = 3091
	CodeModServerError        = 3092

	// 控制台模块
	CodeDashboardBadRequest         = 4001
	CodeDashboardNoPermission       = 4021
	CodeDashboardExecFail           = 4061
	CodeDashboardStartupFail        = 4062
	CodeDashboardShutdownFail       = 4063
	CodeDashboardRestartFail        = 4064
	CodeDashboardResetFail          = 4065
	CodeDashboardDeleteGameFail     = 4066
	CodeDashboardAnnounceFail       = 4067
	CodeDashboardSystemMsgFail      = 4068
	CodeDashboardConnectionCodeFail = 4069
	CodeDashboardCheckLobbyFail     = 4070
	CodeDashboardUpdateFail         = 4071
	CodeDashboardDBError            = 4091

	// 平台模块
	CodePlatformBadRequest      = 5001
	CodePlatformInvalidURL      = 5002
	CodePlatformInstallFail     = 5061
	CodePlatformUninstallFail   = 5062
	CodePlatformUpdateFail      = 5063
	CodePlatformOSInfoFail      = 5064
	CodePlatformScreensFail     = 5065
	CodePlatformKillScreenFail  = 5066
	CodePlatformWebhookTestFail = 5067
	CodePlatformDBError         = 5091

	// 日志模块
	CodeLogsBadRequest   = 6001
	CodeLogsNoPermission = 6021
	CodeLogsDownloadFail = 6061
	CodeLogsDeleteFail   = 6062
	CodeLogsDBError      = 6091

	// 工具模块
	CodeToolsBadRequest                = 7001
	CodeToolsEmbeddingBaseURLInvalid   = 7002
	CodeToolsEmbeddingNotRunning       = 7003
	CodeToolsEmbeddingConfigIncomplete = 7004
	CodeToolsEmbeddingConfigInvalid    = 7005
	CodeToolsNoPermission              = 7021
	CodeToolsMapSaveNotFound           = 7041
	CodeToolsCreateBackupFail          = 7061
	CodeToolsGetBackupFail             = 7062
	CodeToolsRestoreFail               = 7063
	CodeToolsCreateFail                = 7064
	CodeToolsUpdateFail                = 7065
	CodeToolsDeleteFail                = 7066
	CodeToolsExecFail                  = 7067
	CodeToolsGetSnapshotFail           = 7068
	CodeToolsMapBusy                   = 7069
	CodeToolsMapParseFail              = 7070
	CodeToolsSnapshotStartFail         = 7071
	CodeToolsDBError                   = 7091

	// 玩家模块
	CodePlayerBadRequest      = 8001
	CodePlayerNoPermission    = 8021
	CodePlayerAddFail         = 8061
	CodePlayerDeleteFail      = 8062
	CodePlayerChatMessageFail = 8063
	CodePlayerDBError         = 8091

	// 云服务模块
	CodeCloudBadRequest                = 9001
	CodeCloudArchiveIDInvalid          = 9002
	CodeCloudArchiveNoPermission       = 9021
	CodeCloudArchiveCreateNoPermission = 9022
	CodeCloudNotLinked                 = 9061
	CodeCloudAlreadyLinked             = 9062
	CodeCloudLinkNotFound              = 9063
	CodeCloudLocalUserMissing          = 9064
	CodeCloudLocalUserDisabled         = 9065
	CodeCloudPasswordWrong             = 9066
	CodeCloudRelinkRequired            = 9067
	CodeCloudUnavailable               = 9068
	CodeCloudSyncFail                  = 9069
	CodeCloudSyncInvalidDirection      = 9070
	CodeCloudArchiveTooLarge           = 9071
	CodeCloudArchiveUltimateOnly       = 9072
	CodeCloudArchiveBuildFail          = 9073
	CodeCloudArchiveImportFail         = 9074
	CodeCloudArchiveDownloadFail       = 9075
	CodeCloudRemoteError               = 9076
	CodeCloudDBError                   = 9091
	CodeCloudLoginFail                 = 9092
	CodeCloudCreateFail                = 9093
)
