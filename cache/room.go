package cache

import "sync"

var (
	// RoomNoPlayersSeconds 房间中没有玩家的秒数，用于自动重置
	RoomNoPlayersSeconds = make(map[int]int)
	// RoomNoPlayersSecondsMutex RoomNoPlayersSeconds锁
	RoomNoPlayersSecondsMutex sync.Mutex
)
