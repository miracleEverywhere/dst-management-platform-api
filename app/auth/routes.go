package auth

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func RouteAuth(r *gin.Engine) *gin.Engine {
	v1 := r.Group("v1")
	{
		// 系统
		v1.POST("/login", handleLogin)
		v1.GET("/userinfo", utils.MWtoken(), handleUserinfo)
		v1.GET("/menu", utils.MWtoken(), handleMenu)
		v1.POST("/update/password", utils.MWtoken(), handleUpdatePassword)

		// 设置
		v1.GET("/setting/room/base", utils.MWtoken(), handleRoomSettingBaseGet)
		v1.POST("/setting/room/base", utils.MWtoken(), handleRoomSettingBasePost)
		v1.GET("/setting/room/ground", utils.MWtoken(), handleRoomSettingGroundGet)
		v1.POST("/setting/room/ground", utils.MWtoken(), handleRoomSettingGroundPost)
		v1.GET("/setting/room/cave", utils.MWtoken(), handleRoomSettingCaveGet)
		v1.POST("/setting/room/cave", utils.MWtoken(), handleRoomSettingCavePost)
		v1.GET("/setting/room/mod", utils.MWtoken(), handleRoomSettingModGet)
		v1.POST("/setting/room/mod", utils.MWtoken(), handleRoomSettingModPost)
	}

	return r
}
