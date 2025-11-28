package logs

import (
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v := r.Group(utils.ApiVersion)
	{
		logs := v.Group("logs")
		logs.Use(middleware.MWtoken())
		{
			logs.GET("/content", h.contentGet)
		}
	}
}
