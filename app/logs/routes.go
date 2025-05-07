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
			// ！！！！注意！！！！ v1.1.10
			// 此接口原本是下载process日志的，现在改为下载所有日志
			// 考虑到新老接口调用，因此不修改接口的url
			//logs.POST("/process_log", handleLogDownloadPost)
			logs.GET("/historical/log_file", handleHistoricalLogFileGet)
			logs.GET("/historical/log", handleHistoricalLogGet)
			//// 日志清理
			logs.GET("/status", handleGetLogInfoGet)
			logs.POST("/clean", handleCleanLogsPost)
		}
	}

	return r
}
