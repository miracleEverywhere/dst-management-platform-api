package platform

import (
	"bufio"
	"dst-management-platform-api/utils"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

type poEntry struct {
	MsgID  string
	MsgStr string
}

// parseChinesePo 解析中文 PO 文件，返回大写 prefab -> {英文名, 中文名} 的映射
// 只解析 msgctxt 以 STRINGS.NAMES. 开头的条目
func parseChinesePo(filePath string) (map[string]poEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开PO文件失败: %v", err)
	}
	defer file.Close()

	result := make(map[string]poEntry)
	scanner := bufio.NewScanner(file)

	var currentCtxt, currentMsgID, currentMsgStr string

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "msgctxt "):
			// 遇到新条目，保存上一个（如果有）
			if currentCtxt != "" && strings.HasPrefix(currentCtxt, "STRINGS.NAMES.") {
				prefab := strings.TrimPrefix(currentCtxt, "STRINGS.NAMES.")
				result[prefab] = poEntry{MsgID: currentMsgID, MsgStr: currentMsgStr}
			}
			currentCtxt = parsePoValue(line)
			currentMsgID = ""
			currentMsgStr = ""
		case strings.HasPrefix(line, "msgid "):
			currentMsgID = parsePoValue(line)
		case strings.HasPrefix(line, "msgstr "):
			currentMsgStr = parsePoValue(line)
		}
	}

	// 保存最后一个条目
	if currentCtxt != "" && strings.HasPrefix(currentCtxt, "STRINGS.NAMES.") {
		prefab := strings.TrimPrefix(currentCtxt, "STRINGS.NAMES.")
		result[prefab] = poEntry{MsgID: currentMsgID, MsgStr: currentMsgStr}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取PO文件失败: %v", err)
	}

	return result, nil
}

// parsePoValue 解析 PO 文件中单行带引号的值，如 "Birchnut" -> Birchnut
func parsePoValue(line string) string {
	start := strings.Index(line, "\"")
	if start == -1 {
		return ""
	}
	end := strings.LastIndex(line, "\"")
	if end <= start {
		return ""
	}
	return line[start+1 : end]
}

type itemCategory struct {
	Category string   `json:"category"` // 分类名 (如 tool, food, equip)
	Items    []string `json:"items"`    // 该分类下的物品代码列表
}

// tmiItemLists 解析 mod 的 Lua 物品列表文件，返回按分类组织的物品代码
func tmiItemLists() ([]itemCategory, error) {
	entries, err := os.ReadDir(tmirPath)
	if err != nil {
		return nil, fmt.Errorf("读取物品列表目录失败: %v", err)
	}

	var categories []itemCategory

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "itemlist_") || !strings.HasSuffix(entry.Name(), ".lua") {
			continue
		}

		// 从文件名提取分类名: itemlist_tool.lua → tool
		category := strings.TrimPrefix(entry.Name(), "itemlist_")
		category = strings.TrimSuffix(category, ".lua")

		filePath := filepath.Join(tmirPath, entry.Name())
		items, err := parseLuaItemList(filePath)
		if err != nil {
			return nil, fmt.Errorf("解析 %s 失败: %v", entry.Name(), err)
		}

		categories = append(categories, itemCategory{
			Category: category,
			Items:    items,
		})
	}

	// 按分类名排序，保证输出稳定
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Category < categories[j].Category
	})

	return categories, nil
}

// parseLuaItemList 解析单个 Lua 物品列表文件，格式为 return{"item1","item2",...}
func parseLuaItemList(filePath string) ([]string, error) {
	L := utils.NewSafeLuaState()
	defer L.Close()

	if err := L.DoFile(filePath); err != nil {
		return nil, fmt.Errorf("执行 Lua 文件失败: %v", err)
	}

	// 获取返回值 (Lua 栈顶)
	ret := L.Get(-1)
	if ret.Type() != lua.LTTable {
		return nil, fmt.Errorf("返回值不是 table 类型: %s", ret.Type().String())
	}

	table := ret.(*lua.LTable)
	var items []string

	table.ForEach(func(_, value lua.LValue) {
		if value.Type() == lua.LTString {
			items = append(items, value.String())
		}
	})

	return items, nil
}
