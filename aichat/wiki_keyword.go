package aichat

import (
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"math"
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

const (
	wikiPagesDir              = utils.PluginAiChatWikiPath
	wikiIndexFile             = utils.PluginAiChatSearchDataPath + "/search_index.json"
	keywordIndexIdleTimeout   = 5 * time.Minute
	keywordIndexVersion       = 2
	keywordNameWeight         = 8.0
	keywordCategoryWeight     = 5.0
	keywordDescriptionWeight  = 2.5
	keywordCraftingWeight     = 3.5
	keywordAcquisitionWeight  = 3.0
	keywordNotesWeight        = 1.0
	keywordExactNameBonus     = 20.0
	keywordPartialNameBonus   = 8.0
	keywordCategoryMatchBonus = 5.0
)

var (
	keywordEnglishRE = regexp.MustCompile(`[a-zA-Z0-9]{2,}`)
	keywordCleanupRE = regexp.MustCompile(`[*#>` + "`" + `\[\]()!_~|]`)
	keywordSpaceRE   = regexp.MustCompile(`\s+`)
)

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
	Filename      string   `json:"filename"`
}

type wikiIndex struct {
	Version       int                           `json:"version"`
	Pages         map[string]wikiPageInfo       `json:"pages"`
	CategoryIndex map[string][]string           `json:"category_index"`
	TermIndex     map[string]map[string]float64 `json:"term_index"`
}

type keywordWikiSearcher struct {
	pagesDir  string
	indexPath string

	mu        sync.RWMutex
	index     *wikiIndex
	idleTimer *time.Timer
}

func newKeywordWikiSearcher(pagesDir, indexPath string) *keywordWikiSearcher {
	return &keywordWikiSearcher{pagesDir: pagesDir, indexPath: indexPath}
}

func (s *keywordWikiSearcher) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.loadLocked()
}

func (s *keywordWikiSearcher) unload() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.unloadLocked()
}

func (s *keywordWikiSearcher) unloadLocked() {
	if s.index != nil {
		logger.Logger.Infof("释放关键词搜索索引内存 (%d 篇文档)", len(s.index.Pages))
		s.index = nil
	}
	s.stopIdleTimer()
}

func (s *keywordWikiSearcher) isLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.index != nil
}

func (s *keywordWikiSearcher) resetIdleTimer() {
	if s.idleTimer != nil {
		s.idleTimer.Stop()
	}
	s.idleTimer = time.AfterFunc(keywordIndexIdleTimeout, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.unloadLocked()
	})
}

func (s *keywordWikiSearcher) stopIdleTimer() {
	if s.idleTimer != nil {
		s.idleTimer.Stop()
		s.idleTimer = nil
	}
}

func (s *keywordWikiSearcher) loadLocked() error {
	if s.index != nil {
		return nil
	}

	data, err := os.ReadFile(s.indexPath)
	if err != nil {
		return fmt.Errorf("搜索索引文件不存在: %w", err)
	}

	var idx wikiIndex
	if err = json.Unmarshal(data, &idx); err != nil {
		return fmt.Errorf("解析搜索索引失败: %w", err)
	}
	if idx.Version != keywordIndexVersion {
		return fmt.Errorf("关键词索引版本不匹配: got %d, want %d", idx.Version, keywordIndexVersion)
	}
	if idx.Pages == nil || idx.TermIndex == nil {
		return fmt.Errorf("关键词索引内容不完整")
	}

	s.index = &idx
	s.resetIdleTimer()
	logger.Logger.Infof("已加载关键词搜索索引: %d 篇文档, %d 个词条", len(idx.Pages), len(idx.TermIndex))
	return nil
}

func (s *keywordWikiSearcher) search(query string, maxResults int) ([]wikiResult, error) {
	query = strings.TrimSpace(query)
	if query == "" || maxResults <= 0 {
		return []wikiResult{}, nil
	}

	if err := s.ensureIndex(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	idx := s.index
	s.resetIdleTimer()
	s.mu.Unlock()

	queryTerms := tokenizeQuery(query)
	scores := make(map[string]float64)
	for term, queryWeight := range queryTerms {
		for filename, indexScore := range idx.TermIndex[term] {
			scores[filename] += queryWeight * indexScore
		}
	}

	normalizedQuery := normalizeForMatch(query)
	for filename, info := range idx.Pages {
		normalizedTitle := normalizeForMatch(info.Title)
		switch {
		case normalizedTitle == normalizedQuery:
			scores[filename] += keywordExactNameBonus
		case normalizedTitle != "" && strings.Contains(normalizedQuery, normalizedTitle):
			scores[filename] += keywordPartialNameBonus
		case normalizedQuery != "" && strings.Contains(normalizedTitle, normalizedQuery):
			scores[filename] += keywordPartialNameBonus
		}
		for _, category := range info.Categories {
			if normalizedCategory := normalizeForMatch(category); normalizedCategory != "" && strings.Contains(normalizedQuery, normalizedCategory) {
				scores[filename] += keywordCategoryMatchBonus
			}
		}
	}

	type scoredFile struct {
		filename string
		score    float64
	}
	ranked := make([]scoredFile, 0, len(scores))
	for filename, score := range scores {
		if score > 0 {
			ranked = append(ranked, scoredFile{filename: filename, score: score})
		}
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].score == ranked[j].score {
			return ranked[i].filename < ranked[j].filename
		}
		return ranked[i].score > ranked[j].score
	})

	results := make([]wikiResult, 0, min(maxResults, len(ranked)))
	for _, item := range ranked {
		info, ok := idx.Pages[item.filename]
		if !ok {
			continue
		}
		results = append(results, wikiResult{
			Title:      info.Title,
			Filename:   info.Filename,
			Categories: info.Categories,
			Content:    info.Content,
			ContentLen: info.ContentLength,
			Score:      roundFloat(item.score, 3),
		})
		if len(results) == maxResults {
			break
		}
	}
	return results, nil
}

func (s *keywordWikiSearcher) ensureIndex() error {
	s.mu.Lock()
	if s.index != nil {
		s.mu.Unlock()
		return nil
	}
	loadErr := s.loadLocked()
	s.mu.Unlock()
	if loadErr == nil {
		return nil
	}

	logger.Logger.Infof("关键词索引不可用，自动构建新索引: %v", loadErr)
	if err := s.buildIndex(false); err != nil {
		return fmt.Errorf("加载或构建关键词索引失败: %w", err)
	}
	return nil
}

func tokenizeQuery(query string) map[string]float64 {
	terms := make(map[string]float64)
	for term := range tokenizeText(query) {
		weight := 1.0
		length := utf8.RuneCountInString(term)
		if containsHan(term) {
			weight = float64(length - 1) // 2/3/4 字词依次为 1/2/3
		} else {
			weight = 2.0
		}
		if weight > terms[term] {
			terms[term] = weight
		}
	}
	return terms
}

func tokenizeText(text string) map[string]int {
	terms := make(map[string]int)
	for _, chunk := range findChineseChunks(text) {
		runes := []rune(chunk)
		for start := 0; start < len(runes); start++ {
			for length := 2; length <= 4 && start+length <= len(runes); length++ {
				terms[string(runes[start:start+length])]++
			}
		}
	}
	for _, word := range keywordEnglishRE.FindAllString(strings.ToLower(text), -1) {
		terms[word]++
	}
	return terms
}

func findChineseChunks(text string) []string {
	var chunks []string
	var current []rune
	for _, r := range text {
		if unicode.Is(unicode.Han, r) {
			current = append(current, r)
			continue
		}
		if len(current) > 0 {
			chunks = append(chunks, string(current))
			current = nil
		}
	}
	if len(current) > 0 {
		chunks = append(chunks, string(current))
	}
	return chunks
}

func containsHan(value string) bool {
	for _, r := range value {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}

func cleanForTokenize(text string) string {
	text = keywordCleanupRE.ReplaceAllString(text, " ")
	return keywordSpaceRE.ReplaceAllString(text, " ")
}

func normalizeForMatch(text string) string {
	return strings.ToLower(strings.Join(strings.Fields(cleanForTokenize(text)), ""))
}

const maxContextTokens = 8000

func formatWikiContext(results []wikiResult, maxTokens int) string {
	if len(results) == 0 {
		return ""
	}
	if maxTokens <= 0 {
		maxTokens = maxContextTokens
	}

	parts := []string{"# 饥荒 Wiki 参考文档"}
	totalChars := 0
	maxChars := maxTokens * 2
	for i, result := range results {
		header := formatResultHeader(i+1, result)
		body := result.Content
		entryChars := utf8.RuneCountInString(header) + utf8.RuneCountInString(body) + 10
		if totalChars+entryChars > maxChars {
			remaining := maxChars - totalChars - utf8.RuneCountInString(header) - 50
			if remaining <= 200 {
				break
			}
			runes := []rune(body)
			if len(runes) > remaining {
				body = string(runes[:remaining]) + "\n\n（内容已截断...）"
			}
		}
		parts = append(parts, "", header, "", body)
		totalChars += entryChars
	}
	return strings.Join(parts, "\n")
}

func formatResultHeader(index int, result wikiResult) string {
	header := fmt.Sprintf("## %d. %s", index, result.Title)
	if len(result.Categories) > 0 {
		header += "  [" + strings.Join(result.Categories, ", ") + "]"
	}
	return header
}

func roundFloat(value float64, decimals int) float64 {
	pow := math.Pow10(decimals)
	return math.Round(value*pow) / pow
}

func (s *keywordWikiSearcher) buildIndex(force bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !force {
		if _, err := os.Stat(s.indexPath); err == nil {
			if loadErr := s.loadLocked(); loadErr == nil {
				logger.Logger.Infof("关键词索引已存在且版本有效，无需重建")
				return nil
			} else {
				logger.Logger.Infof("丢弃旧关键词索引并重建: %v", loadErr)
			}
		}
	}

	start := time.Now()
	entries, err := os.ReadDir(s.pagesDir)
	if err != nil {
		return fmt.Errorf("扫描 Wiki 页面目录失败: %w", err)
	}

	mdFiles := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".md" {
			mdFiles = append(mdFiles, filepath.Join(s.pagesDir, entry.Name()))
		}
	}
	sort.Strings(mdFiles)

	idx := &wikiIndex{
		Version:       keywordIndexVersion,
		Pages:         make(map[string]wikiPageInfo, len(mdFiles)),
		CategoryIndex: make(map[string][]string),
		TermIndex:     make(map[string]map[string]float64),
	}
	documentScores := make(map[string]map[string]float64, len(mdFiles))

	for _, path := range mdFiles {
		filename := filepath.Base(path)
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			logger.Logger.Warnf("读取 %s 失败: %v", filename, readErr)
			continue
		}

		doc := parseWikiDocument(string(content), filename)
		idx.Pages[filename] = wikiPageInfo{
			Title:         doc.Name,
			Categories:    doc.Categories,
			Content:       doc.Content,
			ContentLength: utf8.RuneCountInString(doc.Content),
			Filename:      filename,
		}
		for _, category := range doc.Categories {
			idx.CategoryIndex[category] = append(idx.CategoryIndex[category], filename)
		}

		scores := make(map[string]float64)
		addFieldTerms(scores, doc.Name, keywordNameWeight)
		addFieldTerms(scores, strings.Join(doc.Categories, " "), keywordCategoryWeight)
		addFieldTerms(scores, doc.Description, keywordDescriptionWeight)
		addFieldTerms(scores, doc.Crafting, keywordCraftingWeight)
		addFieldTerms(scores, doc.Acquisition, keywordAcquisitionWeight)
		addFieldTerms(scores, doc.Notes, keywordNotesWeight)
		documentScores[filename] = scores
	}

	for filename, scores := range documentScores {
		for term, score := range scores {
			if idx.TermIndex[term] == nil {
				idx.TermIndex[term] = make(map[string]float64)
			}
			idx.TermIndex[term][filename] = score
		}
	}
	applyInverseDocumentFrequency(idx.TermIndex, len(idx.Pages))

	if err = saveKeywordIndex(s.indexPath, idx); err != nil {
		return err
	}
	s.index = idx
	s.resetIdleTimer()
	logger.Logger.Infof("关键词索引构建完成: %d 篇文档, %d 个词条, 耗时 %.1fs", len(idx.Pages), len(idx.TermIndex), time.Since(start).Seconds())
	return nil
}

func addFieldTerms(scores map[string]float64, text string, fieldWeight float64) {
	for term, count := range tokenizeText(cleanForTokenize(text)) {
		scores[term] += fieldWeight * (1 + math.Log(float64(count)))
	}
}

func applyInverseDocumentFrequency(index map[string]map[string]float64, documentCount int) {
	for _, postings := range index {
		idf := math.Log((float64(documentCount)+1)/(float64(len(postings))+1)) + 1
		for filename, score := range postings {
			postings[filename] = score * idf
		}
	}
}

func saveKeywordIndex(indexPath string, idx *wikiIndex) error {
	dir := filepath.Dir(indexPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建索引目录失败: %w", err)
	}
	data, err := json.Marshal(idx)
	if err != nil {
		return fmt.Errorf("序列化搜索索引失败: %w", err)
	}

	tempFile, err := os.CreateTemp(dir, ".search-index-*.tmp")
	if err != nil {
		return fmt.Errorf("创建临时索引文件失败: %w", err)
	}
	tempPath := tempFile.Name()
	defer os.Remove(tempPath)

	if err = tempFile.Chmod(0644); err == nil {
		_, err = tempFile.Write(data)
	}
	if closeErr := tempFile.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("保存搜索索引失败: %w", err)
	}
	if err = os.Rename(tempPath, indexPath); err != nil {
		return fmt.Errorf("替换搜索索引失败: %w", err)
	}
	return nil
}
