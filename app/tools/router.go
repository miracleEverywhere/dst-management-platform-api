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
		// 使用浏览器A标签Get，无法带header认证 ，token验证放到handle里面执行
		// 暂时需要.zip后缀，否则会进行Gzip压缩
		tools.GET("/backup/download.zip", h.backupDownloadGet)
		tools.Use(middleware.TokenCheck())
		{
			tools.GET("/backup", h.backupGet)
			tools.POST("/backup", h.backupPost)
			tools.DELETE("/backup", h.backupDelete)
			tools.POST("/backup/restore", h.backupRestorePost)
			tools.GET("/announce", h.announceGet)
			tools.PUT("/announce", h.announcePut)
			tools.GET("/map", h.mapGet)
			tools.POST("/token", middleware.AdminOnly(), tokenPost)
			tools.GET("/snapshot", h.snapshotGet)
			tools.DELETE("/snapshot", h.snapshotDelete)
			tools.GET("/tmi/category", h.categoryGet)
			tools.GET("/tmi/category/items", h.categoryItemsGet)
			tools.POST("/tmi/console", h.consolePost)
			tools.GET("/aichat/setting", h.aiSettingGet)
			tools.PUT("/aichat/setting", h.aiSettingPut)
			tools.POST("/aichat/keyword/rebuild", middleware.AdminOnly(), h.aiKeywordIndexReBuild)
			tools.POST("/aichat/embedding/rebuild", middleware.AdminOnly(), h.aiEmbeddingIndexReBuild)
			tools.GET("/aichat/setting/base", middleware.AdminOnly(), h.aiBaseSettingGet)
			tools.PUT("/aichat/setting/base", middleware.AdminOnly(), h.aiBaseSettingPut)
		}
	}
}
