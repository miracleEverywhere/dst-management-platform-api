package home

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func RouteHome(r *gin.Engine) *gin.Engine {
	v1 := r.Group("v1")
	{
		home := v1.Group("home")
		{
			// 获取房间设置、季节、天数等
			home.GET("/room_info", utils.MWtoken(), handleRoomInfoGet)
			// 获取系统资源监控
			home.GET("/sys_info", utils.MWtoken(), handleSystemInfoGet)
			home.POST("/exec", utils.MWtoken(), handleExecPost)
			home.POST("/announce", utils.MWtoken(), handleAnnouncementPost)
			home.POST("/console", utils.MWtoken(), handleConsolePost)
		}
	}

	return r
}
