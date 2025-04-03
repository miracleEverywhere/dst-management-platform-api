package auth

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"time"
)

func RouteAuth(r *gin.Engine) *gin.Engine {
	// 同一个IP每分钟最多登录3次
	ipLimiter := utils.NewIPRateLimiter(3, time.Minute)
	defer ipLimiter.Stop()
	v1 := r.Group("v1")
	{
		// 系统
		v1.POST("/login", ipLimiter.MWIPLimiter(), handleLogin)
		v1.GET("/userinfo", utils.MWtoken(), handleUserinfo)
		v1.GET("/menu", utils.MWtoken(), handleMenu)
		v1.POST("/update/password", utils.MWtoken(), handleUpdatePassword)
	}

	return r
}
