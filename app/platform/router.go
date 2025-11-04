package platform

import (
	"dst-management-platform-api/constants"
	"dst-management-platform-api/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v := r.Group(constants.ApiVersion)
	{
		platform := v.Group("platform")
		{
			platform.GET("/status", middleware.MWtoken(), h.statusGet)
		}
	}
}
