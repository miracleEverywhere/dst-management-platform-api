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
			user.GET("/register", h.registerGet)
			user.POST("/register", h.registerPost)
			user.POST("/login", h.loginPost)
			user.GET("/base", middleware.TokenCheck(), h.baseGet)
			user.POST("/base", middleware.TokenCheck(), middleware.AdminOnly(), h.basePost)
			user.PUT("/base", middleware.TokenCheck(), middleware.AdminOnly(), h.basePut)
			user.DELETE("/base", middleware.TokenCheck(), middleware.AdminOnly(), h.baseDelete)
			user.GET("/menu", middleware.TokenCheck(), h.menuGet)
			user.GET("/list", middleware.TokenCheck(), middleware.AdminOnly(), h.userListGet)
			user.PUT("/myself", middleware.TokenCheck(), h.myselfPut)
		}
	}
}
