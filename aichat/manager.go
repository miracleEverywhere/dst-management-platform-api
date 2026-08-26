package aichat

import (
	"context"
	"dst-management-platform-api/cache"
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
		pluginDao:        pluginDao,
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
	var reloadErrors []error
	for _, setting := range settings {
		if err = m.reloadRoom(setting.RoomID); err != nil {
			reloadErr := fmt.Errorf("roomID %d: %w", setting.RoomID, err)
			reloadErrors = append(reloadErrors, reloadErr)
			logger.Logger.Errorf("启动房间 AI 对话监听失败, err: %v", reloadErr)
		}
	}
	return errors.Join(reloadErrors...)
}

func (m *Manager) reload(roomIDs ...int) error {
	started, err := m.ensureActive()
	if err != nil {
		return err
	}
	if len(roomIDs) == 0 {
		if started {
			return nil
		}
		m.lifecycle.Lock()
		active := m.active
		m.lifecycle.Unlock()
		if !active {
			return nil
		}
		m.stopAll()
		return m.start()
	}

	if started {
		return nil
	}
	var reloadErrors []error
	for _, roomID := range roomIDs {
		if err = m.reloadRoom(roomID); err != nil {
			reloadErrors = append(reloadErrors, fmt.Errorf("roomID %d: %w", roomID, err))
		}
	}
	return errors.Join(reloadErrors...)
}

func (m *Manager) reloadRoom(roomID int) error {
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
	// 兼容旧版本保存的 60 字默认值，升级后按新默认值运行。
	if setting.MaxReplyLength < models.MinAIReplyLength {
		setting.MaxReplyLength = models.DefaultAIReplyMaxLength
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

// ensureActive 恢复插件已启用但管理器尚未激活的运行状态。
// 返回值表示本次调用是否启动了管理器。
func (m *Manager) ensureActive() (bool, error) {
	m.lifecycle.Lock()
	if m.closed {
		m.lifecycle.Unlock()
		return false, context.Canceled
	}
	if m.active {
		m.lifecycle.Unlock()
		return false, nil
	}
	m.lifecycle.Unlock()

	plugin, err := m.pluginDao.GetPluginByPluginName(models.PluginChat)
	if err != nil {
		return false, fmt.Errorf("获取 %s 插件状态失败: %w", models.PluginChat, err)
	}
	if !plugin.Status {
		return false, nil
	}
	return true, m.start()
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

// isEmbeddingConfigured 只有 embedding 地址、密钥和模型均已配置时才启用向量检索。
func (m *Manager) isEmbeddingConfigured(setting models.AIChatSetting) bool {
	return setting.EmbeddingBaseURL != "" &&
		setting.EmbeddingApiKey != "" &&
		setting.EmbeddingModel != ""
}

// getEmbeddingSearcher 获取或创建向量搜索引擎
// 仅创建 searcher 实例并缓存，不自动构建索引（构建需通过 BuildEmbeddingIndex 手动触发）
func (m *Manager) getEmbeddingSearcher(setting models.AIChatSetting) *embeddingWikiSearcher {
	configKey := setting.EmbeddingBaseURL + "|" + setting.EmbeddingApiKey + "|" + setting.EmbeddingModel

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
		APIURL:     setting.EmbeddingBaseURL,
		APIKey:     setting.EmbeddingApiKey,
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
	m.embedBuildMu.Lock()
	if m.embedBuildCancel != nil {
		m.embedBuildMu.Unlock()
		return fmt.Errorf("向量索引正在构建中")
	}
	ctx, cancel := context.WithCancel(m.ctx)
	m.embedBuildCancel = cancel
	m.embedBuildMu.Unlock()
	defer func() {
		cancel()
		m.embedBuildMu.Lock()
		m.embedBuildCancel = nil
		m.embedBuildMu.Unlock()
	}()

	searcher := newEmbeddingWikiSearcher(wikiPagesDir, config)

	if !force && !searcher.needsSetup() {
		logger.Logger.Infof("向量索引已存在，无需重建。使用 force=true 强制重建。")
		return nil
	}

	logger.Logger.Infof("开始手动构建向量索引 (force=%v)...", force)
	if err := searcher.buildIndex(ctx, force); err != nil {
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
	maxResults := setting.MaxResults
	if maxResults <= 0 {
		maxResults = models.DefaultAIWikiMaxResults
	}

	// 优先使用向量搜索
	if m.isEmbeddingConfigured(setting) {
		if searcher := m.getEmbeddingSearcher(setting); searcher != nil {
			results, err := searcher.search(question, nil, maxResults, 0.3)
			if err != nil {
				logger.Logger.Warnf("向量搜索 Wiki 失败: %v，回退到关键词搜索", err)
			} else if len(results) > 0 {
				logger.Logger.Debugf("Wiki 搜索方式: embedding, roomID: %d, results: %d", setting.RoomID, len(results))
				return formatWikiContext(results, maxContextTokens)
			}
		}
	} else {
		logger.Logger.Debugf("Wiki 搜索跳过 embedding，使用关键词搜索, roomID: %d", setting.RoomID)
	}

	// 回退到关键词搜索
	if m.keywordSearcher != nil {
		results, err := m.keywordSearcher.search(question, maxResults)
		if err != nil {
			logger.Logger.Warnf("关键词搜索 Wiki 失败: %v", err)
			logger.Logger.Debugf("Wiki 搜索方式: keyword, roomID: %d, results: 0", setting.RoomID)
			return ""
		}
		logger.Logger.Debugf("Wiki 搜索方式: keyword, roomID: %d, results: %d", setting.RoomID, len(results))
		if len(results) > 0 {
			return formatWikiContext(results, maxContextTokens)
		}
	}

	return ""
}

func maxReplyLength(setting models.AIChatSetting) int {
	if setting.MaxReplyLength < models.MinAIReplyLength {
		return models.DefaultAIReplyMaxLength
	}
	if setting.MaxReplyLength > models.MaxAIReplyLength {
		return models.MaxAIReplyLength
	}
	return setting.MaxReplyLength
}

// buildSystemPrompt 构建系统提示词（Wiki 上下文 + 用户设定的提示词）。
// playerPrefab 为空时不添加玩家角色上下文。
func buildSystemPrompt(setting models.AIChatSetting, wikiContext, playerPrefab string) string {
	// 默认提示词
	limitPrompt := fmt.Sprintf("回答不能超过%d个字。", maxReplyLength(setting))
	defaultPrompt := "你是饥荒联机版游戏内的 AI 助手。请根据以上参考文档，用中文回答玩家的问题。回答应简洁、准确，适合在游戏聊天框中显示，坚决不能使用使用 Markdown 格式。"
	if playerPrefab != "" {
		defaultPrompt += fmt.Sprintf("当前提问玩家使用的角色是 %s，请结合该角色信息理解问题。", playerPrefab)
	}
	defaultPrompt += limitPrompt
	systemPrompt := strings.TrimSpace(setting.SystemPrompt)
	if systemPrompt == "" {
		systemPrompt = defaultPrompt
	} else {
		systemPrompt = strings.Join([]string{systemPrompt, limitPrompt}, "\n")
	}

	// Wiki 参考文档
	if wikiContext != "" {
		return strings.Join([]string{wikiContext, "", systemPrompt}, "\n")
	}
	return systemPrompt
}

func currentPlayerPrefab(roomID int, uid string) string {
	if roomID <= 0 || uid == "" {
		return ""
	}

	cache.PlayersStatisticMutex.Lock()
	defer cache.PlayersStatisticMutex.Unlock()

	snapshots := cache.PlayersStatistic[roomID]
	if len(snapshots) == 0 {
		return ""
	}
	players := snapshots[len(snapshots)-1].PlayerInfo
	for _, player := range players {
		if player.UID == uid {
			return strings.TrimSpace(player.Prefab)
		}
	}
	return ""
}

func (m *Manager) answer(ctx context.Context, game *dst.Game, setting models.AIChatSetting, sessions map[string]*chatSession, event chatEvent, question string) {
	wikiContext := m.searchWiki(setting, question)
	if wikiContext == "" {
		if err := sendGameReply(game, event.Nickname, "还在学习中"); err != nil {
			logger.Logger.Errorf("发送游戏内 AI 学习中提示失败, roomID: %d, uid: %s, err: %v", setting.RoomID, event.UID, err)
		}
		return
	}

	now := time.Now()
	ttl := time.Duration(setting.ContextTTLMinutes) * time.Minute
	session := sessions[event.UID]
	if session == nil || now.Sub(session.lastActive) >= ttl {
		session = &chatSession{}
		sessions[event.UID] = session
	}

	// 搜索 Wiki 知识库获取参考上下文
	userMessage := chatMessage{Role: "user", Content: truncateRunes(question, maxQuestionRunes)}
	messages := make([]chatMessage, 0, len(session.messages)+3)

	// 构建系统提示词（Wiki 上下文 + 用户设定的提示词）
	systemContent := buildSystemPrompt(setting, wikiContext, currentPlayerPrefab(setting.RoomID, event.UID))
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
	answer = truncateRunes(answer, maxReplyLength(setting))

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

	prefix := fmt.Sprintf("[DMP] %s: ", nickname)
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
