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
			platform.GET("/overview", middleware.MWtoken(), middleware.MWAdminOnly(), h.overviewGet)
			platform.GET("/game_version", middleware.MWtoken(), gameVersionGet)
			platform.GET("/webssh", websshWS)
			platform.GET("/os_info", middleware.MWtoken(), osInfoGet)
			platform.GET("/metrics", middleware.MWtoken(), middleware.MWAdminOnly(), metricsGet)
			platform.GET("/global_settings", middleware.MWtoken(), middleware.MWAdminOnly(), h.globalSettingsGet)
			platform.POST("/global_settings", middleware.MWtoken(), middleware.MWAdminOnly(), h.globalSettingsPost)
			platform.GET("/screen/running", middleware.MWtoken(), middleware.MWAdminOnly(), h.screenRunningGet)
			platform.POST("/screen/kill", middleware.MWtoken(), middleware.MWAdminOnly(), screenKillPost)
		}
	}
}
