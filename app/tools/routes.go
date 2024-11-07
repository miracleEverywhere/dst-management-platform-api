package tools

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func RouteTools(r *gin.Engine) *gin.Engine {
	v1 := r.Group("v1")
	{
		tools := v1.Group("tools")
		{
			// 安装
			tools.GET("/os_info", utils.MWtoken(), handleOSInfoGet)
			tools.POST("/install", utils.MWtoken(), handleInstall)
			tools.GET("/install/status", utils.MWtoken(), handleGetInstallStatus)
			// 定时通知
			tools.GET("/announce", utils.MWtoken(), handleAnnounceGet)
			tools.POST("/announce", utils.MWtoken(), handleAnnouncePost)
			tools.DELETE("/announce", utils.MWtoken(), handleAnnounceDelete)
			tools.PUT("/announce", utils.MWtoken(), handleAnnouncePut)
			// 定时更新
			tools.GET("/update", utils.MWtoken(), handleUpdateGet)
			tools.PUT("/update", utils.MWtoken(), handleUpdatePut)
			// 定时备份
			tools.GET("/backup", utils.MWtoken(), handleBackupGet)
			tools.PUT("/backup", utils.MWtoken(), handleBackupPut)
			tools.DELETE("/backup", utils.MWtoken(), handleBackupDelete)
			tools.DELETE("/backup/multi", utils.MWtoken(), handleMultiDelete)
			tools.POST("/backup/restore", utils.MWtoken(), handleBackupRestore)
			//MOD
			//tools.POST("/mod/install/all", utils.MWtoken(), handleDownloadModManualPost)
			// 统计信息
			tools.GET("/statistics", utils.MWtoken(), handleStatisticsGet)

		}
	}

	return r
}
