package tools

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func RouteTools(r *gin.Engine) *gin.Engine {
	v1 := r.Group("v1")
	v1.Use(utils.MWtoken())
	v1.Use(utils.MWUserCheck())
	{
		tools := v1.Group("tools")
		{
			//// 安装
			tools.GET("/os_info", handleOSInfoGet)
			tools.POST("/install", utils.MWAdminOnly(), handleInstall)
			tools.GET("/install/status", utils.MWAdminOnly(), handleGetInstallStatus)
			//// 定时通知
			tools.GET("/announce", handleAnnounceGet)
			tools.POST("/announce", handleAnnouncePost)
			tools.DELETE("/announce", handleAnnounceDelete)
			tools.PUT("/announce", handleAnnouncePut)
			// 备份管理
			tools.GET("/backup", handleBackupGet)
			tools.POST("/backup", handleBackupPost) // 手动创建备份
			tools.DELETE("/backup", handleBackupDelete)
			tools.DELETE("/backup/multi", handleMultiDelete)
			tools.POST("/backup/restore", handleBackupRestore)
			tools.POST("/backup/download", handleBackupDownload)
			// 统计信息
			tools.GET("/statistics", handleStatisticsGet)
			// 令牌
			tools.POST("/token", handleCreateTokenPost)
			//// 监控
			tools.GET("/metrics", handleMetricsGet)
			//// 版本
			tools.GET("/version", handleVersionGet)
		}
	}

	return r
}
