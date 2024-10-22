package setting

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func RouteSetting(r *gin.Engine) *gin.Engine {
	v1 := r.Group("v1")
	{
		setting := v1.Group("setting")
		{
			// 设置
			setting.GET("/room", utils.MWtoken(), handleRoomSettingGet)
			setting.POST("/save", utils.MWtoken(), handleRoomSettingSavePost)
			setting.POST("/save_restart", utils.MWtoken(), handleRoomSettingSaveAndRestartPost)
			setting.POST("/save_generate", utils.MWtoken(), handleRoomSettingSaveAndGeneratePost)
		}
	}

	return r
}
