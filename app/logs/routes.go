package logs

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func RouteLogs(r *gin.Engine) *gin.Engine {
	v1 := r.Group("v1")
	{
		logs := v1.Group("logs")
		{
			// 获取4种日志
			logs.GET("/log_value", utils.MWtoken(), handleLogGet)
			logs.POST("/process_log", utils.MWtoken(), handleProcessLogPost)
			logs.GET("/historical/log_file", utils.MWtoken(), handleHistoricalLogFileGet)
			logs.GET("/historical/log", utils.MWtoken(), handleHistoricalLogGet)
		}
	}

	return r
}
