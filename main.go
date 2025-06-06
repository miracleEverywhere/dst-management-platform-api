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
	"fmt"
	"github.com/gin-gonic/gin"
	static "github.com/soulteary/gin-static"
	"io"
	"runtime"
)

//go:embed dist
var EmbedFS embed.FS

func main() {
	if !utils.ConsoleOutput {
		gin.DefaultWriter = io.Discard
	}
	if utils.VersionShow {
		fmt.Println(utils.VERSION + "\n" + runtime.Version())
		return
	}

	r := gin.Default()
	// 全局中间件，获取语言
	r.Use(utils.MWlang())
	// 用户、鉴权模块
	r = auth.RouteAuth(r)
	// 主页模块
	r = home.RouteHome(r)
	// 设置模块
	r = setting.RouteSetting(r)
	// 工具模块
	r = tools.RouteTools(r)
	// 日志模块
	r = logs.RouteLogs(r)
	// 外部接口
	r = externalApi.RouteExternalApi(r)
	// 备份下载
	r.Static("/v1/download", "./dmp_files/backup")
	// 静态资源，放在最后
	r.Use(static.ServeEmbed("dist", EmbedFS))

	// 启动服务器
	err := r.Run(fmt.Sprintf(":%d", utils.BindPort))
	if err != nil {
		utils.Logger.Error("启动服务器失败", "err", err)
		panic(err)
	}
}

func init() {
	// 绑定flag
	utils.BindFlags()
	// 数据库检查
	utils.CheckConfig()
	// 设置全局变量
	utils.SetGlobalVariables()
	// 检查目录
	utils.CheckDirs()
	// 创建DST手动安装脚本
	utils.CreateManualInstallScript()
	gin.SetMode(gin.ReleaseMode)
	gin.DisableConsoleColor()
	// 加载定时任务
	scheduler.InitTasks()
	// 启动定时任务调度器
	go scheduler.Scheduler.StartAsync()
}
