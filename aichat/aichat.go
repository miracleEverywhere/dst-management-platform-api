package aichat

import (
	"context"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"sync"
)

// ================================================================
// 导出类型
// ================================================================

// Manager AI 对话服务管理器，是 aichat 包的核心入口。
// 负责管理游戏内 AI 聊天的工作协程、Wiki 知识库搜索（关键词+向量）。
type Manager struct {
	roomAISettingDao *dao.RoomAISettingDAO
	systemDao        *dao.SystemDAO
	pluginDao        *dao.PluginDAO
	client           *aiClient
	ctx              context.Context
	cancel           context.CancelFunc
	lifecycle        sync.Mutex
	mu               sync.Mutex
	workers          map[int]*roomWorker
	active           bool
	closed           bool

	keywordSearcher *keywordWikiSearcher
	embedSearcher   *embeddingWikiSearcher
	embedSearcherMu sync.Mutex
	lastEmbedConfig string
}

// EmbeddingConfig 向量嵌入模型配置
type EmbeddingConfig struct {
	APIURL     string
	APIKey     string
	Model      string
	Dimensions int
}

// EmbeddingStats 向量索引统计信息
type EmbeddingStats struct {
	TotalDocs       int            `json:"totalDocs"`
	TotalCategories int            `json:"totalCategories"`
	VectorDim       int            `json:"vectorDim"`
	Categories      map[string]int `json:"categories"`
}

// NewManager 创建 AI 对话服务管理器
func NewManager(roomAISettingDao *dao.RoomAISettingDAO, systemDao *dao.SystemDAO, pluginDao *dao.PluginDAO) *Manager {
	return newManager(roomAISettingDao, systemDao, pluginDao)
}

// Start 启动所有已启用房间的 AI 对话监听
func (m *Manager) Start() error {
	return m.start()
}

// StopAll 停止所有房间的 AI 对话监听
func (m *Manager) StopAll() {
	m.stopAll()
}

// StopRoom 停止指定房间的 AI 对话监听
func (m *Manager) StopRoom(roomID int) {
	m.lifecycle.Lock()
	defer m.lifecycle.Unlock()
	m.stopRoom(roomID)
}

// Reload 重新加载 AI 对话监听。
// 传入房间 ID 时只重新加载指定房间；不传参数时重新加载所有已启用房间。
func (m *Manager) Reload(roomIDs ...int) error {
	return m.reload(roomIDs...)
}

// Close 关闭 AI 对话服务，释放所有资源
func (m *Manager) Close() {
	m.closeManager()
}

// BuildEmbeddingIndex 手动构建向量索引。config 指定 Embedding API 配置，force 为 true 时强制全量重建。
// 构建过程可能耗时较长。
func (m *Manager) BuildEmbeddingIndex(config EmbeddingConfig, force bool) error {
	return m.buildEmbeddingIndex(config, force)
}

// GetEmbeddingStats 获取向量索引统计信息
func (m *Manager) GetEmbeddingStats(config EmbeddingConfig) EmbeddingStats {
	return m.getEmbeddingStats(config)
}

// BuildKeywordIndex 手动构建关键词搜索索引。force 为 true 时强制重建。
func (m *Manager) BuildKeywordIndex(force bool) error {
	return m.buildKeywordIndex(force)
}

// UnloadKeywordIndex 释放关键词搜索索引占用的内存。下次 Search 时会自动从磁盘重新加载。
func (m *Manager) UnloadKeywordIndex() {
	m.unloadKeywordIndex()
}

// ValidateModelConfig 校验 AI 模型配置参数
func ValidateModelConfig(config *models.AIModelConfig) error {
	return validateModelConfig(config)
}

// ValidateRoomSetting 校验房间 AI 设置参数
func ValidateRoomSetting(setting *models.RoomAISetting) error {
	return validateRoomSetting(setting)
}

// ValidateBaseSetting 校验管理员维护的 AI 基础配置。
func ValidateBaseSetting(setting *models.AIBaseSetting) error {
	return validateBaseSetting(setting)
}
