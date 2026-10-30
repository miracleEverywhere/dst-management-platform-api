package lite

import (
	"dst-management-platform-api/lite/api"
	"dst-management-platform-api/lite/utils"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
)

// frontendDirName 前端产物目录名，与 dmp-lite 可执行文件同级
const frontendDirName = "dist-lite"

var bindPort int

func Run() {
	debug.SetGCPercent(20)

	flag.IntVar(&bindPort, "bind", 9999, "DMP端口, 如: -bind 9999")
	flag.Parse()

	if err := utils.DB.Read(); err != nil {
		utils.Logger.Error("读取数据库失败: " + err.Error())
		return
	}

	stopScheduler := utils.Start()
	defer stopScheduler()

	staticDir := frontendDir()

	mux := http.NewServeMux()
	api.RegisterRoutes(mux)
	api.RegisterStatic(mux, staticDir)

	utils.Logger.Info(fmt.Sprintf("DMP lite 已启动，监听端口: %d，前端目录: %s", bindPort, staticDir))

	if err := http.ListenAndServe(fmt.Sprintf(":%d", bindPort), mux); err != nil {
		panic(fmt.Sprintf("启动服务器失败: %s", err.Error()))
	}
}

// frontendDir 返回与 dmp-lite 可执行文件同级的 dist-lite 目录，并检查入口文件是否存在。
func frontendDir() string {
	exePath, err := os.Executable()
	if err != nil {
		// 兜底：按当前工作目录解析
		utils.Logger.Error("获取程序路径失败，按当前目录查找前端文件: " + err.Error())
	}

	dir := filepath.Join(filepath.Dir(exePath), frontendDirName)
	indexPath := filepath.Join(dir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		utils.Logger.Error("未找到前端入口文件 " + indexPath + "，请将 " + frontendDirName + " 目录放到 dmp-lite 同目录")
	}

	return dir
}
