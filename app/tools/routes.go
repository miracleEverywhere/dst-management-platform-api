package tools

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func RouteTools(r *gin.Engine) *gin.Engine {
	v1 := r.Group("v1")
	{
		tools := v1.Group("tools")
		{
			// 设置
			tools.GET("/os_info", utils.MWtoken(), handleOSInfoGet)
		}
	}

	return r
}
