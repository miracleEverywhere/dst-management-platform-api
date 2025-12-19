package dashboard

import (
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/utils"

	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v := r.Group(utils.ApiVersion)
	{
		dashboard := v.Group("dashboard")
		dashboard.Use(middleware.TokenCheck())
		{
			dashboard.POST("/exec/game", h.execGamePost)
			dashboard.GET("/info/base", h.infoBaseGet)
			dashboard.GET("/info/sys", h.infoSysGet)
			dashboard.GET("/connection_code", h.connectionCodeGet)
			dashboard.PUT("/connection_code", h.connectionCodePut)
			dashboard.POST("/check/lobby", checkLobbyPost)
		}
	}
}
