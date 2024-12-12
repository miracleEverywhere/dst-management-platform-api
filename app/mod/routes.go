package mod

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func RouteMod(r *gin.Engine) *gin.Engine {
	v1 := r.Group("v1")
	{
		mod := v1.Group("mod")
		{
			mod.GET("/setting/format", utils.MWtoken(), handleModSettingFormatGet)
			mod.GET("/config_options", utils.MWtoken(), handleModConfigOptionsGet)
		}
	}

	return r
}
