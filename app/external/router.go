package external

import (
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v := r.Group(utils.ApiVersion)
	{
		external := v.Group("external")
		{
			external.GET("/mod/search", middleware.MWtoken(), modSearchGet)
		}
	}
}
