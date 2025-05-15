package logs

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
)

func RouteLogs(r *gin.Engine) *gin.Engine {
	v1 := r.Group("v1")
	v1.Use(utils.MWtoken())
	v1.Use(utils.MWUserCheck())
	{
		logs := v1.Group("logs")
		{
			// 获取4种日志
			logs.GET("/log_value", handleLogGet)
			logs.POST("/download", handleLogDownloadPost)
			logs.GET("/historical/log_file", handleHistoricalLogFileGet)
			logs.GET("/historical/log", handleHistoricalLogGet)
			//// 日志清理
			logs.GET("/status", handleGetLogInfoGet)
			logs.POST("/clean", handleCleanLogsPost)
		}
	}

	return r
}
