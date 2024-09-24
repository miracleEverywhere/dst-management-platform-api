package main

import (
	"dst-management-platform-api/app/auth"
	"github.com/gin-gonic/gin"
	static "github.com/soulteary/gin-static"
)

func main() {
	r := gin.Default()

	//r.Static("/dist", "web")
	//r.Static("/assets", "web/assets")
	r.Use(static.Serve("/", static.LocalFile("C:/Users/admin/WebstormProjects/dst-management-platform-web/dist", true)))
	// 在主路径返回 index.html
	r.GET("/", func(c *gin.Context) {
		c.File("web/index.html")
	})
	r = auth.RouteAuth(r)
	// 启动服务器
	err := r.Run(":7000")
	if err != nil {
		return
	} // 在7000端口启动
}
