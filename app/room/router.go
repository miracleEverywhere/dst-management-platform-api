package room

import (
	"dst-management-platform-api/constants"
	"dst-management-platform-api/middleware"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v := r.Group(constants.ApiVersion)
	{
		room := v.Group("room")
		{
			room.POST("/base", middleware.MWtoken(), h.basePost)
			room.DELETE("/base", middleware.MWtoken(), h.baseDelete)
			room.GET("/list", middleware.MWtoken(), h.listGet)
		}
	}
}
