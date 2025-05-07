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
		v1.GET("/userinfo", utils.MWtoken(), utils.MWUserCheck(), handleUserinfo)
		v1.GET("/menu", utils.MWtoken(), utils.MWUserCheck(), handleMenu)
		v1.POST("/update/password", utils.MWtoken(), utils.MWUserCheck(), handleUpdatePassword)
		// 用户
		v1.GET("/user/list", utils.MWtoken(), utils.MWUserCheck(), handleUserListGet)
		v1.POST("/user", utils.MWtoken(), utils.MWUserCheck(), utils.MWAdminOnly(), handleUserCreatePost)
		v1.PUT("/user", utils.MWtoken(), utils.MWUserCheck(), utils.MWAdminOnly(), handleUserUpdatePut)
		v1.DELETE("/user", utils.MWtoken(), utils.MWUserCheck(), utils.MWAdminOnly(), handleUserDeleteDelete)
		v1.POST("/register", handleRegisterPost)
		v1.GET("/announce_id", utils.MWtoken(), utils.MWUserCheck(), handleUserAnnounceIDGet)
		v1.POST("/announce_id", utils.MWtoken(), utils.MWUserCheck(), handleUserAnnounceIDPost)
	}

	return r
}
