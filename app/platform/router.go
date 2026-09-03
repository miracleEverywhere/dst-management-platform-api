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
		platform.Use(middleware.TokenCheck())
		{
			platform.GET("/readme", h.readmeGet)
			platform.POST("/readme", h.readmePost)
			platform.GET("/overview", middleware.AdminOnly(), h.overviewGet)
			platform.GET("/game_version", gameVersionGet)
			platform.GET("/webssh", h.websshWS)
			platform.GET("/os_info", osInfoGet)
			platform.GET("/metrics", middleware.AdminOnly(), metricsGet)
			platform.GET("/global_settings", middleware.AdminOnly(), h.globalSettingsGet)
			platform.POST("/global_settings", middleware.AdminOnly(), h.globalSettingsPost)
			platform.GET("/screen/running", middleware.AdminOnly(), h.screenRunningGet)
			platform.POST("/screen/kill", middleware.AdminOnly(), screenKillPost)
			platform.POST("/webhook/test", middleware.AdminOnly(), webhookTestPost)
			platform.GET("/webhook/events", webhookEventsGet)
			platform.GET("/plugin/list", middleware.AdminOnly(), h.pluginListGet)
			platform.POST("/plugin/install", middleware.AdminOnly(), h.pluginInstallPost)
			platform.POST("/plugin/action", middleware.AdminOnly(), h.pluginActionPost)
			platform.GET("/plugin/status", h.pluginStatusGet)
			platform.GET("/os", osTypeGet)
		}
	}
}
