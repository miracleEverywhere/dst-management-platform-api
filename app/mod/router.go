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
			mod.POST("/add/enable", middleware.MWtoken(), h.addEnablePost)
			mod.GET("/setting/mod_config_struct", middleware.MWtoken(), h.settingModConfigStructGet)
			mod.GET("/setting/mod_config_value", middleware.MWtoken(), h.settingModConfigValueGet)
			mod.PUT("/setting/mod_config_value", middleware.MWtoken(), h.settingModConfigValuePut)
			mod.GET("/setting/enabled", middleware.MWtoken(), h.getEnabledModsGet)
		}
	}
}
