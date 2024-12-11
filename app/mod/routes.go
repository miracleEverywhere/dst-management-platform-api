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
			mod.GET("/setting", utils.MWtoken(), handleModSettingGet)
			mod.GET("/info", utils.MWtoken(), handleModInfoGet)
		}
	}

	return r
}
