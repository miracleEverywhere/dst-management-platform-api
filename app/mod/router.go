package mod

import (
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v := r.Group(utils.ApiVersion)
	{
		mod := v.Group("mod")
		{
			mod.GET("/search", middleware.MWtoken(), modSearchGet)
			mod.POST("/download", middleware.MWtoken(), h.downloadPost)
			mod.GET("/downloaded", middleware.MWtoken(), h.downloadedModsGet)
			mod.GET("/setting", middleware.MWtoken(), h.settingGet)
		}
	}
}
