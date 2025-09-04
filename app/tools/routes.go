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
			tools.GET("/os_info", utils.MWtoken(), utils.MWUserCheck(), handleOSInfoGet)
			tools.POST("/install", utils.MWtoken(), utils.MWUserCheck(), utils.MWAdminOnly(), handleInstall)
			tools.GET("/install/status", utils.MWtoken(), utils.MWUserCheck(), utils.MWAdminOnly(), handleGetInstallStatus)
			tools.GET("/install/is_installing", utils.MWtoken(), utils.MWUserCheck(), handleGetIsInstallingGet)
			// 定时通知
			tools.GET("/announce", utils.MWtoken(), utils.MWUserCheck(), handleAnnounceGet)
			tools.POST("/announce", utils.MWtoken(), utils.MWUserCheck(), handleAnnouncePost)
			tools.DELETE("/announce", utils.MWtoken(), utils.MWUserCheck(), handleAnnounceDelete)
			tools.PUT("/announce", utils.MWtoken(), utils.MWUserCheck(), handleAnnouncePut)
			// 备份管理
			tools.GET("/backup", utils.MWtoken(), utils.MWUserCheck(), handleBackupGet)
			// 手动创建备份
			tools.POST("/backup", utils.MWtoken(), utils.MWUserCheck(), handleBackupPost)
			tools.DELETE("/backup", utils.MWtoken(), utils.MWUserCheck(), handleBackupDelete)
			tools.DELETE("/backup/multi", utils.MWtoken(), utils.MWUserCheck(), handleMultiDelete)
			tools.POST("/backup/restore", utils.MWtoken(), utils.MWUserCheck(), handleBackupRestore)
			tools.POST("/backup/import", utils.MWtoken(), utils.MWUserCheck(), handleBackupImport)
			//tools.POST("/backup/download", handleBackupDownload)
			// 统计信息
			tools.GET("/statistics", utils.MWtoken(), utils.MWUserCheck(), handleStatisticsGet)
			// 令牌
			tools.POST("/token", utils.MWtoken(), utils.MWUserCheck(), handleCreateTokenPost)
			// 监控
			tools.GET("/metrics", utils.MWtoken(), utils.MWUserCheck(), handleMetricsGet)
			// 版本
			tools.GET("/version", utils.MWtoken(), utils.MWUserCheck(), handleVersionGet)
			// 终端
			tools.GET("/webssh", handleWebSSHGet)
			// 世界总览
			tools.GET("/location", utils.MWtoken(), utils.MWUserCheck(), handleLocationGet)
		}
	}

	return r
}
