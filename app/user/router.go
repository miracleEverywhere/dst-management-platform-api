package user

import (
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v := r.Group(utils.ApiVersion)
	{
		user := v.Group("user")
		{
			user.POST("/register", h.registerPost)
			user.POST("/login", h.loginPost)
			user.GET("/base", middleware.MWtoken(), h.baseGet)
			user.POST("/base", middleware.MWtoken(), middleware.MWAdminOnly(), h.basePost)
			user.GET("/menu", middleware.MWtoken(), h.menuGet)
			user.GET("/list", middleware.MWtoken(), middleware.MWAdminOnly(), h.userListGet)
		}
	}
}
