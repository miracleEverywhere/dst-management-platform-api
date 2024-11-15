package externalApi

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func RouteExternalApi(r *gin.Engine) *gin.Engine {
	v1 := r.Group("v1")
	{
		externalApi := v1.Group("external/api")
		{
			// 获取饥荒最新版本
			externalApi.GET("/dst_version", utils.MWtoken(), handleVersionGet)
			// 获取直连代码
			externalApi.GET("/connection_code", utils.MWtoken(), handleConnectionCodeGet)
		}
	}

	return r
}
