package externalApi

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func RouteExternalApi(r *gin.Engine) *gin.Engine {
	v1 := r.Group("v1")
	v1.Use(utils.MWtoken())
	v1.Use(utils.MWUserCheck())
	{
		externalApi := v1.Group("external/api")
		{
			// 获取饥荒最新版本
			externalApi.GET("/dst_version", handleVersionGet)
			// 获取直连代码
			externalApi.GET("/connection_code", handleConnectionCodeGet)
			// 获取模组信息
			externalApi.GET("/mod_info", handleModInfoGet)
			externalApi.GET("/mod_search", handleModSearchGet)
			// 已下载的模组信息
			externalApi.GET("/downloaded/mod_info", handleDownloadedModInfoGet)
		}
	}

	return r
}
