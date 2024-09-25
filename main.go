package main

import (
	"dst-management-platform-api/app/auth"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	static "github.com/soulteary/gin-static"
)

func main() {
	r := gin.Default()

	//全局中间件，获取语言
	r.Use(utils.MWlang())

	//静态资源
	r.Use(static.Serve("/", static.LocalFile("C:/Users/admin/WebstormProjects/dst-management-platform-web/dist", true)))

	//用户、鉴权模块
	r = auth.RouteAuth(r)
	// 启动服务器

	err := r.Run(":7000")
	if err != nil {
		return
	}
}
