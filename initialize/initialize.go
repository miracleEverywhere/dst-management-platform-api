package initialize

import (
	"dst-management-platform-api/db"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
)

func Initialize() {
	// 绑定启动参数
	utils.BindFlags()
	// 初始化日志
	logger.InitLogger()
	// 初始化数据库
	db.InitDB()
}
