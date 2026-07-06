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
	"dst-management-platform-api/webhook"
	"fmt"
	"runtime"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	static "github.com/soulteary/gin-static"
)

func Run() {
	// 绑定启动参数
	bindFlags()

	if err := utils.InitWorkDir(workDir); err != nil {
		panic(fmt.Sprintf("初始化工作目录失败: %s", err.Error()))
	}
	db.CurrentDir = utils.WorkDir

	// 打印版本
	if versionShow {
		fmt.Println(utils.Version + "\n" + runtime.Version())
		return
	}

	// 控制台命令
	if consoleCmd != "" {
		runConsole(consoleCmd, dbPath)
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
	pluginDao := dao.NewPluginDAO(db.DB)
	dstImageDao := dao.NewDstImageDAO(db.DB)

	// 初始化 webhook sender
	webhook.Snd = webhook.NewSender(globalSettingDao, roomSettingDao, roomDao)

	// 开启定时任务
	scheduler.Start(roomDao, worldDao, roomSettingDao, globalSettingDao, uidMapDao)

	r := gin.New()

	// 请求日志格式
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		Formatter: logger.AccessFormatter,
		Output:    logger.AccessWriter,
	}))
	// panic恢复，将panic日志写入runtime.log
	r.Use(gin.CustomRecoveryWithWriter(logger.RuntimeWriter, func(c *gin.Context, recovered interface{}) {
		logger.Logger.Errorf("panic recovered: %v", recovered)
		c.AbortWithStatus(500)
	}))
	// 静态资源缓存
	r.Use(middleware.CacheControl())

	// debug日志等级下，注册pprof路由
	if logLevel == "debug" {
		logger.Logger.Debug("debug模式已开启")
		logger.Logger.Warn("debug模式会无条件暴露各种运行数据，请勿在生产环境开启debug")
		pprof.Register(r)
	} else {
		// 设置生产环境
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化即注册路由
	user.NewHandler(userDao, pluginDao).RegisterRoutes(r)
	room.NewHandler(userDao, roomDao, worldDao, roomSettingDao, globalSettingDao, uidMapDao).RegisterRoutes(r)
	mod.NewHandler(roomDao, worldDao, roomSettingDao, userDao).RegisterRoutes(r)
	dashboard.NewHandler(userDao, roomDao, worldDao, roomSettingDao, globalSettingDao).RegisterRoutes(r)
	platform.NewHandler(userDao, roomDao, worldDao, systemDao, globalSettingDao, uidMapDao, roomSettingDao, pluginDao, dstImageDao).RegisterRoutes(r)
	logs.NewHandler(userDao, roomDao, worldDao, roomSettingDao).RegisterRoutes(r)
	tools.NewHandler(userDao, roomDao, worldDao, roomSettingDao, dstImageDao).RegisterRoutes(r)
	player.NewHandler(userDao, roomDao, worldDao, roomSettingDao, uidMapDao, globalSettingDao).RegisterRoutes(r)

	r.Use(static.ServeEmbed("dist", embedFS.Dist))

	// 启动服务器
	var err error
	if cert != "" && key != "" {
		// 证书文件和私钥文件都不为空，则启动https
		err = r.RunTLS(fmt.Sprintf(":%d", bindPort), cert, key)
	} else {
		// 否则启动http
		err = r.Run(fmt.Sprintf(":%d", bindPort))
	}
	if err != nil {
		panic(fmt.Sprintf("启动服务器失败: %s", err.Error()))
	}
}
