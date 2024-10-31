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
		}
	}

	return r
}
