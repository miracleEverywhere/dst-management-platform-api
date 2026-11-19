package diagnosis

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/middleware"

	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v := r.Group(constant.ApiVersion)
	{
		diagnosis := v.Group("diagnosis")
		diagnosis.Use(middleware.TokenCheck())
		{
			diagnosis.POST("", h.startPost)
			diagnosis.GET("/setting", middleware.AdminOnly(), h.settingGet)
			diagnosis.PUT("/setting", middleware.AdminOnly(), h.settingPut)
			diagnosis.GET("/history", h.historyGet)
			diagnosis.GET("/history/:id", h.historyDetailGet)
			diagnosis.GET("/:id", h.getGet)
			diagnosis.DELETE("/:id", h.cancelDelete)
			diagnosis.GET("/:id/events", h.eventsGet)
			diagnosis.POST("/:id/apply", h.applyPost)
			diagnosis.GET("/:id/apply/events", h.applyEventsGet)
		}
	}
}
