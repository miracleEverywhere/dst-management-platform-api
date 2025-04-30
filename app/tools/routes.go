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
			//// 安装
			tools.GET("/os_info", utils.MWtoken(), handleOSInfoGet)
			//tools.POST("/install", utils.MWtoken(), handleInstall)
			//tools.GET("/install/status", utils.MWtoken(), handleGetInstallStatus)
			//// 定时通知
			tools.GET("/announce", utils.MWtoken(), handleAnnounceGet)
			tools.POST("/announce", utils.MWtoken(), handleAnnouncePost)
			tools.DELETE("/announce", utils.MWtoken(), handleAnnounceDelete)
			tools.PUT("/announce", utils.MWtoken(), handleAnnouncePut)
			// 备份管理
			tools.GET("/backup", utils.MWtoken(), handleBackupGet)
			tools.POST("/backup", utils.MWtoken(), handleBackupPost) // 手动创建备份
			tools.DELETE("/backup", utils.MWtoken(), handleBackupDelete)
			tools.DELETE("/backup/multi", utils.MWtoken(), handleMultiDelete)
			tools.POST("/backup/restore", utils.MWtoken(), handleBackupRestore)
			tools.POST("/backup/download", utils.MWtoken(), handleBackupDownload)
			//// 统计信息
			//tools.GET("/statistics", utils.MWtoken(), handleStatisticsGet)
			//// 自动保活
			//tools.GET("/keepalive", utils.MWtoken(), handleKeepaliveGet)
			//tools.PUT("/keepalive", utils.MWtoken(), handleKeepalivePut)
			//// 帮助页面替换steam so文件
			//// 不想再开一个router了，就塞在tools里，后续官方修复后会删除
			//tools.POST("/replace_so", utils.MWtoken(), handleReplaceDSTSOFile)
			//// 令牌
			//tools.POST("/token", utils.MWtoken(), handleCreateTokenPost)
			//// 监控
			//tools.GET("/metrics", utils.MWtoken(), handleMetricsGet)
			//// 版本
			//tools.GET("/version", utils.MWtoken(), handleVersionGet)
		}
	}

	return r
}
