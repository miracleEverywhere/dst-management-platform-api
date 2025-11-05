package room

import (
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v := r.Group(utils.ApiVersion)
	{
		room := v.Group("room")
		{
			room.POST("", middleware.MWtoken(), h.roomPost)
			room.PUT("", middleware.MWtoken(), h.roomPut)
			room.GET("", middleware.MWtoken(), h.roomGet)
			room.GET("/list", middleware.MWtoken(), h.listGet)
			room.GET("/factor", middleware.MWtoken(), h.factorGet)
		}
	}
}
