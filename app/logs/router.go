package logs

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/middleware"

	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v := r.Group(constant.ApiVersion)
	{
		logs := v.Group("logs")
		logs.Use(middleware.TokenCheck())
		{
			logs.GET("/content", h.contentGet)
			logs.GET("/history/list", h.historyListGet)
			logs.GET("/history/content", h.historyContentGet)
			logs.GET("/clean/info", middleware.AdminOnly(), h.cleanInfoGet)
			logs.DELETE("/clean", middleware.AdminOnly(), h.cleanDelete)
			logs.GET("/download", h.downloadGet)
		}
	}
}
