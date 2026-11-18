package diagnosis

import (
	"dst-management-platform-api/logger"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

//go:embed kb.json
var kbJSON []byte

// KBText 是中英双语文本。
type KBText struct {
	ZH string `json:"zh"`
	EN string `json:"en"`
}

// KBTextList 是中英双语文本列表。
type KBTextList struct {
	ZH []string `json:"zh"`
	EN []string `json:"en"`
}

// KBActionHint 是知识库给出的修复动作建议。
type KBActionHint struct {
	ActionID             string `json:"actionID"`
	Risk                 string `json:"risk"`
	RequiresConfirmation bool   `json:"requiresConfirmation"`
	Reason               KBText `json:"reason"`
}

// KBEntry 是一条可检索的诊断知识库条目，内容由维护者手动更新。
type KBEntry struct {
	ID          string         `json:"id"`
	Enabled     bool           `json:"enabled"`
	Category    string         `json:"category"`
	Priority    int            `json:"priority"`
	Severity    string         `json:"severity"`
	Title       KBText         `json:"title"`
	Symptoms    KBTextList     `json:"symptoms"`
	Checks      KBTextList     `json:"checks"`
	Causes      KBTextList     `json:"causes"`
	Solutions   KBTextList     `json:"solutions"`
	ManualOnly  bool           `json:"manualOnly"`
	ActionHints []KBActionHint `json:"actionHints,omitempty"`
	Risk        KBText         `json:"risk"`
	Tags        []string       `json:"tags,omitempty"`
	UpdatedAt   string         `json:"updatedAt"`
}

// KnowledgeBase 是内嵌的诊断知识库。
type KnowledgeBase struct {
	SchemaVersion int       `json:"schemaVersion"`
	Entries       []KBEntry `json:"entries"`
}

// KBView 是提供给模型的检索结果摘要。
type KBView struct {
	ID          string             `json:"id"`
	Title       string             `json:"title"`
	Severity    string             `json:"severity"`
	Category    string             `json:"category"`
	Symptoms    []string           `json:"symptoms,omitempty"`
	Checks      []string           `json:"checks,omitempty"`
	Causes      []string           `json:"causes,omitempty"`
	Solutions   []string           `json:"solutions,omitempty"`
	Risk        string             `json:"risk,omitempty"`
	ManualOnly  bool               `json:"manualOnly"`
	ActionHints []KBActionHintView `json:"actionHints,omitempty"`
	Tags        []string           `json:"tags,omitempty"`
	UpdatedAt   string             `json:"updatedAt,omitempty"`
}

// KBActionHintView 是动作建议的可读形式。
type KBActionHintView struct {
	ActionID             string `json:"actionID"`
	Risk                 string `json:"risk"`
	RequiresConfirmation bool   `json:"requiresConfirmation"`
	Reason               string `json:"reason,omitempty"`
}

func (t KBText) Get(lang string) string {
	if lang == "en" {
		if t.EN != "" {
			return t.EN
		}
		return t.ZH
	}
	if t.ZH != "" {
		return t.ZH
	}
	return t.EN
}

func (t KBTextList) Get(lang string) []string {
	if lang == "en" {
		if len(t.EN) > 0 {
			return t.EN
		}
		return t.ZH
	}
	if len(t.ZH) > 0 {
		return t.ZH
	}
	return t.EN
}

func loadKnowledgeBase() (*KnowledgeBase, error) {
	var kb KnowledgeBase
	if err := json.Unmarshal(kbJSON, &kb); err != nil {
		return nil, fmt.Errorf("解析诊断知识库失败: %w", err)
	}
	if kb.SchemaVersion != 1 {
		return nil, fmt.Errorf("不支持的诊断知识库版本: %d", kb.SchemaVersion)
	}
	if len(kb.Entries) == 0 {
		return nil, fmt.Errorf("诊断知识库为空")
	}
	if len(kb.Entries) > 500 {
		return nil, fmt.Errorf("诊断知识库条目过多: %d", len(kb.Entries))
	}

	seen := make(map[string]struct{}, len(kb.Entries))
	for i := range kb.Entries {
		entry := &kb.Entries[i]
		if err := validateKBEntry(entry); err != nil {
			return nil, fmt.Errorf("知识库条目 %q 无效: %w", entry.ID, err)
		}
		if _, ok := seen[entry.ID]; ok {
			return nil, fmt.Errorf("知识库条目 ID 重复: %s", entry.ID)
		}
		seen[entry.ID] = struct{}{}
	}
	sort.SliceStable(kb.Entries, func(i, j int) bool {
		return kb.Entries[i].Priority > kb.Entries[j].Priority
	})
	return &kb, nil
}

func validateKBEntry(entry *KBEntry) error {
	if strings.TrimSpace(entry.ID) == "" {
		return fmt.Errorf("id 不能为空")
	}
	if entry.Severity != "low" && entry.Severity != "medium" && entry.Severity != "high" && entry.Severity != "critical" {
		return fmt.Errorf("severity 非法")
	}
	if entry.Title.ZH == "" || entry.Title.EN == "" {
		return fmt.Errorf("title 必须包含 zh 和 en")
	}
	if len(entry.Symptoms.ZH) == 0 || len(entry.Symptoms.EN) == 0 {
		return fmt.Errorf("symptoms 必须包含 zh 和 en")
	}
	if len(entry.Causes.ZH) == 0 || len(entry.Causes.EN) == 0 {
		return fmt.Errorf("causes 必须包含 zh 和 en")
	}
	if len(entry.Solutions.ZH) == 0 || len(entry.Solutions.EN) == 0 {
		return fmt.Errorf("solutions 必须包含 zh 和 en")
	}
	if entry.Risk.ZH == "" || entry.Risk.EN == "" {
		return fmt.Errorf("risk 必须包含 zh 和 en")
	}
	if entry.ManualOnly && len(entry.ActionHints) > 0 {
		return fmt.Errorf("manualOnly=true 时不能配置 actionHints")
	}
	for _, hint := range entry.ActionHints {
		if strings.TrimSpace(hint.ActionID) == "" {
			return fmt.Errorf("actionHints.actionID 不能为空")
		}
		if hint.Reason.ZH == "" || hint.Reason.EN == "" {
			return fmt.Errorf("actionHints.reason 必须包含 zh 和 en")
		}
	}
	return nil
}

// Search 按标题、症状、检查项、标签做简单关键词检索，供模型主动查询。
func (kb *KnowledgeBase) Search(query, lang string, limit int) []KBView {
	if kb == nil {
		return nil
	}
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return nil
	}
	if limit <= 0 || limit > 10 {
		limit = 3
	}
	terms := strings.Fields(query)
	type scored struct {
		entry *KBEntry
		score int
	}
	var results []scored
	for i := range kb.Entries {
		entry := &kb.Entries[i]
		if !entry.Enabled {
			continue
		}
		haystack := strings.ToLower(strings.Join([]string{
			entry.ID,
			entry.Title.ZH,
			entry.Title.EN,
			strings.Join(entry.Symptoms.ZH, " "),
			strings.Join(entry.Symptoms.EN, " "),
			strings.Join(entry.Tags, " "),
			strings.Join(entry.Checks.ZH, " "),
			strings.Join(entry.Checks.EN, " "),
		}, " "))
		score := 0
		for _, term := range terms {
			if strings.Contains(haystack, term) {
				score++
			}
		}
		if score > 0 {
			results = append(results, scored{entry: entry, score: score})
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].score == results[j].score {
			return results[i].entry.Priority > results[j].entry.Priority
		}
		return results[i].score > results[j].score
	})
	if len(results) > limit {
		results = results[:limit]
	}
	views := make([]KBView, 0, len(results))
	for _, result := range results {
		views = append(views, kb.view(result.entry, lang))
	}
	return views
}

func (kb *KnowledgeBase) view(entry *KBEntry, lang string) KBView {
	view := KBView{
		ID:         entry.ID,
		Title:      entry.Title.Get(lang),
		Severity:   entry.Severity,
		Category:   entry.Category,
		Symptoms:   entry.Symptoms.Get(lang),
		Checks:     entry.Checks.Get(lang),
		Causes:     entry.Causes.Get(lang),
		Solutions:  entry.Solutions.Get(lang),
		Risk:       entry.Risk.Get(lang),
		ManualOnly: entry.ManualOnly,
		Tags:       entry.Tags,
		UpdatedAt:  entry.UpdatedAt,
	}
	for _, hint := range entry.ActionHints {
		view.ActionHints = append(view.ActionHints, KBActionHintView{
			ActionID:             hint.ActionID,
			Risk:                 hint.Risk,
			RequiresConfirmation: hint.RequiresConfirmation,
			Reason:               hint.Reason.Get(lang),
		})
	}
	return view
}

func logKBLoaded(kb *KnowledgeBase) {
	if kb == nil {
		return
	}
	logger.Logger.Infof("AI 诊断知识库加载完成，共 %d 条条目", len(kb.Entries))
}
