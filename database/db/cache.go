package db

import (
	"os"
	"sync"
)

var (
	// JwtSecret jwt密钥
	JwtSecret string
	// CurrentDir 当前工作目录
	CurrentDir string
	// DstUpdating 饥荒更新中
	DstUpdating bool
	// PlayersStatistic 玩家统计
	PlayersStatistic = make(map[int][]Players)
	// PlayersStatisticMutex 玩家统计锁
	PlayersStatisticMutex sync.Mutex
	// PlayersOnlineTime 玩家在线时长
	PlayersOnlineTime = make(map[int]map[string]int)
	// PlayersOnlineTimeMutex 玩家在线时长锁
	PlayersOnlineTimeMutex sync.Mutex
	// SystemMetrics 系统监控数据
	SystemMetrics []SysMetrics
	// InternetIP 获取外网IP
	InternetIP string
	// ModDownloadExecuting 如果没有模组正在下载(==0)，则执行临时模组文件清理任务 scheduler/global.go ModDownloadClean()
	ModDownloadExecuting int32
)

type PlayerInfo struct {
	UID      string `json:"uid"`
	Nickname string `json:"nickname"`
	Prefab   string `json:"prefab"`
}

type Players struct {
	PlayerInfo []PlayerInfo `json:"playerInfo"`
	Timestamp  int64        `json:"timestamp"`
}

type SysMetrics struct {
	Timestamp   int64   `json:"timestamp"`
	Cpu         float64 `json:"cpu"`
	Memory      float64 `json:"memory"`
	NetUplink   float64 `json:"netUplink"`
	NetDownlink float64 `json:"netDownlink"`
	Disk        float64 `json:"disk"`
}

func init() {
	setCurrentDir()
}

func setCurrentDir() {
	var err error
	CurrentDir, err = os.Getwd()
	if err != nil {
		panic("获取工作路径失败")
	}
}
