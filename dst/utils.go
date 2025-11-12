package dst

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"
)

type Game struct {
	room    *models.Room
	worlds  *[]models.World
	setting *models.RoomSetting
	lang    string
	roomSaveData
	worldSaveData []worldSaveData
	playerSaveData
	modSaveData
	// room全局文件锁
	roomMutex sync.Mutex
	// world全局文件锁
	worldMutex sync.Mutex
	// player全局文件锁
	playerMutex sync.Mutex
	// acf文件锁
	acfMutex sync.Mutex
	// mod 文件、map锁
	modMutex sync.Mutex
}

func NewGameController(room *models.Room, worlds *[]models.World, setting *models.RoomSetting, lang string) *Game {
	game := &Game{
		room:    room,
		worlds:  worlds,
		setting: setting,
		lang:    lang,
	}

	game.initInfo()

	return game
}

func (g *Game) initInfo() {
	// room
	g.clusterName = fmt.Sprintf("Cluster_%d", g.room.ID)
	g.clusterPath = fmt.Sprintf("%s/%s", utils.ClusterPath, g.clusterName)
	g.clusterIniPath = fmt.Sprintf("%s/cluster.ini", g.clusterPath)
	g.clusterTokenTxtPath = fmt.Sprintf("%s/cluster_token.txt", g.clusterPath)

	// worlds
	for _, world := range *g.worlds {
		worldPath := fmt.Sprintf("%s/%s", g.clusterPath, world.WorldName)
		serverIniPath := fmt.Sprintf("%s/server.ini", worldPath)
		levelDataOverridePath := fmt.Sprintf("%s/leveldataoverride.lua", worldPath)
		modOverridesPath := fmt.Sprintf("%s/modoverrides.lua", worldPath)
		screenName := fmt.Sprintf("DMP_%s_%s", g.clusterName, world.WorldName)

		var startCmd string
		switch g.setting.StartType {
		case "32-bit":
			startCmd = fmt.Sprintf("cd dst/bin/ && screen -d -h 200 -m -S %s ./dontstarve_dedicated_server_nullrenderer -console -cluster %s -shard %s", screenName, g.clusterName, world.WorldName)
		case "64-bit":
			startCmd = fmt.Sprintf("cd dst/bin64/ && screen -d -h 200 -m -S %s ./dontstarve_dedicated_server_nullrenderer_x64 -console -cluster %s -shard %s", screenName, g.clusterName, world.WorldName)
		default:
			startCmd = "exit 1"
		}

		g.worldSaveData = append(g.worldSaveData, worldSaveData{
			worldPath:             worldPath,
			serverIniPath:         serverIniPath,
			levelDataOverridePath: levelDataOverridePath,
			modOverridesPath:      modOverridesPath,
			startCmd:              startCmd,
			screenName:            screenName,
			World:                 world,
		})
	}

	// players
	g.adminlistPath = fmt.Sprintf("%s/adminlist.txt", g.clusterPath)
	g.whitelistPath = fmt.Sprintf("%s/whitelist.txt", g.clusterPath)
	g.blocklistPath = fmt.Sprintf("%s/blocklist.txt", g.clusterPath)
	g.adminlist = getPlayerList(g.adminlistPath)
	g.whitelist = getPlayerList(g.whitelistPath)
	g.blocklist = getPlayerList(g.blocklistPath)

	// mods
	g.ugcPath = fmt.Sprintf("%s/dst/ugc_mods/%s", db.CurrentDir, g.clusterName)
}

// ============== //
// steam.acf
// ============== //

type AcfParser struct {
	content     string
	AppWorkshop *AppWorkshop
}

func NewAcfParser(c string) *AcfParser {
	p := &AcfParser{
		content:     c,
		AppWorkshop: &AppWorkshop{},
	}

	p.parse()

	return p
}

type AppWorkshop struct {
	AppID                  string
	SizeOnDisk             string
	NeedsUpdate            string
	NeedsDownload          string
	TimeLastUpdated        string
	TimeLastAppRan         string
	LastBuildID            string
	WorkshopItemsInstalled []ItemInstalled
	WorkshopItemDetails    []ItemDetails
}

type ItemInstalled struct {
	ID          string
	Size        string
	TimeUpdated string
	Manifest    string
}

type ItemDetails struct {
	ID                string
	Manifest          string
	TimeUpdated       string
	TimeTouched       string
	LatestTimeUpdated string
	LatestManifest    string
}

func (p *AcfParser) parse() {
	lines := strings.Split(p.content, "\n")
	appWorkshop := &AppWorkshop{
		WorkshopItemsInstalled: []ItemInstalled{},
		WorkshopItemDetails:    []ItemDetails{},
	}
	var currentItemID string
	var currentInstalled ItemInstalled
	var currentDetail ItemDetails
	inItemsInstalled := false
	inItemDetails := false

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "\"WorkshopItemsInstalled\"") {
			inItemsInstalled = true
			inItemDetails = false
			continue
		}
		if strings.HasPrefix(line, "\"WorkshopItemDetails\"") {
			inItemsInstalled = false
			inItemDetails = true
			continue
		}
		if inItemsInstalled || inItemDetails {
			line = strings.ReplaceAll(line, "\"", "")
			line = strings.ReplaceAll(line, "\t", "")
			if line == "{" {
				continue
			}
			if line == "}" {
				continue
			}

			intRe := regexp.MustCompile(`^(\d+)$`)
			intReMatches := intRe.FindStringSubmatch(line)
			if intReMatches != nil {
				currentItemID = intReMatches[1]
				continue
			}
			if currentItemID != "" {
				sizeRe := regexp.MustCompile(`^size(\d+)$`)
				sizeReMatches := sizeRe.FindStringSubmatch(line)
				if sizeReMatches != nil {
					currentInstalled.Size = sizeReMatches[1]

					continue
				}

				timeupdatedRe := regexp.MustCompile(`^timeupdated(\d+)$`)
				timeupdatedReMatches := timeupdatedRe.FindStringSubmatch(line)
				if timeupdatedReMatches != nil {
					if inItemsInstalled {
						currentInstalled.TimeUpdated = timeupdatedReMatches[1]
					}
					if inItemDetails {
						currentDetail.TimeUpdated = timeupdatedReMatches[1]
					}

					continue
				}

				manifestRe := regexp.MustCompile(`^manifest(\d+)$`)
				manifestReMatches := manifestRe.FindStringSubmatch(line)
				if manifestReMatches != nil {
					if inItemsInstalled {
						currentInstalled.Manifest = manifestReMatches[1]
						currentInstalled.ID = currentItemID
						appWorkshop.WorkshopItemsInstalled = append(appWorkshop.WorkshopItemsInstalled, currentInstalled)
						currentInstalled = ItemInstalled{}
						currentItemID = ""
					}
					if inItemDetails {
						currentDetail.Manifest = manifestReMatches[1]
					}

					continue
				}

				timetouchedRe := regexp.MustCompile(`^timetouched(\d+)$`)
				timetouchedReMatches := timetouchedRe.FindStringSubmatch(line)
				if timetouchedReMatches != nil {
					currentDetail.TimeTouched = timetouchedReMatches[1]

					continue
				}

				latestTimeupdatedRe := regexp.MustCompile(`^latest_timeupdated(\d+)$`)
				latestTimeupdatedReMatches := latestTimeupdatedRe.FindStringSubmatch(line)
				if latestTimeupdatedReMatches != nil {
					currentDetail.LatestTimeUpdated = latestTimeupdatedReMatches[1]

					continue
				}

				latestManifestRe := regexp.MustCompile(`^latest_manifest(\d+)$`)
				latestManifestReMatches := latestManifestRe.FindStringSubmatch(line)
				if latestManifestReMatches != nil {
					currentDetail.LatestManifest = latestManifestReMatches[1]
					currentDetail.ID = currentItemID
					appWorkshop.WorkshopItemDetails = append(appWorkshop.WorkshopItemDetails, currentDetail)
					currentDetail = ItemDetails{}
					currentItemID = ""

					continue
				}

			}
		} else {
			line = strings.ReplaceAll(line, "\"", "")
			line = strings.ReplaceAll(line, "\t", "")

			appidRe := regexp.MustCompile(`^appid(\d+)$`)
			appidReMatches := appidRe.FindStringSubmatch(line)
			if appidReMatches != nil {
				appWorkshop.AppID = appidReMatches[1]

				continue
			}

			sizeOnDiskRe := regexp.MustCompile(`^SizeOnDisk(\d+)$`)
			sizeOnDiskReMatches := sizeOnDiskRe.FindStringSubmatch(line)
			if sizeOnDiskReMatches != nil {
				appWorkshop.SizeOnDisk = sizeOnDiskReMatches[1]

				continue
			}

			needsUpdateRe := regexp.MustCompile(`^NeedsUpdate(\d+)$`)
			needsUpdateReMatches := needsUpdateRe.FindStringSubmatch(line)
			if needsUpdateReMatches != nil {
				appWorkshop.NeedsUpdate = needsUpdateReMatches[1]

				continue
			}

			needsDownloadRe := regexp.MustCompile(`^NeedsDownload(\d+)$`)
			needsDownloadReMatches := needsDownloadRe.FindStringSubmatch(line)
			if needsDownloadReMatches != nil {
				appWorkshop.NeedsDownload = needsDownloadReMatches[1]

				continue
			}

			timeLastUpdatedRe := regexp.MustCompile(`^TimeLastUpdated(\d+)$`)
			timeLastUpdatedReMatches := timeLastUpdatedRe.FindStringSubmatch(line)
			if timeLastUpdatedReMatches != nil {
				appWorkshop.TimeLastUpdated = timeLastUpdatedReMatches[1]

				continue
			}

			timeLastAppRanRe := regexp.MustCompile(`^TimeLastAppRan(\d+)$`)
			timeLastAppRanReMatches := timeLastAppRanRe.FindStringSubmatch(line)
			if timeLastAppRanReMatches != nil {
				appWorkshop.TimeLastAppRan = timeLastAppRanReMatches[1]

				continue
			}

			lastBuildIDRe := regexp.MustCompile(`^LastBuildID(\d+)$`)
			lastBuildIDReMatches := lastBuildIDRe.FindStringSubmatch(line)
			if lastBuildIDReMatches != nil {
				appWorkshop.LastBuildID = lastBuildIDReMatches[1]

				continue
			}
		}
	}

	p.AppWorkshop = appWorkshop
}

func (p *AcfParser) FileContent() string {
	var (
		workshopItemsInstalled string
		workshopItemDetails    string
	)

	for _, itemInstalled := range p.AppWorkshop.WorkshopItemsInstalled {
		workshopItemsInstalled = workshopItemsInstalled + generateItemInstalled(itemInstalled)
	}

	for _, itemDetails := range p.AppWorkshop.WorkshopItemDetails {
		workshopItemDetails = workshopItemDetails + generateItemDetails(itemDetails)
	}

	content := `"AppWorkshop"
{
	"appid"		"322330"
	"SizeOnDisk"		"2071004"
	"NeedsUpdate"		"0"
	"NeedsDownload"		"0"
	"TimeLastUpdated"		"0"
	"TimeLastAppRan"		"0"
	"LastBuildID"		"0"
	"WorkshopItemsInstalled"
	{
` + workshopItemsInstalled + `
	}
	"WorkshopItemDetails"
	{
` + workshopItemDetails + `
	}
}`

	return content
}

func generateItemInstalled(i ItemInstalled) string {
	return `		"` + i.ID + `"
		{
			"size"		"` + i.Size + `"
			"timeupdated"		"` + i.TimeUpdated + `"
			"manifest"		"` + i.Manifest + `"
		}
`
}

func generateItemDetails(i ItemDetails) string {
	return `		"` + i.ID + `"
		{
			"manifest"		"` + i.Manifest + `"
			"timeupdated"		"` + i.TimeUpdated + `"
			"timetouched"		"` + i.TimeTouched + `"
			"latest_timeupdated"		"` + i.LatestTimeUpdated + `"
			"latest_manifest"		"` + i.LatestManifest + `"
		}
`
}

// ============== //
// modinfo.lua
// ============== //

type Option struct {
	Description string      `json:"description"`
	Data        interface{} `json:"data"`
	Hover       string      `json:"hover,omitempty"`
}

type ConfigurationOption struct {
	Name    string      `json:"name"`
	Label   string      `json:"label"`
	Hover   string      `json:"hover"`
	Options []Option    `json:"options"`
	Default interface{} `json:"default"`
}

type ModInfoParser struct {
	ModInfoLua    string `json:"modInfoLua"`
	Configuration *[]ConfigurationOption
}

func NewModInfoParser(luaPath string) (*ModInfoParser, error) {
	content, err := os.ReadFile(luaPath)
	if err != nil {
		return &ModInfoParser{}, err
	}

	m := &ModInfoParser{
		ModInfoLua: string(content),
	}

	return m, nil
}

// convertLuaValue 将 Lua 值转换为 Go 值
func convertLuaValue(lv lua.LValue) interface{} {
	switch v := lv.(type) {
	case lua.LBool:
		return bool(v)
	case lua.LNumber:
		return float64(v)
	case lua.LString:
		return string(v)
	case *lua.LTable:
		// 检查是数组还是字典
		dict := make(map[string]interface{})
		array := make([]interface{}, 0)

		isArray := true
		maxIndex := 0
		count := 0

		v.ForEach(func(key lua.LValue, value lua.LValue) {
			count++
			if num, ok := key.(lua.LNumber); ok {
				index := int(num)
				if index > maxIndex {
					maxIndex = index
				}
				// 如果索引不是连续整数，则视为字典
				if index != count {
					isArray = false
				}
				// 确保索引从1开始（Lua惯例）
				if count == 1 && index != 1 {
					isArray = false
				}
			} else {
				isArray = false
			}

			// 同时填充字典
			dict[key.String()] = convertLuaValue(value)
		})

		// 如果是数组且索引连续
		if isArray && maxIndex == count {
			// 按索引填充数组
			array = make([]interface{}, maxIndex)
			v.ForEach(func(key lua.LValue, value lua.LValue) {
				if num, ok := key.(lua.LNumber); ok {
					index := int(num) - 1 // Lua索引从1开始，Go从0开始
					if index >= 0 && index < maxIndex {
						array[index] = convertLuaValue(value)
					}
				}
			})
			return array
		}

		return dict
	case *lua.LFunction:
		return nil // 函数不转换
	case *lua.LUserData:
		return nil // 用户数据不转换
	default:
		return nil
	}
}

func (mf *ModInfoParser) Parse(lang string) error {
	var options []ConfigurationOption

	L := lua.NewState()
	defer L.Close()

	L.SetGlobal("locale", lua.LString(lang))
	// insight模组需要ChooseTranslationTable才能返回i18n
	L.SetGlobal("ChooseTranslationTable", L.NewFunction(func(L *lua.LState) int {
		tbl := L.ToTable(1)
		CTT := tbl.RawGetString(lang)
		if CTT != lua.LNil {
			L.Push(CTT)
		} else {
			L.Push(tbl.RawGetInt(1))
		}
		return 1
	}))

	// 加载并执行 Lua 脚本
	if err := L.DoString(mf.ModInfoLua); err != nil {
		logger.Logger.Debug("执行modinfo.lua失败", "err", err)
		return err
	}

	// 获取 configuration_options 表
	configOptions := L.GetGlobal("configuration_options")
	if configOptions.Type() != lua.LTTable {
		return fmt.Errorf("获取modinfo.lua中的configuration_options失败")
	}

	// 遍历 configuration_options 表
	table := configOptions.(*lua.LTable)
	table.ForEach(func(k lua.LValue, v lua.LValue) {
		if v.Type() == lua.LTTable {
			option := ConfigurationOption{}
			v.(*lua.LTable).ForEach(func(key lua.LValue, value lua.LValue) {
				switch key.String() {
				case "name":
					option.Name = value.String()
				case "label":
					option.Label = value.String()
				case "hover":
					option.Hover = value.String()
				case "default":
					option.Default = convertLuaValue(value)
				case "options":
					optionsTable := value.(*lua.LTable)
					optionsTable.ForEach(func(k lua.LValue, v lua.LValue) {
						if v.Type() == lua.LTTable {
							opt := Option{}
							v.(*lua.LTable).ForEach(func(key lua.LValue, value lua.LValue) {
								switch key.String() {
								case "description":
									opt.Description = value.String()
								case "data":
									opt.Data = convertLuaValue(value)
								case "hover":
									opt.Hover = value.String()
								}
							})
							option.Options = append(option.Options, opt)
						}
					})
				}
			})
			if option.Name != "" && option.Label != "" {
				options = append(options, option)
			}
		}
	})

	mf.Configuration = &options

	return nil
}

// ============== //
// modoverrides.lua
// ============== //

// ModORConfig 表示单个mod的配置
type ModORConfig struct {
	ConfigurationOptions map[string]interface{} `json:"configuration_options"`
	Enabled              bool                   `json:"enabled"`
}

// ModORCollection 表示整个mod集合
type ModORCollection map[string]*ModORConfig

// ModORParser Lua配置解析器
type ModORParser struct {
	L *lua.LState
}

// NewModORParser 创建新的解析器
func NewModORParser() *ModORParser {
	return &ModORParser{
		L: lua.NewState(),
	}
}

// close 关闭Lua状态
func (p *ModORParser) close() {
	if p.L != nil {
		p.L.Close()
	}
}

// Parse 解析Lua配置文件内容
func (p *ModORParser) Parse(luaContent, lang string) (ModORCollection, error) {
	// 执行Lua脚本
	p.L.SetGlobal("locale", lua.LString(lang))
	// insight模组需要ChooseTranslationTable才能返回i18n
	p.L.SetGlobal("ChooseTranslationTable", p.L.NewFunction(func(L *lua.LState) int {
		tbl := L.ToTable(1)
		CTT := tbl.RawGetString(lang)
		if CTT != lua.LNil {
			L.Push(CTT)
		} else {
			L.Push(tbl.RawGetInt(1))
		}
		return 1
	}))

	if err := p.L.DoString(luaContent); err != nil {
		logger.Logger.Debug("这里出问题?", "err", err)
		return nil, err
	}

	// 获取返回值（return的内容）
	luaTable := p.L.Get(-1)
	p.L.Pop(1)

	// 转换Lua table为Go结构
	return p.convertLuaTableToGo(luaTable)
}

// convertLuaTableToGo 将Lua table转换为Go结构
func (p *ModORParser) convertLuaTableToGo(lv lua.LValue) (ModORCollection, error) {
	if lv.Type() != lua.LTTable {
		return nil, nil
	}

	mods := make(ModORCollection)
	table := lv.(*lua.LTable)

	table.ForEach(func(key lua.LValue, value lua.LValue) {
		modID := key.String()
		if value.Type() == lua.LTTable {
			if modConfig := p.parseModConfig(value.(*lua.LTable)); modConfig != nil {
				mods[modID] = modConfig
			}
		}
	})

	return mods, nil
}

// parseModConfig 解析单个mod配置
func (p *ModORParser) parseModConfig(table *lua.LTable) *ModORConfig {
	config := &ModORConfig{
		ConfigurationOptions: make(map[string]interface{}),
	}

	table.ForEach(func(key lua.LValue, value lua.LValue) {
		keyStr := key.String()

		switch keyStr {
		case "enabled":
			if value.Type() == lua.LTBool {
				config.Enabled = bool(value.(lua.LBool))
			}
		case "configuration_options":
			if value.Type() == lua.LTTable {
				config.ConfigurationOptions = p.parseConfigurationOptions(value.(*lua.LTable))
			}
		}
	})

	return config
}

// parseConfigurationOptions 解析配置选项
func (p *ModORParser) parseConfigurationOptions(table *lua.LTable) map[string]interface{} {
	options := make(map[string]interface{})

	table.ForEach(func(key lua.LValue, value lua.LValue) {
		keyStr := key.String()
		options[keyStr] = p.convertLuaValue(value)
	})

	return options
}

// convertLuaValue 转换Lua值到Go值
func (p *ModORParser) convertLuaValue(lv lua.LValue) interface{} {
	switch v := lv.(type) {
	case lua.LBool:
		return bool(v)
	case lua.LNumber:
		return float64(v)
	case lua.LString:
		return string(v)
	case *lua.LTable:
		// 判断是数组还是map
		if p.isArray(v) {
			return p.convertLuaArray(v)
		}
		return p.convertLuaMap(v)
	default:
		return lv.String()
	}
}

// isArray 判断table是否是数组
func (p *ModORParser) isArray(table *lua.LTable) bool {
	// 收集所有的数字键
	var numericKeys []int
	hasNonNumericKey := false

	table.ForEach(func(key lua.LValue, value lua.LValue) {
		if key.Type() == lua.LTNumber {
			if num := float64(key.(lua.LNumber)); num == float64(int(num)) && num > 0 {
				numericKeys = append(numericKeys, int(num))
			} else {
				hasNonNumericKey = true
			}
		} else {
			hasNonNumericKey = true
		}
	})

	// 如果有非数字键，则不是数组
	if hasNonNumericKey {
		return false
	}

	// 如果没有数字键，也不是数组
	if len(numericKeys) == 0 {
		return false
	}

	// 对数字键排序
	sort.Ints(numericKeys)

	// 检查是否是从1开始的连续整数
	for i, key := range numericKeys {
		if key != i+1 {
			return false
		}
	}

	return true
}

// convertLuaArray 转换Lua数组为Go slice
func (p *ModORParser) convertLuaArray(table *lua.LTable) []interface{} {
	var arr []interface{}
	maxIndex := 0

	// 先找出最大索引
	table.ForEach(func(key lua.LValue, value lua.LValue) {
		if key.Type() == lua.LTNumber {
			if num := float64(key.(lua.LNumber)); num == float64(int(num)) && int(num) > maxIndex {
				maxIndex = int(num)
			}
		}
	})

	// 初始化切片
	arr = make([]interface{}, maxIndex)

	// 填充数组
	table.ForEach(func(key lua.LValue, value lua.LValue) {
		if key.Type() == lua.LTNumber {
			if num := float64(key.(lua.LNumber)); num == float64(int(num)) {
				idx := int(num)
				if idx >= 1 { // Lua数组通常从1开始
					arr[idx-1] = p.convertLuaValue(value)
				}
			}
		}
	})

	return arr
}

// convertLuaMap 转换Lua map为Go map
func (p *ModORParser) convertLuaMap(table *lua.LTable) map[string]interface{} {
	m := make(map[string]interface{})

	table.ForEach(func(key lua.LValue, value lua.LValue) {
		keyStr := key.String()
		m[keyStr] = p.convertLuaValue(value)
	})

	return m
}

// GetModConfig 获取指定workshop ID的mod配置
func (mc ModORCollection) GetModConfig(workshopID string) *ModORConfig {
	return mc[workshopID]
}

// IsModEnabled 检查指定workshop ID的mod是否启用
func (mc ModORCollection) IsModEnabled(workshopID string) bool {
	if config := mc[workshopID]; config != nil {
		return config.Enabled
	}
	return false
}

// GetConfigValue 获取指定mod的配置项值
func (mc ModORCollection) GetConfigValue(workshopID, configKey string) interface{} {
	if config := mc[workshopID]; config != nil {
		return config.ConfigurationOptions[configKey]
	}
	return nil
}

// GetNestedConfig 获取嵌套配置项的值
func (mc ModORCollection) GetNestedConfig(workshopID, parentKey, childKey string) interface{} {
	if config := mc[workshopID]; config != nil {
		if parent, ok := config.ConfigurationOptions[parentKey].(map[string]interface{}); ok {
			return parent[childKey]
		}
	}
	return nil
}

// AddModConfig 向ModCollection中添加或更新一个mod配置
func (mc ModORCollection) AddModConfig(workshopID string, config *ModORConfig) {
	mc[workshopID] = config
}

// ToLuaCode 将ModCollection转换为Lua代码
func (mc ModORCollection) ToLuaCode() string {
	var builder strings.Builder
	builder.WriteString("return {\n")

	// 将所有workshopID收集并排序，以便输出顺序一致
	var workshopIDs []string
	for workshopID := range mc {
		workshopIDs = append(workshopIDs, workshopID)
	}
	sort.Strings(workshopIDs)

	// 处理每个mod配置
	for i, workshopID := range workshopIDs {
		builder.WriteString(fmt.Sprintf("  [\"%s\"]={\n", workshopID))
		config := mc[workshopID]

		builder.WriteString("    configuration_options={\n")

		// 收集并排序配置选项键
		var optionKeys []string
		for key := range config.ConfigurationOptions {
			optionKeys = append(optionKeys, key)
		}
		sort.Strings(optionKeys)

		// 输出配置选项
		for j, key := range optionKeys {
			value := config.ConfigurationOptions[key]
			if j == len(optionKeys)-1 {
				// 最后一个配置选项不加逗号
				builder.WriteString(fmt.Sprintf("      %s=%s\n", formatLuaKey(key), formatLuaValue(value)))
			} else {
				builder.WriteString(fmt.Sprintf("      %s=%s,\n", formatLuaKey(key), formatLuaValue(value)))
			}
		}

		builder.WriteString("    },\n")
		builder.WriteString(fmt.Sprintf("    enabled=%t\n", config.Enabled))

		if i == len(workshopIDs)-1 {
			// 最后一个mod配置不加逗号
			builder.WriteString("  }\n")
		} else {
			builder.WriteString("  },\n")
		}
	}

	builder.WriteString("}")
	return builder.String()
}

// formatLuaValue 将Go值格式化为Lua值
func formatLuaValue(value interface{}) string {
	switch v := value.(type) {
	case bool:
		return strconv.FormatBool(v)
	case float64:
		// 检查是否为整数
		if v == float64(int64(v)) {
			return strconv.FormatInt(int64(v), 10)
		}
		return strconv.FormatFloat(v, 'g', -1, 64)
	case string:
		return fmt.Sprintf("\"%s\"", v)
	case []interface{}:
		// 数组格式
		var builder strings.Builder
		builder.WriteString("{")
		for i, item := range v {
			if i > 0 {
				builder.WriteString(",")
			}
			builder.WriteString(formatLuaValue(item))
		}
		builder.WriteString("}")
		return builder.String()
	case map[string]interface{}:
		// 表格式
		var builder strings.Builder
		builder.WriteString("{")
		first := true
		for key, item := range v {
			if !first {
				builder.WriteString(",")
			}
			// 检查键是否需要引号
			if isValidLuaIdentifier(key) {
				builder.WriteString(fmt.Sprintf("%s=%s", key, formatLuaValue(item)))
			} else {
				builder.WriteString(fmt.Sprintf("[\"%s\"]=%s", key, formatLuaValue(item)))
			}
			first = false
		}
		builder.WriteString("}")
		return builder.String()
	default:
		return fmt.Sprintf("\"%v\"", v)
	}
}

// isValidLuaIdentifier 检查字符串是否为有效的Lua标识符
func isValidLuaIdentifier(s string) bool {
	if len(s) == 0 {
		return false
	}

	// 第一个字符必须是字母或下划线
	firstChar := rune(s[0])
	if !unicode.IsLetter(firstChar) && firstChar != '_' {
		return false
	}

	// 后续字符可以是字母、数字或下划线
	for _, char := range s[1:] {
		if !unicode.IsLetter(char) && !unicode.IsDigit(char) && char != '_' {
			return false
		}
	}

	return true
}

func formatLuaKey(s string) string {
	if len(s) == 0 {
		return fmt.Sprintf("[\"\"]")
	}

	// 数字开头
	numRe := regexp.MustCompile(`^\d`)
	if numRe.MatchString(s) {
		return fmt.Sprintf("[\"%s\"]", s)
	}

	// 正常变量
	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)
	if re.MatchString(s) {
		return fmt.Sprintf("[\"%s\"]", s)
	}

	return s
}
