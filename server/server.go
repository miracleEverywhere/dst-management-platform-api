package server

import (
	"dst-management-platform-api/app/dashboard"
	"dst-management-platform-api/app/logs"
	"dst-management-platform-api/app/mod"
	"dst-management-platform-api/app/platform"
	"dst-management-platform-api/app/player"
	"dst-management-platform-api/app/room"
	"dst-management-platform-api/app/tools"
	"dst-management-platform-api/app/user"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/embedFS"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/utils"
	"fmt"
	"runtime"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	static "github.com/soulteary/gin-static"
)

func Run() {
	// 绑定启动参数
	bindFlags()

	// 打印版本
	if versionShow {
		fmt.Println(utils.Version + "\n" + runtime.Version())
		return
	}

	// 初始化日志
	logger.InitLogger(logLevel)

	// 初始化文件
	embedFS.GenerateDefaultFile()

	// 初始化数据库
	db.InitDB(dbPath)
	userDao := dao.NewUserDAO(db.DB)
	systemDao := dao.NewSystemDAO(db.DB)
	roomDao := dao.NewRoomDAO(db.DB)
	roomSettingDao := dao.NewRoomSettingDAO(db.DB)
	worldDao := dao.NewWorldDAO(db.DB)
	globalSettingDao := dao.NewGlobalSettingDAO(db.DB)
	uidMapDao := dao.NewUidMapDAO(db.DB)

	// 开启定时任务
	scheduler.Start(roomDao, worldDao, roomSettingDao, globalSettingDao, uidMapDao)

	// 初始化及注册路由
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()
	r.Use(middleware.CacheControl())

	// bug日志等级下，注册pprof路由
	if logLevel == "debug" {
		pprof.Register(r)
	}

	user.NewHandler(userDao).RegisterRoutes(r)
	room.NewHandler(userDao, roomDao, worldDao, roomSettingDao, globalSettingDao, uidMapDao).RegisterRoutes(r)
	mod.NewHandler(roomDao, worldDao, roomSettingDao).RegisterRoutes(r)
	dashboard.NewHandler(userDao, roomDao, worldDao, roomSettingDao, globalSettingDao).RegisterRoutes(r)
	platform.NewHandler(userDao, roomDao, worldDao, systemDao, globalSettingDao, uidMapDao, roomSettingDao).RegisterRoutes(r)
	logs.NewHandler(userDao, roomDao, worldDao, roomSettingDao).RegisterRoutes(r)
	tools.NewHandler(userDao, roomDao, worldDao, roomSettingDao).RegisterRoutes(r)
	player.NewHandler(userDao, roomDao, worldDao, roomSettingDao, uidMapDao).RegisterRoutes(r)

	r.Use(static.ServeEmbed("dist", embedFS.Dist))

	// 启动服务器
	err := r.Run(fmt.Sprintf(":%d", bindPort))
	if err != nil {
		panic(fmt.Sprintf("启动服务器失败: %s", err.Error()))
	}
}
