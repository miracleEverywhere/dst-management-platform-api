package utils

import (
	"encoding/json"
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"regexp"
	"strconv"
	"strings"
)

type Option struct {
	Description string      `json:"description"`
	Data        interface{} `json:"data"`
}

type ConfigurationOption struct {
	Name    string      `json:"name"`
	Label   string      `json:"label"`
	Hover   string      `json:"hover"`
	Options []Option    `json:"options"`
	Default interface{} `json:"default"`
}

type ModInfo struct {
}

type ModOverrides struct {
	ID                   int                    `json:"id"`
	Name                 string                 `json:"name"`
	Enabled              bool                   `json:"enabled"`
	ConfigurationOptions map[string]interface{} `json:"configurationOptions"`
}

func GetModConfigOptions(luaScript string, lang string) []ConfigurationOption {
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
	if err := L.DoString(luaScript); err != nil {
		return []ConfigurationOption{}
	}

	// 获取 configuration_options 表
	configOptions := L.GetGlobal("configuration_options")
	if configOptions.Type() != lua.LTTable {
		return []ConfigurationOption{}
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
					option.Default = value
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
									opt.Data = value
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
	return options
}

func ModOverridesToStruct(luaScript string) []ModOverrides {
	L := lua.NewState()
	defer L.Close()

	// 加载并执行 Lua 脚本
	if err := L.DoString(luaScript); err != nil {
		return []ModOverrides{}
	}
	// 获取返回值
	results := L.Get(-1)
	L.Pop(1)
	table, ok := results.(*lua.LTable)

	//fmt.Println(table.Len())
	if !ok {
		fmt.Println("Returned value is not a table")
		return []ModOverrides{}
	}

	var modOverrides []ModOverrides

	table.ForEach(func(k lua.LValue, v lua.LValue) {
		// k是workshop-xxx, v是configuration_options和enabled
		var modOverridesItem ModOverrides
		modOverridesItem.Name = k.String()
		re := regexp.MustCompile(`\d+`)
		modOverridesItem.ID, _ = strconv.Atoi(re.FindAllString(k.String(), -1)[0])
		if v.Type() == lua.LTTable {
			v.(*lua.LTable).ForEach(func(key lua.LValue, value lua.LValue) {
				// key是configuration_options和enabled
				if key.String() == "enabled" {
					var err error
					modOverridesItem.Enabled, err = StringToBool(value.String())
					if err != nil {
						Logger.Error(err.Error())
					}
				} else if key.String() == "configuration_options" {
					itemMap := make(map[string]interface{})
					value.(*lua.LTable).ForEach(func(optionsKey lua.LValue, optionsValue lua.LValue) {
						var (
							parsedValue interface{}
							err         error
						)

						switch optionsValue.Type() {
						case lua.LTBool:
							parsedValue, err = StringToBool(optionsValue.String())
						case lua.LTNumber:
							parsedValue, err = strconv.Atoi(optionsValue.String())
						default:
							parsedValue, err = optionsValue.String(), nil
						}
						if err != nil {
							Logger.Error(err.Error())
						}
						itemMap[optionsKey.String()] = parsedValue
					})
					modOverridesItem.ConfigurationOptions = itemMap

				}

			})
		}
		modOverrides = append(modOverrides, modOverridesItem)
	})

	return modOverrides
}

func StringToBool(s string) (bool, error) {
	switch s {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}

	return false, fmt.Errorf("无法转换字符串%s", s)
}

type ModConfig struct {
	ID                int             `json:"id"`
	Name              string          `json:"name"`
	Enable            bool            `json:"enable"`
	ConfigurationOpts map[string]bool `json:"configurationOptions"`
	PreviewURL        string          `json:"preview_url"`
}

type ModEntry struct {
	ID                string
	ConfigurationOpts map[string]bool
	Enabled           bool
}

func JsonToLua(jsonStr string) (string, error) {
	var mods []ModConfig
	err := json.Unmarshal([]byte(jsonStr), &mods)
	if err != nil {
		return "", fmt.Errorf("Failed to parse JSON: %v", err)
	}

	luaTableEntries := make([]string, 0, len(mods))

	for _, mod := range mods {
		if !mod.Enable {
			continue // Skip disabled mods
		}

		configOpts := make(map[string]bool)
		for k, v := range mod.ConfigurationOpts {
			configOpts[k] = v
		}

		entry := ModEntry{
			ID:                fmt.Sprintf("workshop-%d", mod.ID),
			ConfigurationOpts: configOpts,
			Enabled:           mod.Enable,
		}

		luaEntry := generateLuaEntry(entry)
		luaTableEntries = append(luaTableEntries, luaEntry)
	}

	luaCode := "return {\n" + strings.Join(luaTableEntries, ",\n") + "\n}"

	return luaCode, nil
}

func generateLuaEntry(entry ModEntry) string {
	// Generate configuration_options table
	var configOptsParts []string
	for k, v := range entry.ConfigurationOpts {
		configOptsParts = append(configOptsParts, fmt.Sprintf("%q = %t", k, v))
	}
	configOptsStr := "{ " + strings.Join(configOptsParts, ", ") + " }"

	return fmt.Sprintf("[\"%s\"] = { configuration_options = %s, enabled = %t }",
		entry.ID, configOptsStr, entry.Enabled)
}

// 计算 Lua 表的元素个数（包括数组部分和哈希部分）
//func getTableLength(table *lua.LTable) int {
//	// 计算数组部分的元素个数
//	arrayLen := table.Len()
//
//	// 计算哈希部分的元素个数
//	var hashLen int
//	table.ForEach(func(key lua.LValue, value lua.LValue) {
//		hashLen++
//	})
//
//	// 返回总元素个数
//	return arrayLen + hashLen
//}
