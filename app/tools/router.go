package tools

import (
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/utils"

	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v := r.Group(utils.ApiVersion)
	{
		tools := v.Group("tools")
		tools.Use(middleware.TokenCheck())
		{
			tools.GET("/backup", h.backupGet)
			tools.POST("/backup", h.backupPost)
			tools.DELETE("/backup", h.backupDelete)
			tools.POST("/backup/restore", h.backupRestorePost)
			tools.GET("/backup/download", h.backupDownloadGet)
			tools.GET("/announce", h.announceGet)
			tools.PUT("/announce", h.announcePut)
			tools.GET("/map", h.mapGet)
			tools.POST("/token", middleware.AdminOnly(), tokenPost)
			tools.GET("/snapshot", h.snapshotGet)
			tools.DELETE("/snapshot", h.snapshotDelete)
		}
	}
}
