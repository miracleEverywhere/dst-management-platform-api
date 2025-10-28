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
			room.POST("/create", middleware.MWtoken(), h.createPost)
		}
	}
}
