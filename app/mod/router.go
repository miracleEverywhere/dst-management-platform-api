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
		mod.Use(middleware.TokenCheck())
		{
			mod.GET("/search", modSearchGet)
			mod.POST("/download", h.downloadPost)
			mod.GET("/downloaded", h.downloadedModsGet)
			mod.POST("/add/enable", h.addEnablePost)
			mod.POST("/setting/disable", h.addDisablePost)
			mod.GET("/setting/mod_config_struct", h.settingModConfigStructGet)
			mod.GET("/setting/mod_config_value", h.settingModConfigValueGet)
			mod.PUT("/setting/mod_config_value", h.settingModConfigValuePut)
			mod.GET("/setting/enabled", h.getEnabledModsGet)
			mod.POST("/delete", h.deletePost)
		}
	}
}
