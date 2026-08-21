package cache

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"os"
)

// InitCache 初始化缓存
func InitCache() {
	initCurrentDir()
	initCustomGameStartupCmd()
	initModDownloadStatus()
}

func initCurrentDir() {
	var err error
	CurrentDir, err = os.Getwd()
	if err != nil {
		panic("获取工作路径失败")
	}
}

func initCustomGameStartupCmd() {
	var globalSetting models.GlobalSetting
	if err := db.DB.First(&globalSetting).Error; err != nil {
		panic("初始化自定义游戏启动命令失败: " + err.Error())
	}
	SetCustomGameStartupCmd(globalSetting.CustomStartupCmd)
}

func initModDownloadStatus() {
	ModDownloadStatus = NewModCache()
}
