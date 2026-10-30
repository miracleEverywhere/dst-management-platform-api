//go:build !lite

package main

import (
	"dst-management-platform-api/logger"
	"dst-management-platform-api/server"
)

func main() {
	// 捕获 main goroutine 中未被处理的 panic，把 panic 日志与调用栈写入 runtime 日志
	defer logger.HandlePanic()
	// 正常退出前刷盘日志
	defer logger.Sync()

	server.Run()
}
