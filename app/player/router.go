package player

import (
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v := r.Group(utils.ApiVersion)
	{
		player := v.Group("player")
		player.Use(middleware.MWtoken())
		{
			player.GET("/online", h.onlineGet)
			player.GET("/list", h.listGet)
			player.POST("/list", h.listPost)
			player.GET("/uidmap", h.uidMapGet)

		}
	}
}
