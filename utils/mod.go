package utils

import (
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"os"
	"regexp"
	"sort"
	"strconv"
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

// ModFormattedData 下面这两个结构体其实可以合并，但是enable和enabled很烦， 前端也要改，直接用for互相转换一下吧，累了，毁灭吧
type ModFormattedData struct {
	ID                   int                    `json:"id"`
	Name                 string                 `json:"name"`
	Enable               bool                   `json:"enable"`
	ConfigurationOptions map[string]interface{} `json:"configurationOptions"`
	PreviewUrl           string                 `json:"preview_url"`
	FileUrl              string                 `json:"file_url"`
}

// ModOverrides 下面这两个结构体其实可以合并，但是enable和enabled很烦， 前端也要改，直接用for互相转换一下吧，累了，毁灭吧
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
	if luaScript == "" {
		return []ModOverrides{}
	}
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
		return []ModOverrides{}
	}

	var modOverrides []ModOverrides

	table.ForEach(func(k lua.LValue, v lua.LValue) {
		// k是workshop-xxx, v是configuration_options和enabled
		var modOverridesItem ModOverrides
		modOverridesItem.Name = k.String()

		if modOverridesItem.Name == "client_mods_disabled" {
			modOverridesItem.ID = 1
		} else {
			re := regexp.MustCompile(`\d+`)
			modOverridesItem.ID, _ = strconv.Atoi(re.FindAllString(k.String(), -1)[0])
		}
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
							// 尝试转换整数
							parsedValue, err = strconv.Atoi(optionsValue.String())
							if err != nil {
								parsedValue, err = strconv.ParseFloat(optionsValue.String(), 64)
							}
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

func ParseToLua(data []ModFormattedData) string {
	luaString := "return {\n"
	modNum := len(data)
	modCount := 1
	var keys []string

	for _, mod := range data {
		if mod.ID == 1 {
			luaString += "  client_mods_disabled={\n    configuration_options={\n"
			luaString += "    },\n"
		} else {
			modID := "workshop-" + strconv.Itoa(mod.ID)
			luaString += "  [\"" + modID + "\"]={\n    configuration_options={\n"
			configurationOptions := mod.ConfigurationOptions
			keyNum := len(configurationOptions)
			keyCount := 1
			keys = []string{}

			// keys为configurationOptions排序切片
			for key := range configurationOptions {
				keys = append(keys, key)
			}
			// 对键切片进行排序
			sort.Strings(keys)

			for _, key := range keys {
				value := configurationOptions[key]

				var stringValue string
				switch value.(type) {
				case string:
					stringValue = "\"" + value.(string) + "\""
				case int:
					stringValue = strconv.Itoa(value.(int))
				case float64:
					stringValue = strconv.FormatFloat(value.(float64), 'f', -1, 64)
				case bool:
					stringValue = fmt.Sprintf("%t", value)
				}
				// 判断是否需要['key']这种形式
				if NeedDoubleQuotes(key) {
					luaString += "      [\"" + key + "\"]=" + stringValue
				} else {
					luaString += "      " + key + "=" + stringValue
				}
				//fmt.Println(value, "---", stringValue)
				if keyCount == keyNum {
					luaString += "\n"
				} else {
					luaString += ",\n"
				}
				keyCount++
			}
			luaString += "    },\n"
		}

		stat := mod.Enable
		luaString += "    enabled=" + Bool2String(stat, "lua") + "\n"
		if modCount == modNum {
			luaString += "  }\n"
		} else {
			luaString += "  },\n"
		}
		modCount++
	}
	luaString += "}\n"

	return luaString
}

func NeedDoubleQuotes(s string) bool {
	if len(s) == 0 {
		return true
	}

	re := regexp.MustCompile(`[^a-zA-Z0-9_]`)

	return re.MatchString(s)
}

func GenerateModDownloadCMD(id int) string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		Logger.Error("无法获取 home 目录", "err", err)
		return ""
	}
	filePath := homeDir + "/" + ModDownloadPath
	cmd := "steamcmd/steamcmd.sh +force_install_dir "
	cmd += filePath + " +login anonymous"
	cmd += " +workshop_download_item 322330 " + strconv.Itoa(id)
	cmd += " +quit"

	return cmd
}

func GetModDefaultConfigs(id int) {

}

func SyncMods() error {
	// 处理UGC模组
	cmd := "cp -r " + MasterModUgcPath + "/* " + ModDownloadPath + "/steamapps/workshop/content/322330"
	err := BashCMD(cmd)
	if err != nil {
		Logger.Error("同步UGC模组失败", "err", err)
		return err
	}
	// 处理非UGC模组
	cmd = "for dir in " + ModNoUgcPath + "/workshop-*; do [ -d \"$dir\" ] && cp -r \"$dir\" \"" + ModDownloadPath + "/not_ugc/$(basename \"$dir\" | sed 's/workshop-//')\"; done"
	if err != nil {
		Logger.Error("同步非UGC模组失败", "err", err)
		return err
	}

	return nil
}

func DeleteDownloadedMod(isUgc bool, id int) error {
	var err error
	if isUgc {
		err = RemoveDir(ModDownloadPath + "/steamapps/workshop/content/322330/" + strconv.Itoa(id))
	} else {
		err = RemoveDir(ModDownloadPath + "/not_ugc/" + strconv.Itoa(id))
	}

	return err
}

func AddModDefaultConfig(newModLuaScript string, id int, langStr string) []ModOverrides {
	var modDefaultConfig ModOverrides
	modConfig := GetModConfigOptions(newModLuaScript, langStr)
	modDefaultConfig.ID = id
	modDefaultConfig.Enabled = true
	modDefaultConfig.ConfigurationOptions = make(map[string]interface{})

	for _, option := range modConfig {
		modDefaultConfig.ConfigurationOptions[option.Name] = option.Default
	}
	config, err := ReadConfig()
	if err != nil {
		Logger.Error("配置文件读取失败", "err", err)
		return []ModOverrides{}
	}
	modOverridesLuaScript := config.RoomSetting.Mod
	modOverrides := ModOverridesToStruct(modOverridesLuaScript)
	modOverrides = append(modOverrides, modDefaultConfig)

	return modOverrides
}

func CheckModDownloadedReady(ugc bool, modID int, modSize string) (bool, error) {
	var (
		modPath string
	)

	if !ugc {
		// not ugc
		modPath = ModDownloadPath + "/not_ugc/" + strconv.Itoa(modID)
	} else {
		// ugc
		modPath = ModDownloadPath + "/steamapps/workshop/content/322330/" + strconv.Itoa(modID)
	}

	realSize, err := GetDirSize(modPath)
	if err != nil {
		return false, err
	}

	correctSize, err := strconv.Atoi(modSize)
	diffSize := int64(correctSize) - realSize

	if diffSize != 0 {
		return false, nil
	} else {
		return true, nil
	}
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
