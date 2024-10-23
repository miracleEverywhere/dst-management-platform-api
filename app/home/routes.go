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
			// 设置
			home.GET("/room_info", utils.MWtoken(), handleRoomInfoGet)
		}
	}

	return r
}
