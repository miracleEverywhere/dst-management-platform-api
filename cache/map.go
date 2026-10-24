package cache

import (
	"context"
	"fmt"
	"sync"
)

// 地图是实时生成的，不做结果缓存；这里只做并发保护：
//  1. 全局信号量，避免同时解析多个世界的存档把 CPU 打满；
//  2. 同一世界串行，避免同一个世界被多个请求同时解析。
//
// 锁用带缓冲 channel 实现，等待时可以响应 context 取消。
const maxMapConcurrent = 2

var (
	mapSemaphore = make(chan struct{}, maxMapConcurrent)
	mapLocks     = struct {
		sync.Mutex
		locks map[string]chan struct{}
	}{
		locks: make(map[string]chan struct{}),
	}
)

// AcquireMap 获取指定房间+世界的生成许可。返回的 release 必须调用一次。
func AcquireMap(ctx context.Context, roomID, worldID int) (func(), error) {
	key := fmt.Sprintf("%d:%d", roomID, worldID)

	mapLocks.Lock()
	lock, ok := mapLocks.locks[key]
	if !ok {
		lock = make(chan struct{}, 1)
		mapLocks.locks[key] = lock
	}
	mapLocks.Unlock()

	select {
	case lock <- struct{}{}:
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	select {
	case mapSemaphore <- struct{}{}:
	case <-ctx.Done():
		<-lock
		return nil, ctx.Err()
	}

	var once sync.Once
	return func() {
		once.Do(func() {
			<-mapSemaphore
			<-lock
		})
	}, nil
}
