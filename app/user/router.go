package user

import (
	"dst-management-platform-api/constants"
	"dst-management-platform-api/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v := r.Group(constants.ApiVersion)
	{
		user := v.Group("user")
		{
			user.POST("/register", h.registerPost)
			user.POST("/create", h.createPost)
			user.POST("/login", h.loginPost)
			user.GET("/menu", middleware.MWtoken(), h.menuGet)
			user.GET("/userinfo", middleware.MWtoken(), h.userInfo)
		}
	}
}
