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
			platform.GET("/overview", middleware.TokenCheck(), middleware.AdminOnly(), h.overviewGet)
			platform.GET("/game_version", middleware.TokenCheck(), gameVersionGet)
			platform.GET("/webssh", websshWS)
			platform.GET("/os_info", middleware.TokenCheck(), osInfoGet)
			platform.GET("/metrics", middleware.TokenCheck(), middleware.AdminOnly(), metricsGet)
			platform.GET("/global_settings", middleware.TokenCheck(), middleware.AdminOnly(), h.globalSettingsGet)
			platform.POST("/global_settings", middleware.TokenCheck(), middleware.AdminOnly(), h.globalSettingsPost)
			platform.GET("/screen/running", middleware.TokenCheck(), middleware.AdminOnly(), h.screenRunningGet)
			platform.POST("/screen/kill", middleware.TokenCheck(), middleware.AdminOnly(), screenKillPost)
		}
	}
}
