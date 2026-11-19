package tools

import (
	"sync"
	"time"
)

// lobbyCache 区域级大厅房间列表缓存：命中直接内存过滤分页，未命中才拉取 Klei 接口。
// 每个区域单独一把锁，同一区域同一时刻只有一个请求真正下载，其余请求等待并复用结果。
type lobbyCache struct {
	mu      sync.Mutex
	entries map[string]*lobbyRegionCache
}

// lobbyRegionCache 单个区域的缓存条目
type lobbyRegionCache struct {
	mu        sync.Mutex
	rooms     []LobbyRoom
	fetchedAt time.Time
}

func newLobbyCache() *lobbyCache {
	return &lobbyCache{
		entries: make(map[string]*lobbyRegionCache),
	}
}

// entry 获取区域缓存条目
func (c *lobbyCache) entry(region string) *lobbyRegionCache {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[region]
	if !ok {
		entry = &lobbyRegionCache{}
		c.entries[region] = entry
	}

	return entry
}

// rooms 读取区域房间列表：
//   - 命中 TTL 且未强制刷新时直接返回缓存；
//   - 强制刷新距上次拉取不足最小间隔时退化为普通读取；
//   - 拉取失败但存在旧缓存时返回旧缓存与错误，由调用方标记该区域刷新失败。
func (c *lobbyCache) rooms(region string, refresh bool, fetch func(region string) ([]LobbyRoom, error)) ([]LobbyRoom, time.Time, error) {
	entry := c.entry(region)
	entry.mu.Lock()
	defer entry.mu.Unlock()

	now := time.Now()
	if refresh && !entry.fetchedAt.IsZero() && now.Sub(entry.fetchedAt) < lobbyMinRefreshInterval {
		refresh = false
	}

	if !refresh && entry.rooms != nil && now.Sub(entry.fetchedAt) < lobbyCacheTTL {
		return entry.rooms, entry.fetchedAt, nil
	}

	rooms, err := fetch(region)
	if err != nil {
		if entry.rooms != nil {
			return entry.rooms, entry.fetchedAt, err
		}

		return nil, time.Time{}, err
	}

	entry.rooms = rooms
	entry.fetchedAt = now

	return entry.rooms, entry.fetchedAt, nil
}
