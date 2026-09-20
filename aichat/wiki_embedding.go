package aichat

import (
	"bytes"
	"context"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

// ========== 配置 ==========

const embeddingDir = utils.PluginAiChatSearchDataPath + "/embeddings"

// ========== 构建索引常量 ==========

const (
	embedBatchSize        = 10
	embedMaxChars         = 4000
	embedRequestInterval  = 300 * time.Millisecond
	embedMinContentLength = 50
	embeddingIdleTimeout  = 5 * time.Minute
)

// embeddingIndexMetaFile 记录向量索引构建来源的元数据文件名。
const embeddingIndexMetaFile = "index_meta.json"

// ========== 预处理正则 ==========

var (
	embedMdTitleRe = regexp.MustCompile(`(?m)^#\s+.+$`)
	embedMdHrRe    = regexp.MustCompile(`(?m)^---$`)
	embedMdLinkRe  = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	embedMdImgRe   = regexp.MustCompile(`!\[.*?\]\(.*?\)`)
	embedMdFmtRe   = regexp.MustCompile(`[*#>` + "`" + `|]`)
	embedWsRe      = regexp.MustCompile(`\s+`)
)

// ========== 元数据结构 ==========

type wikiMeta struct {
	Title         string   `json:"title"`
	Categories    []string `json:"categories"`
	Content       string   `json:"content"`
	ContentLength int      `json:"content_length"`
	Filename      string   `json:"filename"`
}

// embeddingIndexMeta 记录向量索引的构建来源，用于检测模型或维度变更。
// 旧版本没有该文件，此时各字段为零值，除维度外的校验会被跳过。
type embeddingIndexMeta struct {
	Model      string `json:"model"`
	Dimensions int    `json:"dimensions"`
	TotalDocs  int    `json:"totalDocs"`
	UpdatedAt  string `json:"updatedAt"`
}

// ========== 向量搜索引擎 ==========

// EmbeddingWikiSearcher 基于向量相似度的 Wiki 搜索引擎
type embeddingWikiSearcher struct {
	pagesDir     string
	embeddingDir string
	apiURL       string
	apiKey       string
	model        string
	// dimensions 为 0 时不向 API 发送 dimensions 参数，由模型决定原生维度。
	dimensions int

	httpClient *http.Client

	mu         sync.RWMutex
	metadata   map[string]wikiMeta  // filename -> meta
	embeddings map[string][]float64 // filename -> vector
	meta       embeddingIndexMeta   // 已保存索引的构建来源
	loaded     bool
	idleTimer  *time.Timer // 空闲自动释放计时器
}

// newEmbeddingWikiSearcher 创建向量搜索引擎。
// dimensions 为 0 表示不发送 dimensions 参数（BGE-M3 等模型不支持该参数）。
func newEmbeddingWikiSearcher(pagesDir string, config EmbeddingConfig) *embeddingWikiSearcher {
	return &embeddingWikiSearcher{
		pagesDir:     pagesDir,
		embeddingDir: embeddingDir,
		apiURL:       strings.TrimRight(config.APIURL, "/"),
		apiKey:       config.APIKey,
		model:        config.Model,
		dimensions:   config.Dimensions,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

// indexDimensions 返回已加载索引中向量的实际维度，索引为空时返回 0。
// 调用方需持有 s.mu。
func (s *embeddingWikiSearcher) indexDimensions() int {
	for _, vector := range s.embeddings {
		return len(vector)
	}
	return 0
}

// configMismatch 检查已保存的索引是否与当前配置一致，返回不一致的原因（一致时返回空字符串）。
// 维度以索引中向量的实际长度为准；dimensions 为 0 时表示跟随模型原生维度，无法在构建前比较。
// 调用方需持有 s.mu。
func (s *embeddingWikiSearcher) configMismatch() string {
	dim := s.indexDimensions()
	if dim == 0 {
		return ""
	}
	if s.meta.Model != "" && s.meta.Model != s.model {
		return fmt.Sprintf("索引由模型 %s 构建，当前模型为 %s", s.meta.Model, s.model)
	}
	if s.dimensions > 0 && dim != s.dimensions {
		return fmt.Sprintf("索引维度为 %d，当前配置维度为 %d", dim, s.dimensions)
	}
	return ""
}

// NeedsSetup 是否需要构建索引
func (s *embeddingWikiSearcher) needsSetup() bool {
	s.mu.RLock()
	loaded := s.loaded
	s.mu.RUnlock()
	if !loaded {
		s.load()
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	if len(s.embeddings) == 0 {
		return true
	}
	return s.configMismatch() != ""
}

// Search 向量语义搜索
func (s *embeddingWikiSearcher) search(query string, categories []string, maxResults int, minScore float64) ([]wikiResult, error) {
	s.mu.RLock()
	if !s.loaded {
		s.mu.RUnlock()
		s.load()
		s.mu.RLock()
	}
	embeddings := s.embeddings
	metadata := s.metadata
	indexDim := s.indexDimensions()
	mismatch := s.configMismatch()
	s.mu.RUnlock()

	// 每次搜索重置空闲计时器
	s.mu.Lock()
	s.resetIdleTimer()
	s.mu.Unlock()

	if len(embeddings) == 0 {
		return nil, fmt.Errorf("向量索引为空")
	}
	if mismatch != "" {
		return nil, fmt.Errorf("%s，请重建向量索引", mismatch)
	}

	// 对查询文本做 embedding
	queryVector, err := s.embedSingle(query)
	if err != nil {
		return nil, fmt.Errorf("查询文本 embedding 失败: %w", err)
	}

	// 查询向量与索引向量维度必须一致，否则余弦相似度全部为 0。
	if indexDim > 0 && len(queryVector) != indexDim {
		return nil, fmt.Errorf("查询向量维度为 %d，索引维度为 %d，embedding 模型或向量维度配置已变更，请重建向量索引", len(queryVector), indexDim)
	}

	// 计算余弦相似度
	type scoredDoc struct {
		filename string
		score    float64
	}
	var scores []scoredDoc

	for filename, docVec := range embeddings {
		meta, ok := metadata[filename]
		if !ok || meta.ContentLength < embedMinContentLength {
			continue
		}
		if len(categories) > 0 {
			catSet := make(map[string]struct{}, len(categories))
			for _, c := range categories {
				catSet[c] = struct{}{}
			}
			hasCat := false
			for _, c := range meta.Categories {
				if _, ok := catSet[c]; ok {
					hasCat = true
					break
				}
			}
			if !hasCat {
				continue
			}
		}

		sim := cosineSimilarity(queryVector, docVec)
		if sim >= minScore {
			scores = append(scores, scoredDoc{filename, sim})
		}
	}

	// 排序取 Top N
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	results := make([]wikiResult, 0, maxResults)
	for i := 0; i < len(scores) && i < maxResults; i++ {
		meta := metadata[scores[i].filename]
		results = append(results, wikiResult{
			Title:      meta.Title,
			Filename:   meta.Filename,
			Categories: meta.Categories,
			Content:    meta.Content,
			ContentLen: meta.ContentLength,
			Score:      roundFloat(scores[i].score, 4),
		})
	}

	return results, nil
}

// ========== 内部方法 ==========

func (s *embeddingWikiSearcher) load() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.loadLocked()
}

// loadLocked 不加锁地从磁盘加载（调用方需持有 s.mu 写锁）
func (s *embeddingWikiSearcher) loadLocked() {
	if s.loaded {
		return
	}

	metadataPath := filepath.Join(s.embeddingDir, "metadata.json")
	embeddingsPath := filepath.Join(s.embeddingDir, "embeddings.json")

	if data, err := os.ReadFile(metadataPath); err == nil {
		err = json.Unmarshal(data, &s.metadata)
		if err != nil {
			logger.Logger.Errorf("载入向量模型失败: %v", err)
		}
	}
	if s.metadata == nil {
		s.metadata = make(map[string]wikiMeta)
	}

	if data, err := os.ReadFile(embeddingsPath); err == nil {
		err = json.Unmarshal(data, &s.embeddings)
		if err != nil {
			logger.Logger.Errorf("载入向量模型失败: %v", err)
		}
	}
	if s.embeddings == nil {
		s.embeddings = make(map[string][]float64)
	}

	s.meta = embeddingIndexMeta{}
	if meta, metaErr := readEmbeddingIndexMeta(); metaErr == nil {
		s.meta = meta
	} else if !os.IsNotExist(metaErr) {
		logger.Logger.Errorf("载入向量索引元数据失败: %v", metaErr)
	}

	s.loaded = true
	s.resetIdleTimer()
	logger.Logger.Infof("已加载向量索引: %d 篇文档 (模型: %s, 维度: %d)", len(s.embeddings), s.meta.Model, s.indexDimensions())
}

// unload 释放向量索引占用的内存。下次 search 时会自动从磁盘重新加载。
func (s *embeddingWikiSearcher) unload() {
	if s.embeddings != nil {
		logger.Logger.Infof("释放向量索引内存 (%d 篇文档)", len(s.embeddings))
	}
	s.metadata = nil
	s.embeddings = nil
	s.meta = embeddingIndexMeta{}
	s.loaded = false
	s.stopIdleTimer()
}

// resetIdleTimer 重置空闲计时器。调用方需持有 s.mu 写锁。
func (s *embeddingWikiSearcher) resetIdleTimer() {
	if s.idleTimer != nil {
		s.idleTimer.Stop()
	}
	s.idleTimer = time.AfterFunc(embeddingIdleTimeout, func() {
		s.mu.Lock()
		//logger.Logger.Infof("向量索引空闲超过 %v，自动释放内存", embeddingIdleTimeout)
		s.unload()
		s.mu.Unlock()
	})
}

// stopIdleTimer 停止空闲计时器
func (s *embeddingWikiSearcher) stopIdleTimer() {
	if s.idleTimer != nil {
		s.idleTimer.Stop()
		s.idleTimer = nil
	}
}

func (s *embeddingWikiSearcher) embedSingle(text string) ([]float64, error) {
	vectors, err := s.embedBatch(context.Background(), []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("embedding 返回为空")
	}
	return vectors[0], nil
}

// dimensionsLabel 返回 dimensions 参数的日志描述。
func (s *embeddingWikiSearcher) dimensionsLabel() string {
	if s.dimensions > 0 {
		return fmt.Sprintf("%d", s.dimensions)
	}
	return "模型原生（不发送 dimensions 参数）"
}

// embedBatch 调用 embedding API。
// dimensions 为 0 时不发送该参数；BGE-M3 等不支持 dimensions 的模型必须如此。
func (s *embeddingWikiSearcher) embedBatch(ctx context.Context, texts []string) ([][]float64, error) {
	// 构建请求
	reqBody := map[string]interface{}{
		"model":           s.model,
		"input":           texts,
		"encoding_format": "float",
	}
	if s.dimensions > 0 {
		reqBody["dimensions"] = s.dimensions
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化 embedding 请求失败: %w", err)
	}

	endpoint := s.apiURL + "/embeddings"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建 embedding 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if s.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+s.apiKey)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 32*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("读取 embedding 响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("embedding API 返回 HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data []struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		} `json:"data"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("解析 embedding 响应失败: %w", err)
	}
	if result.Error != nil && result.Error.Message != "" {
		return nil, fmt.Errorf("embedding API 错误: %s", result.Error.Message)
	}

	// 按 index 排序
	sort.Slice(result.Data, func(i, j int) bool {
		return result.Data[i].Index < result.Data[j].Index
	})

	vectors := make([][]float64, len(result.Data))
	for i, d := range result.Data {
		vectors[i] = d.Embedding
	}
	return vectors, nil
}

func cosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// ================================================================
// 构建向量索引
// ================================================================

// save 保存索引及其元数据（调用方需持有 s.mu 写锁）。
// 维度以实际写入的向量长度为准，因此 dimensions 配置为 0 时也能记录模型原生维度。
func (s *embeddingWikiSearcher) save() error {
	if err := os.MkdirAll(s.embeddingDir, 0755); err != nil {
		return fmt.Errorf("创建 embedding 目录失败: %w", err)
	}

	metadataPath := filepath.Join(s.embeddingDir, "metadata.json")
	embeddingsPath := filepath.Join(s.embeddingDir, "embeddings.json")
	metaPath := filepath.Join(s.embeddingDir, embeddingIndexMetaFile)

	s.meta = embeddingIndexMeta{
		Model:      s.model,
		Dimensions: s.indexDimensions(),
		TotalDocs:  len(s.embeddings),
		UpdatedAt:  time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(s.metadata)
	if err != nil {
		return fmt.Errorf("序列化 metadata 失败: %w", err)
	}
	if err := os.WriteFile(metadataPath, data, 0644); err != nil {
		return fmt.Errorf("保存 metadata 失败: %w", err)
	}

	data, err = json.Marshal(s.embeddings)
	if err != nil {
		return fmt.Errorf("序列化 embeddings 失败: %w", err)
	}
	if err := os.WriteFile(embeddingsPath, data, 0644); err != nil {
		return fmt.Errorf("保存 embeddings 失败: %w", err)
	}

	data, err = json.Marshal(s.meta)
	if err != nil {
		return fmt.Errorf("序列化索引元数据失败: %w", err)
	}
	if err := os.WriteFile(metaPath, data, 0644); err != nil {
		return fmt.Errorf("保存索引元数据失败: %w", err)
	}

	return nil
}

// BuildIndex 构建向量索引。只跑一次（或 force=true 强制重建）。
//
// 流程:
//   - 扫描所有 .md 页面
//   - 文本预处理
//   - 分批调用 Embedding API
//   - 每批完成后保存（支持断点续传）
func (s *embeddingWikiSearcher) buildIndex(ctx context.Context, force bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 确保已加载已有数据
	s.loadLocked()

	// 扫描所有 .md 页面
	entries, err := os.ReadDir(s.pagesDir)
	if err != nil {
		return fmt.Errorf("扫描 Wiki 页面目录失败: %w", err)
	}

	var mdFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			mdFiles = append(mdFiles, filepath.Join(s.pagesDir, entry.Name()))
		}
	}
	sort.Strings(mdFiles)
	total := len(mdFiles)

	// 增量构建 vs 全量重建
	if !force {
		// 模型或维度变更后继续增量会写入与旧向量不可比的向量，必须全量重建。
		if mismatch := s.configMismatch(); mismatch != "" {
			return fmt.Errorf("已有向量索引与当前配置不一致（%s），请强制重建索引", mismatch)
		}
		var newFiles []string
		for _, f := range mdFiles {
			if _, ok := s.embeddings[filepath.Base(f)]; !ok {
				newFiles = append(newFiles, f)
			}
		}
		if len(newFiles) == 0 {
			logger.Logger.Infof("所有 %d 篇文档已有向量，无需重建。用 force=true 强制重建。", total)
			return nil
		}
		logger.Logger.Infof("增量构建: %d/%d 篇新文档需要 embedding", len(newFiles), total)
		mdFiles = newFiles
	} else {
		s.embeddings = make(map[string][]float64)
		s.metadata = make(map[string]wikiMeta)
		s.meta = embeddingIndexMeta{}
	}

	// 分批处理
	batches, totalBatches := makeBatches(mdFiles, embedBatchSize)

	logger.Logger.Infof("共 %d 篇文档，分 %d 批 (batch_size=%d)", len(mdFiles), totalBatches, embedBatchSize)
	logger.Logger.Infof("API: %s  模型: %s  dimensions: %s", s.apiURL, s.model, s.dimensionsLabel())

	batchIdx := 0

	for batchIdx < totalBatches {
		if err := ctx.Err(); err != nil {
			return err
		}
		batchFiles := batches[batchIdx]

		// 准备这批的文本
		texts, metas, prepErr := s.prepareBatch(batchFiles)
		if prepErr != nil {
			return fmt.Errorf("准备第 %d/%d 批文档失败: %w", batchIdx+1, totalBatches, prepErr)
		}

		// 调用 API。任何一批失败都直接终止构建，不做重试或批次降级，
		// 避免留下与配置不符的部分索引。
		vectors, apiErr := s.embedBatch(ctx, texts)
		if apiErr != nil {
			return fmt.Errorf("第 %d/%d 批 embedding 失败: %w", batchIdx+1, totalBatches, apiErr)
		}
		if len(vectors) != len(batchFiles) {
			return fmt.Errorf("第 %d/%d 批返回 %d 个向量，期望 %d 个", batchIdx+1, totalBatches, len(vectors), len(batchFiles))
		}

		for i, fp := range batchFiles {
			s.embeddings[filepath.Base(fp)] = vectors[i]
		}
		for k, v := range metas {
			s.metadata[k] = v
		}

		if saveErr := s.save(); saveErr != nil {
			logger.Logger.Errorf("保存 embedding 失败: %v", saveErr)
		}
		logger.Logger.Infof("第 %d/%d 批完成 (%d 篇)", batchIdx+1, totalBatches, len(s.embeddings))

		if batchIdx < totalBatches-1 {
			timer := time.NewTimer(embedRequestInterval)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
		batchIdx++
	}

	// 完成
	s.resetIdleTimer()
	logger.Logger.Infof("索引构建完成! 共 %d 篇文档，维度 %d。", len(s.embeddings), s.indexDimensions())

	return nil
}

// ========== 批次准备 ==========

// prepareBatch 批量读取并预处理文档
func (s *embeddingWikiSearcher) prepareBatch(files []string) (texts []string, metas map[string]wikiMeta, err error) {
	texts = make([]string, 0, len(files))
	metas = make(map[string]wikiMeta, len(files))

	for _, fp := range files {
		content, readErr := os.ReadFile(fp)
		if readErr != nil {
			return nil, nil, fmt.Errorf("读取 %s 失败: %w", filepath.Base(fp), readErr)
		}
		rawContent := string(content)

		title := extractWikiTitle(rawContent, filepath.Base(fp))
		categories := extractWikiCategories(rawContent)
		cleanText := preprocessWikiDoc(rawContent, title)
		strippedContent := stripWikiMeta(rawContent)

		texts = append(texts, cleanText)
		metas[filepath.Base(fp)] = wikiMeta{
			Title:         title,
			Categories:    categories,
			Content:       strippedContent,
			ContentLength: utf8.RuneCountInString(cleanText),
			Filename:      filepath.Base(fp),
		}
	}

	return texts, metas, nil
}

// ========== 预处理 ==========

// extractWikiTitle 从文件名提取条目名称
func extractWikiTitle(_ string, fallback string) string {
	return wikiPageName(fallback)
}

// extractWikiCategories 从固定的“分类”章节提取分类
func extractWikiCategories(rawContent string) []string {
	return parseWikiDocument(rawContent, "").Categories
}

// preprocessWikiDoc 预处理文档文本，使其适合 embedding
//
// 处理步骤:
//   - 去 markdown 语法
//   - 标题前置（标题是最重要的语义信号）
//   - 截断到 embedMaxChars
func preprocessWikiDoc(rawContent, title string) string {
	text := rawContent

	// 去章节标题和水平线
	text = embedMdTitleRe.ReplaceAllString(text, "")
	text = embedMdHrRe.ReplaceAllString(text, "")

	// 去 markdown 格式字符
	text = embedMdLinkRe.ReplaceAllString(text, "$1") // [text](url) → text
	text = embedMdImgRe.ReplaceAllString(text, " ")
	text = embedMdFmtRe.ReplaceAllString(text, " ")
	text = embedWsRe.ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	// 标题前置
	text = fmt.Sprintf("标题: %s\n正文: %s", title, text)

	// 截断
	runes := []rune(text)
	if len(runes) > embedMaxChars {
		text = string(runes[:embedMaxChars])
	}

	return text
}

// stripWikiMeta 去除元数据行，保留纯正文
func stripWikiMeta(rawContent string) string {
	text := rawContent
	text = embedMdTitleRe.ReplaceAllString(text, "")
	text = embedMdHrRe.ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}

// ========== 辅助函数 ==========

func makeBatches(files []string, size int) (batches [][]string, total int) {
	for i := 0; i < len(files); i += size {
		end := i + size
		if end > len(files) {
			end = len(files)
		}
		batches = append(batches, files[i:end])
	}
	return batches, len(batches)
}

// readEmbeddingIndexMeta 读取索引元数据文件。
func readEmbeddingIndexMeta() (embeddingIndexMeta, error) {
	var meta embeddingIndexMeta
	data, err := os.ReadFile(filepath.Join(embeddingDir, embeddingIndexMetaFile))
	if err != nil {
		return meta, err
	}
	if err = json.Unmarshal(data, &meta); err != nil {
		return meta, err
	}
	return meta, nil
}

// firstEmbeddingDimensions 读取 embeddings.json 中第一个向量的长度。
// 使用流式解析，避免为了取维度而加载整个索引。没有索引时返回 0。
func firstEmbeddingDimensions() int {
	file, err := os.Open(filepath.Join(embeddingDir, "embeddings.json"))
	if err != nil {
		return 0
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	if _, err = decoder.Token(); err != nil { // 左花括号
		return 0
	}
	for decoder.More() {
		if _, err = decoder.Token(); err != nil { // 文件名
			return 0
		}
		var vector []float64
		if err = decoder.Decode(&vector); err != nil {
			return 0
		}
		if len(vector) > 0 {
			return len(vector)
		}
	}
	return 0
}

// embeddingIndexDimensions 返回已构建向量索引的维度，没有索引时返回 -1。
// 优先使用索引元数据；旧版本索引没有元数据文件时回退到读取第一个向量的长度。
func embeddingIndexDimensions() int {
	if meta, err := readEmbeddingIndexMeta(); err == nil && meta.Dimensions > 0 {
		return meta.Dimensions
	}
	if dim := firstEmbeddingDimensions(); dim > 0 {
		return dim
	}
	return -1
}

// ================================================================
// 索引统计
// ================================================================

// GetStats 获取索引统计信息
func (s *embeddingWikiSearcher) getStats() EmbeddingStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.loaded {
		s.mu.RUnlock()
		s.load()
		s.mu.RLock()
	}

	stats := EmbeddingStats{
		TotalDocs:  len(s.embeddings),
		Categories: make(map[string]int),
	}

	for _, meta := range s.metadata {
		for _, c := range meta.Categories {
			stats.Categories[c]++
		}
	}
	stats.TotalCategories = len(stats.Categories)

	if len(s.embeddings) > 0 {
		for _, v := range s.embeddings {
			stats.VectorDim = len(v)
			break
		}
	}

	return stats
}
