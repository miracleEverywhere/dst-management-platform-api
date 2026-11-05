package cloud

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/middleware"

	"github.com/gin-gonic/gin"
)

func (h *Handler) RegisterRoutes(r *gin.Engine) {
	v := r.Group(constant.ApiVersion)
	{
		cloud := v.Group("cloud")
		// 云账号登录是登录入口，必须公开；与本地登录共用用户名校验限流。
		cloud.POST("/login", middleware.LoginRateLimit(), h.loginPost)

		auth := cloud.Group("")
		auth.Use(middleware.TokenCheck())
		{
			auth.GET("/status", h.statusGet)
			auth.POST("/link", h.linkPost)
			auth.POST("/unlink", h.unlinkPost)
			auth.POST("/sync", h.syncPost)
			auth.GET("/favorites", h.favoritesGet)
			auth.GET("/tokens", h.tokensGet)
			auth.GET("/game-admins", h.gameAdminsGet)
			auth.GET("/game-reserved-slots", h.gameSlotsGet)
			auth.POST("/archives/upload", h.archiveUploadPost)
			auth.GET("/archives", h.archivesGet)
			auth.GET("/archives/:id/download", h.archiveDownloadGet)
			auth.POST("/archives/:id/import", h.archiveImportPost)
		}
	}
}
