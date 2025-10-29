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
			user.POST("/login", h.loginPost)
			user.GET("/base", middleware.MWtoken(), h.baseGet)
			user.POST("/base", middleware.MWtoken(), h.basePost)
			user.GET("/menu", middleware.MWtoken(), h.menuGet)
		}
	}
}
