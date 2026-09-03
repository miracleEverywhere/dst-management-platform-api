package cache

import (
	"dst-management-platform-api/logger"
	"os"

	"github.com/shirou/gopsutil/v3/host"
)

var (
	// CurrentDir 当前工作目录
	CurrentDir string
	// DstUpdating 饥荒更新中
	DstUpdating bool
	// InternetIP 获取外网IP
	InternetIP string
	// GameServerVersion 饥荒的版本号
	GameServerVersion int
	// OsType 当前运行的系统
	OsType string
)

func initCurrentDir() {
	var err error
	CurrentDir, err = os.Getwd()
	if err != nil {
		panic("获取工作路径失败")
	}
}

func initOsType() {
	hostInfo, err := host.Info()
	if err != nil {
		panic("获取系统类型失败：" + err.Error())
	}

	OsType = hostInfo.OS
	logger.Logger.Debugf("当前系统为：%s", OsType)
}
