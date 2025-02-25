package setting

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func RouteSetting(r *gin.Engine) *gin.Engine {
	v1 := r.Group("v1")
	{
		setting := v1.Group("setting")
		{
			// 设置
			setting.GET("/room", utils.MWtoken(), handleRoomSettingGet)
			setting.GET("/room/multihost", utils.MWtoken(), handleGetMultiHostGet)
			setting.POST("/room/multihost", utils.MWtoken(), handleChangeMultiHostPost)
			setting.POST("/room/save", utils.MWtoken(), handleRoomSettingSavePost)
			setting.POST("/room/save_restart", utils.MWtoken(), handleRoomSettingSaveAndRestartPost)
			setting.POST("/room/save_generate", utils.MWtoken(), handleRoomSettingSaveAndGeneratePost)
			// Player
			setting.GET("/player/list", utils.MWtoken(), handlePlayerListGet)
			setting.GET("/player/list/history", utils.MWtoken(), handleHistoryPlayerGet)
			setting.POST("/player/add/admin", utils.MWtoken(), handleAdminAddPost)
			setting.POST("/player/add/block", utils.MWtoken(), handleBlockAddPost)
			setting.POST("/player/add/block/upload", utils.MWtoken(), handleBlockUpload)
			setting.POST("/player/add/white", utils.MWtoken(), handleWhiteAddPost)
			setting.POST("/player/delete/admin", utils.MWtoken(), handleAdminDeletePost)
			setting.POST("/player/delete/block", utils.MWtoken(), handleBlockDeletePost)
			setting.POST("/player/delete/white", utils.MWtoken(), handleWhiteDeletePost)
			setting.POST("/player/kick", utils.MWtoken(), handleKick)
			// 存档导入
			setting.POST("/import/upload", utils.MWtoken(), handleImportFileUploadPost)
			// MOD
			setting.GET("/mod/setting/format", utils.MWtoken(), handleModSettingFormatGet)
			setting.GET("/mod/config_options", utils.MWtoken(), handleModConfigOptionsGet)
			setting.POST("/mod/download", utils.MWtoken(), handleModDownloadPost)
			setting.POST("/mod/sync", utils.MWtoken(), handleSyncModPost)
			setting.POST("/mod/delete", utils.MWtoken(), handleDeleteDownloadedModPost)
			setting.POST("/mod/enable", utils.MWtoken(), handleEnableModPost)
			setting.POST("/mod/disable", utils.MWtoken(), handleDisableModPost)
			setting.POST("/mod/config/change", utils.MWtoken(), handleModConfigChangePost)
			setting.POST("/mod/export/macos", utils.MWtoken(), handleMacOSModExportPost)
			setting.POST("/mod/update", utils.MWtoken(), handleModUpdatePost)
			setting.POST("/mod/add/clint_mods_disabled", utils.MWtoken(), handleAddClientModsDisabledConfig)
			setting.POST("/mod/delete/clint_mods_disabled", utils.MWtoken(), handleDeleteClientModsDisabledConfig)
			// System
			setting.GET("/system/setting", utils.MWtoken(), handleSystemSettingGet)
			setting.PUT("/system/setting", utils.MWtoken(), handleSystemSettingPut)
		}
	}

	return r
}
