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
			setting.GET("/clusters", utils.MWtoken(), handleClustersGet)
			setting.GET("/cluster", utils.MWtoken(), handleClusterGet)
			setting.POST("/cluster", utils.MWtoken(), handleClusterPost)
			setting.POST("/cluster/save", utils.MWtoken(), handleClusterSavePost)
			setting.POST("/cluster/save_restart", utils.MWtoken(), handleClusterSaveRestartPost)
			setting.POST("/cluster/save_regenerate", utils.MWtoken(), handleClusterSaveRegeneratePost)
			//// Player
			setting.GET("/player/list", utils.MWtoken(), handlePlayerListGet)
			setting.GET("/player/list/history", utils.MWtoken(), handleHistoryPlayerGet)
			setting.POST("player/history/clean", utils.MWtoken(), handleHistoryPlayerCleanPost)
			setting.POST("/player/change", utils.MWtoken(), handlePlayerListChangePost)
			setting.POST("/player/add/block/upload", utils.MWtoken(), handleBlockUpload)
			setting.POST("/player/kick", utils.MWtoken(), handleKick)
			// 存档导入
			setting.POST("/import/upload", utils.MWtoken(), handleImportPost)
			//// MOD
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
			//// System
			setting.GET("/system/setting", utils.MWtoken(), handleSystemSettingGet)
			setting.PUT("/system/setting", utils.MWtoken(), handleSystemSettingPut)
		}
	}

	return r
}
