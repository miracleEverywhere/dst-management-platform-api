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
		{
			dashboard.POST("/exec/game", middleware.MWtoken(), h.execGamePost)
		}
	}
}
