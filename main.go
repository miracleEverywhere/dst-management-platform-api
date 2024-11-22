package main

import (
	"dst-management-platform-api/app/auth"
	"dst-management-platform-api/app/externalApi"
	"dst-management-platform-api/app/home"
	"dst-management-platform-api/app/logs"
	"dst-management-platform-api/app/setting"
	"dst-management-platform-api/app/tools"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/utils"
	"embed"
	"flag"
	"fmt"
	"github.com/gin-gonic/gin"
	static "github.com/soulteary/gin-static"
	"io"
	"runtime"
)

const VERSION string = "1.0.5"

var (
	// flag绑定的变量
	bindPort      int
	consoleOutput bool
	versionShow   bool
)

//go:embed dist
var EmbedFS embed.FS

func main() {
	//一些启动前检查
	initialize()

	if !consoleOutput {
		gin.DefaultWriter = io.Discard
	}
	if versionShow {
		fmt.Println(VERSION + "\n" + runtime.Version())
		return
	}

	r := gin.Default()

	//全局中间件，获取语言
	r.Use(utils.MWlang())

	//用户、鉴权模块
	r = auth.RouteAuth(r)
	//主页模块
	r = home.RouteHome(r)
	//设置模块
	r = setting.RouteSetting(r)
	//工具模块
	r = tools.RouteTools(r)
	//工具模块
	r = logs.RouteLogs(r)
	// 外部接口
	r = externalApi.RouteExternalApi(r)

	//静态资源，放在最后
	r.Use(static.ServeEmbed("dist", EmbedFS))

	// 启动服务器
	err := r.Run(fmt.Sprintf(":%d", bindPort))
	if err != nil {
		utils.Logger.Error("启动服务器失败", "err", err)
		panic(err)
	}
}

func initialize() {
	flag.IntVar(&bindPort, "l", 80, "监听端口，如： -l 8080 (Listening Port, e.g. -l 8080)")
	flag.BoolVar(&consoleOutput, "c", false, "开启控制台日志输出，如： -c (Enable console log output, e.g. -c)")
	flag.BoolVar(&versionShow, "v", false, "查看版本，如： -v (Check version, e.g. -v)")
	flag.Parse()

	//数据库检查
	utils.CreateConfig()

	//创建手动安装脚本
	err := utils.TruncAndWriteFile("manual_install.sh", utils.ManualInstall)
	if err != nil {
		utils.Logger.Error("手动安装脚本创建失败", "err", err)
	}
	err = utils.BashCMD("chmod +x manual_install.sh")
	if err != nil {
		utils.Logger.Error("手动安装脚本添加执行权限失败", "err", err)
	}

	//gin.SetMode(gin.DebugMode)
	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()

	//加载定时任务
	scheduler.InitTasks()
	//启动定时任务调度器
	go scheduler.Scheduler.StartAsync()
	utils.Logger.Info("定时任务已启动")
}
