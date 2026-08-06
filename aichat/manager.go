package aichat

import (
	"context"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"

	"gorm.io/gorm"
)

const (
	chatLogBufferSize = 256
	maxQuestionRunes  = 2000
	maxReplyRunes     = 180
)

type chatSession struct {
	messages   []chatMessage
	lastActive time.Time
}

type roomWorker struct {
	cancel context.CancelFunc
}

func newManager(roomAISettingDao *dao.RoomAISettingDAO, systemDao *dao.SystemDAO, pluginDao *dao.PluginDAO) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	aiManager := &Manager{
		roomAISettingDao: roomAISettingDao,
		systemDao:        systemDao,
		client:           newClient(),
		ctx:              ctx,
		cancel:           cancel,
		workers:          make(map[int]*roomWorker),
		keywordSearcher:  newKeywordWikiSearcher(wikiPagesDir, wikiIndexFile),
	}

	chatPlugin, err := pluginDao.GetPluginByPluginName(models.PluginChat)
	if err != nil {
		logger.Logger.Errorf("获取 %s 插件状态失败, err: %v", models.PluginChat, err)
	} else if chatPlugin.Status {
		if err = aiManager.start(); err != nil {
			logger.Logger.Errorf("启动游戏内 AI 对话服务失败, err: %v", err)
		}
	}

	return aiManager
}

func (m *Manager) start() error {
	m.lifecycle.Lock()
	if m.closed {
		m.lifecycle.Unlock()
		return context.Canceled
	}
	m.active = true
	m.lifecycle.Unlock()

	settings, err := m.roomAISettingDao.ListEnabled()
	if err != nil {
		m.stopAll()
		return err
	}
	for _, setting := range settings {
		if err = m.reload(setting.RoomID); err != nil {
			logger.Logger.Errorf("启动房间 AI 对话监听失败, roomID: %d, err: %v", setting.RoomID, err)
		}
	}
	return nil
}

func (m *Manager) reload(roomID int) error {
	m.lifecycle.Lock()
	defer m.lifecycle.Unlock()
	m.stopRoom(roomID)
	if !m.active || m.closed {
		return nil
	}

	setting, err := m.roomAISettingDao.GetByRoomID(roomID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	if !setting.Enabled {
		return nil
	}
	if err = validateRoomSetting(setting); err != nil {
		return err
	}
	baseSetting, err := m.systemDao.GetAIBaseSetting()
	if err != nil {
		return fmt.Errorf("获取 AI 基础配置失败: %w", err)
	}
	if err = validateBaseSetting(baseSetting); err != nil {
		return fmt.Errorf("AI 基础配置无效: %w", err)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		return err
	}
	if !room.Status {
		return nil
	}
	game := dst.NewGameController(room, worlds, roomSetting, "zh")
	workerCtx, cancel := context.WithCancel(m.ctx)

	m.mu.Lock()
	m.workers[roomID] = &roomWorker{cancel: cancel}
	m.mu.Unlock()

	runtimeSetting := models.AIChatSetting{
		RoomAISetting: *setting,
		AIBaseSetting: *baseSetting,
	}
	go m.runRoom(workerCtx, game, runtimeSetting)
	return nil
}

func (m *Manager) reloadAll() error {
	settings, err := m.roomAISettingDao.ListEnabled()
	if err != nil {
		return err
	}
	var reloadErrors []error
	for _, setting := range settings {
		if err = m.reload(setting.RoomID); err != nil {
			reloadErrors = append(reloadErrors, fmt.Errorf("roomID %d: %w", setting.RoomID, err))
		}
	}
	return errors.Join(reloadErrors...)
}

func (m *Manager) stopAll() {
	m.lifecycle.Lock()
	defer m.lifecycle.Unlock()
	m.active = false
	m.stopAllWorkers()
}

func (m *Manager) stopRoom(roomID int) {
	m.mu.Lock()
	worker, ok := m.workers[roomID]
	if ok {
		delete(m.workers, roomID)
	}
	m.mu.Unlock()

	if ok {
		worker.cancel()
	}
}

func (m *Manager) closeManager() {
	m.lifecycle.Lock()
	defer m.lifecycle.Unlock()
	m.active = false
	m.closed = true
	m.cancel()
	m.stopAllWorkers()
}

func (m *Manager) stopAllWorkers() {
	m.mu.Lock()
	workers := m.workers
	m.workers = make(map[int]*roomWorker)
	m.mu.Unlock()
	for _, worker := range workers {
		worker.cancel()
	}
}

func (m *Manager) runRoom(ctx context.Context, game *dst.Game, setting models.AIChatSetting) {
	lines := make(chan string, chatLogBufferSize)
	go m.watchChatLog(ctx, game, setting.RoomID, lines)

	sessions := make(map[string]*chatSession)
	cleanupTicker := time.NewTicker(time.Minute)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-cleanupTicker.C:
			cleanupSessions(sessions, time.Duration(setting.ContextTTLMinutes)*time.Minute)
		case line := <-lines:
			event, ok := parseChatEvent(line)
			if !ok {
				continue
			}
			question, ok := matchQuestion(setting.Prefix, event.Message)
			if !ok {
				continue
			}
			m.answer(ctx, game, setting, sessions, event, question)
		}
	}
}

func (m *Manager) watchChatLog(ctx context.Context, game *dst.Game, roomID int, lines chan<- string) {
	for {
		err := game.TailChatLog(ctx, 0, lines)
		if ctx.Err() != nil {
			return
		}
		logger.Logger.Errorf("房间聊天日志监听异常, roomID: %d, err: %v", roomID, err)

		timer := time.NewTimer(2 * time.Second)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

// isEmbeddingConfigured 检查是否配置了嵌入模型
func (m *Manager) isEmbeddingConfigured(setting models.AIChatSetting) bool {
	return setting.EmbeddingModel != "" && setting.ChatBaseURL != ""
}

// getEmbeddingSearcher 获取或创建向量搜索引擎
// 仅创建 searcher 实例并缓存，不自动构建索引（构建需通过 BuildEmbeddingIndex 手动触发）
func (m *Manager) getEmbeddingSearcher(setting models.AIChatSetting) *embeddingWikiSearcher {
	// EmbeddingApiKey 如果单独配置了则优先使用，否则复用 ChatApiKey
	apiKey := setting.ChatApiKey
	if setting.EmbeddingApiKey != "" {
		apiKey = setting.EmbeddingApiKey
	}
	configKey := setting.ChatBaseURL + "|" + apiKey + "|" + setting.EmbeddingModel

	m.embedSearcherMu.Lock()
	defer m.embedSearcherMu.Unlock()

	// 配置未变，返回缓存的 searcher
	if m.embedSearcher != nil && m.lastEmbedConfig == configKey {
		return m.embedSearcher
	}

	// 配置变更或首次调用 — 创建新的 searcher（不自动构建索引）
	if m.embedSearcher != nil {
		m.embedSearcher.stopIdleTimer()
	}

	embedConfig := EmbeddingConfig{
		APIURL:     setting.ChatBaseURL,
		APIKey:     apiKey,
		Model:      setting.EmbeddingModel,
		Dimensions: 1024,
	}
	searcher := newEmbeddingWikiSearcher(wikiPagesDir, embedConfig)

	m.embedSearcher = searcher
	m.lastEmbedConfig = configKey
	return m.embedSearcher
}

// BuildEmbeddingIndex 手动构建向量索引。
// config 指定 Embedding API 配置，force 为 true 时强制全量重建。
// 构建过程可能耗时较长，建议在后台执行。
func (m *Manager) buildEmbeddingIndex(config EmbeddingConfig, force bool) error {
	searcher := newEmbeddingWikiSearcher(wikiPagesDir, config)

	if !force && !searcher.needsSetup() {
		logger.Logger.Infof("向量索引已存在，无需重建。使用 force=true 强制重建。")
		return nil
	}

	logger.Logger.Infof("开始手动构建向量索引 (force=%v)...", force)
	if err := searcher.buildIndex(force); err != nil {
		return fmt.Errorf("构建向量索引失败: %w", err)
	}

	// 更新缓存的 searcher
	m.embedSearcherMu.Lock()
	if m.embedSearcher != nil {
		m.embedSearcher.stopIdleTimer()
	}
	apiKey := config.APIKey
	configKey := config.APIURL + "|" + apiKey + "|" + config.Model
	m.embedSearcher = searcher
	m.lastEmbedConfig = configKey
	m.embedSearcherMu.Unlock()

	logger.Logger.Infof("向量索引构建完成")
	return nil
}

// GetEmbeddingStats 获取向量索引统计信息
func (m *Manager) getEmbeddingStats(config EmbeddingConfig) EmbeddingStats {
	searcher := newEmbeddingWikiSearcher(wikiPagesDir, config)
	defer searcher.stopIdleTimer()
	return searcher.getStats()
}

// BuildKeywordIndex 手动构建关键词搜索索引。
// force 为 true 时强制重建，否则索引已存在则跳过。
func (m *Manager) buildKeywordIndex(force bool) error {
	// 停止旧 searcher 的空闲计时器
	if m.keywordSearcher != nil {
		m.keywordSearcher.stopIdleTimer()
	}

	searcher := newKeywordWikiSearcher(wikiPagesDir, wikiIndexFile)

	if !force {
		if _, err := os.Stat(wikiIndexFile); err == nil {
			if loadErr := searcher.load(); loadErr == nil {
				logger.Logger.Infof("关键词索引已存在，无需重建。使用 force=true 强制重建。")
				m.keywordSearcher = searcher
				return nil
			}
		}
	}

	logger.Logger.Infof("开始构建关键词搜索索引 (force=%v)...", force)
	if err := searcher.buildIndex(force); err != nil {
		return fmt.Errorf("构建关键词索引失败: %w", err)
	}

	m.keywordSearcher = searcher
	logger.Logger.Infof("关键词搜索索引构建完成")
	return nil
}

// UnloadKeywordIndex 释放关键词搜索索引占用的内存。
// 下次 Search 时会自动从磁盘重新加载。
func (m *Manager) unloadKeywordIndex() {
	if m.keywordSearcher != nil {
		m.keywordSearcher.unload()
	}
}

// searchWiki 搜索 Wiki 知识库，返回格式化的参考上下文
func (m *Manager) searchWiki(setting models.AIChatSetting, question string) string {
	// 优先使用向量搜索
	if m.isEmbeddingConfigured(setting) {
		if searcher := m.getEmbeddingSearcher(setting); searcher != nil {
			results, err := searcher.search(question, nil, 3, 0.3)
			if err != nil {
				logger.Logger.Warnf("向量搜索 Wiki 失败: %v，回退到关键词搜索", err)
			} else if len(results) > 0 {
				return formatWikiContext(results, maxContextTokens)
			}
		}
	}

	// 回退到关键词搜索
	if m.keywordSearcher != nil {
		results, err := m.keywordSearcher.search(question, 3)
		if err != nil {
			logger.Logger.Warnf("关键词搜索 Wiki 失败: %v", err)
			return ""
		}
		if len(results) > 0 {
			return formatWikiContext(results, maxContextTokens)
		}
	}

	return ""
}

// buildSystemPrompt 构建系统提示词（Wiki 上下文 + 用户设定的提示词）
func buildSystemPrompt(setting models.AIChatSetting, wikiContext string) string {
	// 默认提示词
	defaultPrompt := "你是饥荒联机版游戏内的 AI 助手。请根据以上参考文档，用中文回答玩家的问题。回答应简洁、准确，适合在游戏聊天框中显示，坚决不能使用使用 Markdown 格式，回答不能超过30个字。"
	systemPrompt := strings.TrimSpace(setting.SystemPrompt)
	if systemPrompt == "" {
		systemPrompt = defaultPrompt
	}

	// Wiki 参考文档
	if wikiContext != "" {
		return strings.Join([]string{wikiContext, "", systemPrompt}, "\n")
	}
	return systemPrompt
}

func (m *Manager) answer(ctx context.Context, game *dst.Game, setting models.AIChatSetting, sessions map[string]*chatSession, event chatEvent, question string) {
	now := time.Now()
	ttl := time.Duration(setting.ContextTTLMinutes) * time.Minute
	session := sessions[event.UID]
	if session == nil || now.Sub(session.lastActive) >= ttl {
		session = &chatSession{}
		sessions[event.UID] = session
	}

	// 搜索 Wiki 知识库获取参考上下文
	wikiContext := m.searchWiki(setting, question)

	userMessage := chatMessage{Role: "user", Content: truncateRunes(question, maxQuestionRunes)}
	messages := make([]chatMessage, 0, len(session.messages)+3)

	// 构建系统提示词（Wiki 上下文 + 用户设定的提示词）
	systemContent := buildSystemPrompt(setting, wikiContext)
	if systemContent != "" {
		messages = append(messages, chatMessage{Role: "system", Content: systemContent})
	}
	messages = append(messages, session.messages...)
	messages = append(messages, userMessage)

	answer, err := m.client.complete(ctx, setting.AIModelConfig, messages)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		logger.Logger.Errorf("游戏内 AI 回答失败, roomID: %d, uid: %s, err: %v", setting.RoomID, event.UID, err)
		_ = sendGameReply(game, event.Nickname, "暂时无法回答，请稍后再试。")
		return
	}

	session.messages = append(session.messages, userMessage, chatMessage{Role: "assistant", Content: answer})
	maxMessages := setting.ContextMaxMessages
	if maxMessages%2 != 0 {
		maxMessages--
	}
	if len(session.messages) > maxMessages {
		session.messages = append([]chatMessage(nil), session.messages[len(session.messages)-maxMessages:]...)
	}
	session.lastActive = time.Now()

	if err = sendGameReply(game, event.Nickname, answer); err != nil {
		logger.Logger.Errorf("发送游戏内 AI 回答失败, roomID: %d, uid: %s, err: %v", setting.RoomID, event.UID, err)
	}
}

func cleanupSessions(sessions map[string]*chatSession, ttl time.Duration) {
	now := time.Now()
	for uid, session := range sessions {
		if now.Sub(session.lastActive) >= ttl {
			delete(sessions, uid)
		}
	}
}

func matchQuestion(prefix, message string) (string, bool) {
	message = strings.TrimSpace(message)
	if prefix != "" {
		if !strings.HasPrefix(message, prefix) {
			return "", false
		}
		message = strings.TrimSpace(strings.TrimPrefix(message, prefix))
	}
	return message, message != ""
}

func sendGameReply(game *dst.Game, nickname, answer string) error {
	answer = strings.Join(strings.Fields(answer), " ")
	if answer == "" {
		return fmt.Errorf("AI 回答为空")
	}

	prefix := fmt.Sprintf("[AI] %s: ", nickname)
	for _, part := range splitRunes(answer, maxReplyRunes-utf8.RuneCountInString(prefix)) {
		if err := game.SystemMsg(prefix + part); err != nil {
			return err
		}
	}
	return nil
}

func splitRunes(value string, size int) []string {
	if size <= 0 {
		size = maxReplyRunes
	}
	runes := []rune(value)
	parts := make([]string, 0, (len(runes)+size-1)/size)
	for len(runes) > 0 {
		n := size
		if len(runes) < n {
			n = len(runes)
		}
		parts = append(parts, string(runes[:n]))
		runes = runes[n:]
	}
	return parts
}

func truncateRunes(value string, size int) string {
	runes := []rune(value)
	if len(runes) <= size {
		return value
	}
	return string(runes[:size])
}
