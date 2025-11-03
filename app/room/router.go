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
			room.POST("", middleware.MWtoken(), h.roomPost)
			room.GET("", middleware.MWtoken(), h.roomGet)
			room.GET("/list", middleware.MWtoken(), h.listGet)
			room.GET("/port/factor", middleware.MWtoken(), h.portFactorGet)
		}
	}
}
