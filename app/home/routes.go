package home

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func RouteHome(r *gin.Engine) *gin.Engine {
	v1 := r.Group("v1")
	v1.Use(utils.MWtoken())
	v1.Use(utils.MWUserCheck())
	{
		home := v1.Group("home")
		{
			// 获取房间设置、季节、天数等
			home.GET("/room_info", handleRoomInfoGet)
			// 获取系统资源监控
			home.GET("/sys_info", handleSystemInfoGet)
			home.GET("/world_info", handleWorldInfoGet)
			home.GET("/cluster/all_screens", handleGetClusterAllScreensGet)
			home.POST("/cluster/screen_kill", handleKillScreenManualPost)
			home.POST("/exec", handleExecPost)
			// 是否增在更新游戏
			home.GET("/update/is_updating", handleGetIsUpdatingGet)
		}
	}

	return r
}
