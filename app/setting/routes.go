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
			setting.POST("/room/save", utils.MWtoken(), handleRoomSettingSavePost)
			setting.POST("/room/save_restart", utils.MWtoken(), handleRoomSettingSaveAndRestartPost)
			setting.POST("/room/save_generate", utils.MWtoken(), handleRoomSettingSaveAndGeneratePost)
			// Player
			setting.GET("/player/list", utils.MWtoken(), handlePlayerListGet)
			setting.POST("/player/add/admin", utils.MWtoken(), handleAdminAddPost)
			setting.POST("/player/add/block", utils.MWtoken(), handleBlockAddPost)
			setting.POST("/player/add/white", utils.MWtoken(), handleWhiteAddPost)
		}
	}

	return r
}
