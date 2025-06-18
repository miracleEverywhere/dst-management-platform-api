package setting

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func RouteSetting(r *gin.Engine) *gin.Engine {
	v1 := r.Group("v1")
	v1.Use(utils.MWtoken())
	v1.Use(utils.MWUserCheck())
	{
		setting := v1.Group("setting")
		{
			// 设置
			setting.GET("/clusters", handleClustersGet)
			setting.GET("/clusters/all", handleAllClustersGet)
			setting.GET("/clusters/world_port", utils.MWAdminOnly(), handleClustersWorldPortGet)
			setting.GET("/cluster", handleClusterGet)
			setting.POST("/cluster", handleClusterPost)
			setting.PUT("/cluster", utils.MWAdminOnly(), handleClusterPut)
			setting.DELETE("/cluster", utils.MWAdminOnly(), handleClusterDelete)
			setting.PUT("/cluster/status", utils.MWAdminOnly(), handleClusterStatusPut)
			setting.POST("/cluster/save", handleClusterSavePost)
			setting.POST("/cluster/save_restart", handleClusterSaveRestartPost)
			setting.POST("/cluster/save_regenerate", handleClusterSaveRegeneratePost)
			// Player
			setting.GET("/player/list", handlePlayerListGet)
			setting.GET("/player/list/history", handleHistoryPlayerGet)
			setting.POST("player/history/clean", handleHistoryPlayerCleanPost)
			setting.POST("/player/change", handlePlayerListChangePost)
			setting.POST("/player/add/block/upload", handleBlockUpload)
			setting.POST("/player/kick", handleKick)
			// 存档导入
			setting.POST("/import/upload", handleImportPost)
			// MOD
			setting.GET("/mod/setting/format", handleModSettingFormatGet)
			setting.GET("/mod/config_options", handleModConfigOptionsGet)
			setting.POST("/mod/download", handleModDownloadPost)
			setting.GET("/mod/download/process", handleModDownloadProcessGet)
			setting.POST("/mod/sync", handleSyncModPost)
			setting.POST("/mod/delete", handleDeleteDownloadedModPost)
			setting.POST("/mod/enable", handleEnableModPost)
			setting.POST("/mod/disable", handleDisableModPost)
			setting.POST("/mod/config/change", handleModConfigChangePost)
			setting.POST("/mod/export/macos", handleMacOSModExportPost)
			setting.POST("/mod/update", handleModUpdatePost)
			setting.POST("/mod/add/clint_mods_disabled", handleAddClientModsDisabledConfig)
			setting.POST("/mod/delete/clint_mods_disabled", handleDeleteClientModsDisabledConfig)
			// System
			setting.GET("/system/setting", handleSystemSettingGet)
			setting.PUT("/system/setting", handleSystemSettingPut)
		}
	}

	return r
}
