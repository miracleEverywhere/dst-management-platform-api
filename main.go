package main

import (
	"dst-management-platform-api/app/external"
	"dst-management-platform-api/app/platform"
	"dst-management-platform-api/app/room"
	"dst-management-platform-api/app/user"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/utils"
	"embed"
	"fmt"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	static "github.com/soulteary/gin-static"
	"runtime"
)

//go:embed dist/*
//go:embed dist/assets/*
var EmbedFS embed.FS

func main() {
	// 绑定启动参数
	utils.BindFlags()

	// 打印版本
	if utils.VersionShow {
		fmt.Println(utils.Version + "\n" + runtime.Version())
		return
	}

	// 初始化日志
	logger.InitLogger()

	// 初始化数据库
	db.InitDB()
	userDao := dao.NewUserDAO(db.DB)
	systemDao := dao.NewSystemDAO(db.DB)
	roomDao := dao.NewRoomDAO(db.DB)
	roomSettingDao := dao.NewRoomSettingDAO(db.DB)
	worldDao := dao.NewWorldDAO(db.DB)
	globalSettingDao := dao.NewGlobalSettingDAO(db.DB)

	// 开启定时任务
	scheduler.Start(roomDao, worldDao, roomSettingDao, globalSettingDao)

	// 初始化及注册路由
	r := gin.Default()
	gin.SetMode(gin.ReleaseMode)
	user.NewHandler(userDao).RegisterRoutes(r)
	room.NewHandler(userDao, roomDao, worldDao, roomSettingDao).RegisterRoutes(r)
	external.NewHandler(userDao, roomDao, worldDao, roomSettingDao).RegisterRoutes(r)
	platform.NewHandler(userDao, systemDao).RegisterRoutes(r)

	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(static.ServeEmbed("dist", EmbedFS))

	// 启动服务器
	err := r.Run(fmt.Sprintf(":%d", utils.BindPort))
	if err != nil {
		logger.Logger.Error("启动服务器失败", "err", err)
		panic(fmt.Sprintf("启动服务器失败: %s", err.Error()))
	}
}
