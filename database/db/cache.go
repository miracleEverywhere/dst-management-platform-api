package db

import (
	"os"
	"sync"
)

var (
	JwtSecret             string
	CurrentDir            string
	DstUpdating           bool
	PlayersStatistic      = make(map[int][]Players)
	PlayersStatisticMutex sync.Mutex
	SystemMetrics         []SysMetrics
	InternetIP            string
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
