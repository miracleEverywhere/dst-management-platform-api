package aichat

import (
	"path/filepath"
	"regexp"
	"strings"
)

const (
	wikiSectionCategory    = "分类"
	wikiSectionDescription = "描述"
	wikiSectionCrafting    = "制作材料及解锁"
	wikiSectionAcquisition = "如何获取"
	wikiSectionNotes       = "备注"
)

var (
	wikiSectionHeadingRE = regexp.MustCompile(`^\s*#\s*(分类|描述|制作材料及解锁|如何获取|备注)\s*$`)
	wikiMarkdownLinkRE   = regexp.MustCompile(`\[([^\]]+)\]\([^)]+\)`)
	wikiCategorySplitRE  = regexp.MustCompile(`[\n,，、;；|/]+`)
)

type wikiDocument struct {
	Name        string
	Categories  []string
	Description string
	Crafting    string
	Acquisition string
	Notes       string
	Content     string
}

func parseWikiDocument(rawContent, filename string) wikiDocument {
	sections := make(map[string][]string, 5)
	currentSection := ""

	for _, line := range strings.Split(strings.ReplaceAll(rawContent, "\r\n", "\n"), "\n") {
		if match := wikiSectionHeadingRE.FindStringSubmatch(line); len(match) == 2 {
			currentSection = match[1]
			continue
		}
		if currentSection != "" {
			sections[currentSection] = append(sections[currentSection], line)
		}
	}

	return wikiDocument{
		Name:        wikiPageName(filename),
		Categories:  parseWikiCategories(strings.Join(sections[wikiSectionCategory], "\n")),
		Description: cleanWikiSection(sections[wikiSectionDescription]),
		Crafting:    cleanWikiSection(sections[wikiSectionCrafting]),
		Acquisition: cleanWikiSection(sections[wikiSectionAcquisition]),
		Notes:       cleanWikiSection(sections[wikiSectionNotes]),
		Content:     strings.TrimSpace(rawContent),
	}
}

func wikiPageName(filename string) string {
	name := filepath.Base(filename)
	return strings.TrimSpace(strings.TrimSuffix(name, filepath.Ext(name)))
}

func cleanWikiSection(lines []string) string {
	return strings.TrimSpace(strings.Join(lines, "\n"))
}

func parseWikiCategories(section string) []string {
	section = wikiMarkdownLinkRE.ReplaceAllString(section, "$1")
	parts := wikiCategorySplitRE.Split(section, -1)
	categories := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		category := strings.TrimSpace(strings.TrimLeft(part, "-* "))
		if category == "" {
			continue
		}
		key := strings.ToLower(category)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		categories = append(categories, category)
	}
	return categories
}
