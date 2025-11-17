package platform

import (
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v := r.Group(utils.ApiVersion)
	{
		platform := v.Group("platform")
		{
			platform.GET("/overview", middleware.MWtoken(), h.overviewGet)
			platform.GET("/game_version", middleware.MWtoken(), gameVersion)
			platform.GET("/webssh", webSshWS)
		}
	}
}
