package cache

import (
	"os"
)

// InitCache 初始化缓存
func InitCache() {
	initCurrentDir()
	initModDownloadStatus()
}

func initCurrentDir() {
	var err error
	CurrentDir, err = os.Getwd()
	if err != nil {
		panic("获取工作路径失败")
	}
}

func initModDownloadStatus() {
	ModDownloadStatus = NewModCache()
}
