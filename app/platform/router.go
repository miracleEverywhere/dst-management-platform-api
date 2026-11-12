package platform

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/middleware"

	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v := r.Group(constant.ApiVersion)
	{
		platform := v.Group("platform")
		platform.Use(middleware.TokenCheck())
		{
			platform.GET("/readme", h.readmeGet)
			platform.POST("/readme", h.readmePost)
			platform.GET("/overview", middleware.AdminOnly(), h.overviewGet)
			platform.GET("/game_version", h.gameVersionGet)
			platform.GET("/webssh", middleware.AdminOnly(), h.websshWS)
			platform.GET("/os_info", h.osInfoGet)
			platform.GET("/metrics", middleware.AdminOnly(), h.metricsGet)
			platform.GET("/global_settings", middleware.AdminOnly(), h.globalSettingsGet)
			platform.POST("/global_settings", middleware.AdminOnly(), h.globalSettingsPost)
			platform.GET("/screen/running", middleware.AdminOnly(), h.screenRunningGet)
			platform.POST("/screen/kill", middleware.AdminOnly(), h.screenKillPost)
			platform.POST("/webhook/test", middleware.AdminOnly(), h.webhookTestPost)
			platform.GET("/webhook/events", h.webhookEventsGet)
			platform.GET("/plugin/list", middleware.AdminOnly(), h.pluginListGet)
			platform.POST("/plugin/install", middleware.AdminOnly(), h.pluginInstallPost)
			platform.POST("/plugin/action", middleware.AdminOnly(), h.pluginActionPost)
			platform.GET("/plugin/status", h.pluginStatusGet)
		}
	}
}
