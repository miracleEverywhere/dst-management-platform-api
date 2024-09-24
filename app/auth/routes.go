package auth

import "github.com/gin-gonic/gin"

func RouteAuth(r *gin.Engine) *gin.Engine {
	v1 := r.Group("v1")
	{
		v1.POST("/login", handleLogin)
		v1.GET("/userinfo", handleUserinfo)
	}

	return r
}
