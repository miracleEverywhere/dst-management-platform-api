package cache

import "sync"

type PlayerInfo struct {
	UID      string `json:"uid"`
	Nickname string `json:"nickname"`
	Prefab   string `json:"prefab"`
}

type Players struct {
	PlayerInfo []PlayerInfo `json:"playerInfo"`
	Timestamp  int64        `json:"timestamp"`
}

var (
	// PlayersStatistic 玩家统计
	PlayersStatistic = make(map[int][]Players)
	// PlayersStatisticMutex 玩家统计锁
	PlayersStatisticMutex sync.Mutex
	// PlayersOnlineTime 玩家在线时长
	PlayersOnlineTime = make(map[int]map[string]int)
	// PlayersOnlineTimeMutex 玩家在线时长锁
	PlayersOnlineTimeMutex sync.Mutex
)
