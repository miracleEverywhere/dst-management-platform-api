package cache

import (
	"fmt"
	"sync"
)

type ModItem struct {
	ID          int `json:"id"`
	Size        int `json:"size"`
	CurrentSize int `json:"currentSize"`
}

type ModCache struct {
	mu   sync.RWMutex
	mods map[int][]*ModItem
}

// NewModCache 创建新的缓存实例
func NewModCache() *ModCache {
	return &ModCache{
		mods: make(map[int][]*ModItem),
	}
}

// Set 设置或更新缓存项
func (m *ModCache) Set(roomID int, item *ModItem) error {
	if m.mods == nil {
		return fmt.Errorf("缓存未初始化")
	}
	if item == nil {
		return fmt.Errorf("item 不能为 nil")
	}
	if roomID <= 0 || item.ID <= 0 {
		return fmt.Errorf("参数非法: roomID=%d, modID=%d", roomID, item.ID)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否已存在，存在则更新
	items, exists := m.mods[roomID]
	if exists {
		for i, existing := range items {
			if existing.ID == item.ID {
				// 更新已存在的项
				items[i] = item
				return nil
			}
		}
	}

	// 不存在则追加
	m.mods[roomID] = append(m.mods[roomID], item)
	return nil
}

// Get 根据roomID和modID获取缓存项
func (m *ModCache) Get(roomID, modID int) (*ModItem, error) {
	if m.mods == nil {
		return nil, fmt.Errorf("缓存未初始化")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	items, exists := m.mods[roomID]
	if !exists {
		return nil, fmt.Errorf("房间 %d 不存在于缓存中", roomID)
	}

	for _, item := range items {
		if item.ID == modID {
			return item, nil
		}
	}

	return nil, fmt.Errorf("模组 %d 不存在于房间 %d 中", modID, roomID)
}

// Delete 删除缓存项
func (m *ModCache) Delete(roomID, modID int) error {
	if m.mods == nil {
		return fmt.Errorf("缓存未初始化")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	items, exists := m.mods[roomID]
	if !exists {
		return fmt.Errorf("房间 %d 不存在于缓存中", roomID)
	}

	for i, item := range items {
		if item.ID == modID {
			// 删除该元素
			m.mods[roomID] = append(items[:i], items[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("模组 %d 不存在于房间 %d 中", modID, roomID)
}
