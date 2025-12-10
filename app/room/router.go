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
		room.Use(middleware.MWtoken())
		{
			room.POST("", h.roomPost)
			room.PUT("", h.roomPut)
			room.GET("", h.roomGet)
			room.GET("/list", h.listGet)
			room.GET("/factor", h.factorGet)
			room.GET("/basic", h.allRoomBasicGet)
			room.GET("/worlds", h.roomWorldsGet)
			room.POST("/upload", h.uploadPost)
		}
	}
}
