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
		player.Use(middleware.TokenCheck())
		{
			player.GET("/online", h.onlineGet)
			player.GET("/list", h.listGet)
			player.POST("/list", h.listPost)
			player.GET("/uidmap", h.uidMapGet)
			player.GET("/statistics/online_time", h.statisticsOnlineTimeGet)
			player.GET("/statistics/player_count", h.statisticsPlayerCountGet)
		}
	}
}
