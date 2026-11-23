package tools

import (
	"dst-management-platform-api/logger"
	"sync"
	"time"
)

// lobbyCache 区域级大厅房间列表缓存：命中直接内存过滤分页，未命中才拉取 Klei 接口。
//
// Klei CDN 的区域列表压缩后 0.1~2MB、解压后可达 12MB，跨境慢链路上单区域下载实测 2~30 秒，
// 因此除缓存外还做了「先返回旧数据、后台刷新」（stale-while-revalidate）：
// 只要该区域还有一份旧数据，请求就不会阻塞在下载上。
type lobbyCache struct {
	mu      sync.Mutex
	entries map[string]*lobbyRegionCache
}

// lobbyRegionCache 单个区域的缓存条目
type lobbyRegionCache struct {
	mu        sync.Mutex
	rooms     []LobbyRoom
	fetchedAt time.Time
	// refreshing 标记该区域是否已有刷新（同步或后台）在进行，保证同一区域同一时刻只有一次下载
	refreshing bool
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
//   - 没有任何缓存时同步下载，同区域的并发请求在区域锁上等待并复用第一次的结果；
//   - 命中 TTL 且未强制刷新时直接返回缓存；
//   - 缓存已过期时先返回旧缓存，再后台刷新；
//   - 用户主动刷新（refresh=true）且距上次下载超过最小间隔时同步等待新数据，
//     下载失败则回退旧缓存并返回错误，由调用方标记该区域刷新失败。
func (c *lobbyCache) rooms(region string, refresh bool, fetch func(region string) ([]LobbyRoom, error)) ([]LobbyRoom, time.Time, error) {
	entry := c.entry(region)

	// 冷启动：没有任何缓存时必须同步下载；期间的并发请求在区域锁上等待，不会重复下载
	entry.mu.Lock()
	if entry.rooms == nil {
		rooms, err := fetch(region)
		if err != nil {
			entry.mu.Unlock()
			return nil, time.Time{}, err
		}
		entry.rooms, entry.fetchedAt = rooms, time.Now()
		cached, fetchedAt := entry.rooms, entry.fetchedAt
		entry.mu.Unlock()

		return cached, fetchedAt, nil
	}
	age := time.Since(entry.fetchedAt)
	entry.mu.Unlock()

	cached, fetchedAt := entry.snapshot()

	// 强制刷新在最小间隔内退化为普通读取，避免连点刷新反复下载
	if refresh && age >= lobbyMinRefreshInterval {
		performed, err := entry.refreshSync(region, fetch)
		if performed && err != nil {
			return cached, fetchedAt, err
		}
		cached, fetchedAt = entry.snapshot()

		return cached, fetchedAt, nil
	}

	if age < lobbyCacheTTL {
		return cached, fetchedAt, nil
	}

	// 缓存已过期：先返回旧数据，后台刷新，避免请求阻塞在下载上
	entry.refreshAsync(region, fetch)

	return cached, fetchedAt, nil
}

// snapshot 读取当前缓存快照
func (e *lobbyRegionCache) snapshot() ([]LobbyRoom, time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()

	return e.rooms, e.fetchedAt
}

// refreshSync 同步刷新：已有刷新在进行时不重复下载，返回 performed=false
func (e *lobbyRegionCache) refreshSync(region string, fetch func(region string) ([]LobbyRoom, error)) (bool, error) {
	if !e.beginRefresh() {
		return false, nil
	}

	rooms, err := fetch(region)
	if err == nil {
		e.mu.Lock()
		e.rooms, e.fetchedAt = rooms, time.Now()
		e.mu.Unlock()
	}
	e.finishRefresh()

	return true, err
}

// refreshAsync 后台刷新：已有刷新在进行时直接跳过，下载结果写回缓存
func (e *lobbyRegionCache) refreshAsync(region string, fetch func(region string) ([]LobbyRoom, error)) {
	if !e.beginRefresh() {
		return
	}

	go func() {
		rooms, err := fetch(region)
		if err != nil {
			logger.Logger.Warnf("后台刷新游戏大厅房间列表失败, region: %s, err: %v", region, err)
		} else {
			e.mu.Lock()
			e.rooms, e.fetchedAt = rooms, time.Now()
			e.mu.Unlock()
		}

		e.finishRefresh()
	}()
}

// beginRefresh 抢占刷新权，已被占用时返回 false
func (e *lobbyRegionCache) beginRefresh() bool {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.refreshing {
		return false
	}
	e.refreshing = true

	return true
}

// finishRefresh 释放刷新权
func (e *lobbyRegionCache) finishRefresh() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.refreshing = false
}
