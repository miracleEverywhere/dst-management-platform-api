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
			setting.POST("/player/delete/admin", utils.MWtoken(), handleAdminDeletePost)
			setting.POST("/player/delete/block", utils.MWtoken(), handleBlockDeletePost)
			setting.POST("/player/delete/white", utils.MWtoken(), handleWhiteDeletePost)
			setting.POST("/player/kick", utils.MWtoken(), handleKick)
			// 存档导入
			setting.POST("/import/upload", utils.MWtoken(), handleImportFileUploadPost)
		}
	}

	return r
}
