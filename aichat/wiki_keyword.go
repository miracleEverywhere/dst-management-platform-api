package aichat

import (
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

// ========== 配置 ==========

const (
	wikiPagesDir            = utils.PluginAiChatWikiPath + "/pages"
	wikiIndexFile           = utils.PluginAiChatSearchDataPath + "/search_index.json"
	keywordIndexIdleTimeout = 5 * time.Minute
)

// ========== 数据结构 ==========

// wikiResult 单条搜索结果
type wikiResult struct {
	Title      string   `json:"title"`
	Filename   string   `json:"filename"`
	Categories []string `json:"categories"`
	Content    string   `json:"content"`
	ContentLen int      `json:"content_length"`
	Score      float64  `json:"score"`
}

type wikiPageInfo struct {
	Title         string   `json:"title"`
	Categories    []string `json:"categories"`
	Content       string   `json:"content"`
	ContentLength int      `json:"content_length"`
	Links         []string `json:"links"`
	Filename      string   `json:"filename"`
	Terms         []string `json:"terms"`
}

type wikiIndex struct {
	Pages         map[string]wikiPageInfo       `json:"pages"`
	CategoryIndex map[string][]string           `json:"category_index"`
	TermIndex     map[string]map[string]float64 `json:"term_index"`
}

// ========== 关键词搜索引擎 ==========

// KeywordWikiSearcher 基于关键词的 Wiki 搜索引擎
type keywordWikiSearcher struct {
	pagesDir  string
	indexPath string

	mu        sync.RWMutex
	index     *wikiIndex
	idleTimer *time.Timer // 空闲自动释放计时器
}

// NewKeywordWikiSearcher 创建关键词搜索引擎
func newKeywordWikiSearcher(pagesDir, indexPath string) *keywordWikiSearcher {
	return &keywordWikiSearcher{
		pagesDir:  pagesDir,
		indexPath: indexPath,
	}
}

// Load 从磁盘加载搜索索引
func (s *keywordWikiSearcher) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

// Unload 释放搜索索引占用的内存。下次 Search 时会自动重新从磁盘加载。
func (s *keywordWikiSearcher) unload() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.index != nil {
		logger.Logger.Infof("释放关键词搜索索引内存 (%d 篇文档)", len(s.index.Pages))
		s.index = nil
	}
	s.stopIdleTimer()
}

// IsLoaded 检查索引是否已加载到内存
func (s *keywordWikiSearcher) isLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.index != nil
}

// resetIdleTimer 重置空闲计时器。调用方需持有 s.mu 写锁。
// 每次 Search 调用时重置，超时后自动 Unload。
func (s *keywordWikiSearcher) resetIdleTimer() {
	if s.idleTimer != nil {
		s.idleTimer.Stop()
	}
	s.idleTimer = time.AfterFunc(keywordIndexIdleTimeout, func() {
		//logger.Logger.Infof("关键词搜索索引空闲超过 %v，自动释放内存", keywordIndexIdleTimeout)
		s.mu.Lock()
		s.unload()
		s.mu.Unlock()
	})
}

// stopIdleTimer 停止空闲计时器
func (s *keywordWikiSearcher) stopIdleTimer() {
	if s.idleTimer != nil {
		s.idleTimer.Stop()
		s.idleTimer = nil
	}
}

// loadLocked 不加锁加载（调用方需持有 s.mu 写锁）
func (s *keywordWikiSearcher) loadLocked() error {
	if s.index != nil {
		return nil
	}

	data, err := os.ReadFile(s.indexPath)
	if err != nil {
		return fmt.Errorf("搜索索引文件不存在: %w", err)
	}

	var idx wikiIndex
	if err := json.Unmarshal(data, &idx); err != nil {
		return fmt.Errorf("解析搜索索引失败: %w", err)
	}

	s.index = &idx
	s.resetIdleTimer()
	logger.Logger.Infof("已加载关键词搜索索引: %d 篇文档, %d 个词条", len(idx.Pages), len(idx.TermIndex))
	return nil
}

// Search 搜索 Wiki 文档。索引未构建时返回 error。
func (s *keywordWikiSearcher) search(query string, maxResults int) ([]wikiResult, error) {
	s.mu.Lock()
	if s.index == nil {
		if err := s.loadLocked(); err != nil {
			s.mu.Unlock()
			return nil, fmt.Errorf("搜索索引未构建，请先构建索引: %w", err)
		}
	}
	idx := s.index
	s.resetIdleTimer() // 每次 Search 重置空闲计时器
	s.mu.Unlock()

	queryTerms := tokenizeQuery(query)
	scores := make(map[string]float64)
	matchedTerms := make(map[string][]string)

	queryLower := strings.ToLower(strings.TrimSpace(query))

	// TF 评分
	for term := range queryTerms {
		if docs, ok := idx.TermIndex[term]; ok {
			for filename, tfScore := range docs {
				scores[filename] += tfScore
				matchedTerms[filename] = append(matchedTerms[filename], term)
			}
		}
	}

	// 标题匹配加权
	for filename, info := range idx.Pages {
		titleLower := strings.ToLower(info.Title)
		if strings.Contains(titleLower, queryLower) {
			scores[filename] += 5.0
			matchedTerms[filename] = append(matchedTerms[filename], "标题精确匹配")
		}
		for term := range queryTerms {
			if utf8.RuneCountInString(term) >= 3 && strings.Contains(titleLower, term) {
				scores[filename] += 1.0
			}
		}
	}

	// 排序
	type scoredFile struct {
		filename string
		score    float64
	}
	ranked := make([]scoredFile, 0, len(scores))
	for filename, score := range scores {
		ranked = append(ranked, scoredFile{filename, score})
	}
	sort.Slice(ranked, func(i, j int) bool {
		return ranked[i].score > ranked[j].score
	})

	// 取 Top N
	results := make([]wikiResult, 0, maxResults)
	for _, sf := range ranked {
		if sf.score <= 0 {
			continue
		}
		info := idx.Pages[sf.filename]
		if info.ContentLength < 50 {
			continue
		}
		results = append(results, wikiResult{
			Title:      info.Title,
			Filename:   info.Filename,
			Categories: info.Categories,
			Content:    info.Content,
			ContentLen: info.ContentLength,
			Score:      roundFloat(sf.score, 2),
		})
		if len(results) >= maxResults {
			break
		}
	}

	return results, nil
}

// ========== 分词 ==========

func tokenizeQuery(query string) map[string]struct{} {
	terms := make(map[string]struct{})

	// 中文连续 2-4 字
	for _, chunk := range findChineseChunks(query) {
		runes := []rune(chunk)
		for i := 0; i < len(runes); i++ {
			for length := 2; length <= 4; length++ {
				if i+length <= len(runes) {
					terms[string(runes[i:i+length])] = struct{}{}
				}
			}
		}
	}

	// 英文词
	for _, word := range regexp.MustCompile(`[a-zA-Z0-9]{2,}`).FindAllString(strings.ToLower(query), -1) {
		terms[word] = struct{}{}
	}

	// 完整查询词
	terms[strings.ToLower(strings.TrimSpace(query))] = struct{}{}

	return terms
}

func findChineseChunks(text string) []string {
	var chunks []string
	var current []rune
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			current = append(current, r)
		} else {
			if len(current) > 0 {
				chunks = append(chunks, string(current))
				current = nil
			}
		}
	}
	if len(current) > 0 {
		chunks = append(chunks, string(current))
	}
	return chunks
}

// ========== 上下文格式化 ==========

// maxContextTokens 默认最大上下文 token 数
const maxContextTokens = 8000

// formatWikiContext 将搜索结果格式化为 AI 聊天的参考上下文
func formatWikiContext(results []wikiResult, maxTokens int) string {
	if len(results) == 0 {
		return ""
	}

	if maxTokens <= 0 {
		maxTokens = maxContextTokens
	}

	var parts []string
	parts = append(parts, "# 饥荒 Wiki 参考文档")

	totalChars := 0
	maxChars := maxTokens * 2 // 粗略按字符数/2 估算 token

	for i, r := range results {
		header := formatResultHeader(i+1, r)
		body := r.Content

		entryChars := utf8.RuneCountInString(header) + utf8.RuneCountInString(body) + 10
		if totalChars+entryChars > maxChars {
			remaining := maxChars - totalChars - utf8.RuneCountInString(header) - 50
			if remaining > 200 {
				runes := []rune(body)
				if len(runes) > remaining {
					body = string(runes[:remaining]) + "\n\n（内容已截断...）"
				}
			} else {
				break
			}
		}

		parts = append(parts, "")
		parts = append(parts, header)
		parts = append(parts, "")
		parts = append(parts, body)

		totalChars += entryChars
	}

	return strings.Join(parts, "\n")
}

func formatResultHeader(index int, r wikiResult) string {
	h := "## " + itoa(index) + ". " + r.Title
	if len(r.Categories) > 0 {
		h += "  [" + strings.Join(r.Categories, ", ") + "]"
	}
	return h
}

func itoa(n int) string {
	if n <= 0 {
		return "0"
	}
	digits := make([]byte, 0, 10)
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func roundFloat(f float64, decimals int) float64 {
	pow := 1.0
	for i := 0; i < decimals; i++ {
		pow *= 10
	}
	return float64(int(f*pow+0.5)) / pow
}

// ================================================================
// 构建关键词搜索索引
// ================================================================

// BuildIndex 构建关键词搜索索引。扫描所有 .md 文件，构建 TF 评分索引。
// force 为 true 时强制重建，否则索引已存在则跳过。
func (s *keywordWikiSearcher) buildIndex(force bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 非强制模式下，如果索引文件已存在，直接加载
	if !force {
		if _, err := os.Stat(s.indexPath); err == nil {
			logger.Logger.Infof("关键词索引已存在，加载中...")
			return s.loadLocked()
		}
	}

	logger.Logger.Infof("正在构建关键词搜索索引...")
	start := time.Now()

	idx := &wikiIndex{
		Pages:         make(map[string]wikiPageInfo),
		CategoryIndex: make(map[string][]string),
		TermIndex:     make(map[string]map[string]float64),
	}

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

	for i, fp := range mdFiles {
		if i%200 == 0 {
			logger.Logger.Infof("处理中... %d/%d", i, total)
		}

		filename := filepath.Base(fp)
		content, readErr := os.ReadFile(fp)
		if readErr != nil {
			logger.Logger.Warnf("读取 %s 失败: %v", filename, readErr)
			continue
		}
		rawContent := string(content)

		// 提取标题
		title := extractWikiTitle(rawContent, filename)

		// 提取分类
		categories := extractWikiCategories(rawContent)

		// 提取正文
		body := stripWikiMeta(rawContent)

		// 提取内部链接
		links := extractWikiLinks(body)

		// 分词
		textForSearch := title + " " + body
		textForSearch = cleanForTokenize(textForSearch)
		allTerms := tokenizeText(textForSearch)

		// 记录页面信息
		idx.Pages[filename] = wikiPageInfo{
			Title:         title,
			Categories:    categories,
			Content:       body,
			ContentLength: utf8.RuneCountInString(body),
			Links:         links,
			Filename:      filename,
			Terms:         sortedTermSlice(allTerms),
		}

		// 分类索引
		for _, cat := range categories {
			idx.CategoryIndex[cat] = append(idx.CategoryIndex[cat], filename)
		}

		// 词索引 (TF scoring: 词在文档中出现越多，分越高)
		textLower := strings.ToLower(textForSearch)
		termCounts := make(map[string]int)
		for term := range allTerms {
			termCounts[term] = strings.Count(textLower, strings.ToLower(term))
		}

		maxCount := 1
		for _, count := range termCounts {
			if count > maxCount {
				maxCount = count
			}
		}

		for term, count := range termCounts {
			// 归一化 TF + 标题中出现加权
			score := float64(count) / float64(maxCount)
			if strings.Contains(strings.ToLower(title), strings.ToLower(term)) {
				score *= 2.0
			}
			if idx.TermIndex[term] == nil {
				idx.TermIndex[term] = make(map[string]float64)
			}
			idx.TermIndex[term][filename] = score
		}
	}

	// 保存索引
	if err := saveKeywordIndex(s.indexPath, idx); err != nil {
		return err
	}

	s.index = idx
	s.resetIdleTimer()

	elapsed := time.Since(start)
	logger.Logger.Infof("索引构建完成! %d 个页面, %d 个词条, 耗时 %.1fs", total, len(idx.TermIndex), elapsed.Seconds())
	return nil
}

// ========== 索引构建辅助函数 ==========

// tokenizeText 对文本全文分词（用于构建索引，不含完整查询词）
func tokenizeText(text string) map[string]struct{} {
	terms := make(map[string]struct{})

	// 中文连续 2-4 字
	for _, chunk := range findChineseChunks(text) {
		runes := []rune(chunk)
		for j := 0; j < len(runes); j++ {
			for length := 2; length <= 4; length++ {
				if j+length <= len(runes) {
					terms[string(runes[j:j+length])] = struct{}{}
				}
			}
		}
	}

	// 英文词
	for _, word := range regexp.MustCompile(`[a-zA-Z0-9]{2,}`).FindAllString(strings.ToLower(text), -1) {
		terms[word] = struct{}{}
	}

	return terms
}

// cleanForTokenize 去除 markdown 格式字符，压缩空白
func cleanForTokenize(text string) string {
	text = regexp.MustCompile(`[*#>`+"`"+`\[\]()!_~|]`).ReplaceAllString(text, " ")
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	return text
}

// extractWikiLinks 提取 Wiki 内部链接的显示文本
var wikiLinkRe = regexp.MustCompile(`\[([^\]]+)\]\(([^)]+\.md)\)`)

func extractWikiLinks(body string) []string {
	matches := wikiLinkRe.FindAllStringSubmatch(body, -1)
	links := make([]string, 0, len(matches))
	for _, m := range matches {
		if len(m) >= 2 {
			links = append(links, m[1])
		}
	}
	return links
}

// sortedTermSlice 将 term set 转为排序的 slice
func sortedTermSlice(terms map[string]struct{}) []string {
	result := make([]string, 0, len(terms))
	for t := range terms {
		result = append(result, t)
	}
	sort.Strings(result)
	return result
}

// saveKeywordIndex 保存搜索索引到磁盘
func saveKeywordIndex(indexPath string, idx *wikiIndex) error {
	dir := filepath.Dir(indexPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建索引目录失败: %w", err)
	}

	data, err := json.Marshal(idx)
	if err != nil {
		return fmt.Errorf("序列化搜索索引失败: %w", err)
	}

	if err := os.WriteFile(indexPath, data, 0644); err != nil {
		return fmt.Errorf("保存搜索索引失败: %w", err)
	}

	return nil
}
