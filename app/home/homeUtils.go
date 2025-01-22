package home

import (
	"dst-management-platform-api/utils"
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type seasonLength struct {
	Summer int `json:"summer"`
	Autumn int `json:"autumn"`
	Spring int `json:"spring"`
	Winter int `json:"winter"`
}

type SeasonI18N struct {
	En string `json:"en"`
	Zh string `json:"zh"`
}

type metaInfo struct {
	Cycles       int          `json:"cycles"`
	Phase        SeasonI18N   `json:"phase"`
	Season       SeasonI18N   `json:"season"`
	ElapsedDays  int          `json:"elapsedDays"`
	SeasonLength seasonLength `json:"seasonLength"`
}

func FindLatestMetaFile(directory string) (string, error) {
	// 检查指定目录是否存在
	_, err := os.Stat(directory)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("目录不存在：%s", directory)
	}

	// 获取指定目录下的所有子目录
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", fmt.Errorf("读取目录失败：%s", err)
	}

	// 用于存储最新的.meta文件路径和其修改时间
	var latestMetaFile string
	var latestMetaFileTime time.Time

	for _, entry := range entries {
		// 检查是否是目录
		if entry.IsDir() {
			subDirPath := filepath.Join(directory, entry.Name())

			// 获取子目录下的所有文件
			files, err := os.ReadDir(subDirPath)
			if err != nil {
				return "", fmt.Errorf("读取子目录失败：%s", err)
			}

			for _, file := range files {
				// 检查文件是否是.meta文件
				if !file.IsDir() && filepath.Ext(file.Name()) == ".meta" {
					// 获取文件的完整路径
					fullPath := filepath.Join(subDirPath, file.Name())

					// 获取文件的修改时间
					info, err := file.Info()
					if err != nil {
						return "", fmt.Errorf("获取文件信息失败：%s", err)
					}
					modifiedTime := info.ModTime()

					// 如果找到的文件的修改时间比当前最新的.meta文件的修改时间更晚，则更新最新的.meta文件路径和修改时间
					if modifiedTime.After(latestMetaFileTime) {
						latestMetaFile = fullPath
						latestMetaFileTime = modifiedTime
					}
				}
			}
		}
	}

	if latestMetaFile == "" {
		return "", fmt.Errorf("未找到.meta文件")
	}

	return latestMetaFile, nil
}

func getMetaInfo(path string) (metaInfo, error) {
	var seasonInfo metaInfo
	seasonInfo.Season.En = "Failed to retrieve"
	seasonInfo.Season.Zh = "获取失败"

	seasonInfo.Cycles = -1
	seasonInfo.Phase.En = "Failed to retrieve"
	seasonInfo.Phase.Zh = "获取失败"

	// 读取二进制文件
	data, err := os.ReadFile(path)
	if err != nil {
		return seasonInfo, fmt.Errorf("读取文件失败: %w", err)
	}

	// 创建 Lua 虚拟机
	L := lua.NewState()
	defer L.Close()

	// 将文件内容作为 Lua 代码执行
	content := string(data)
	content = content[:len(content)-1]

	err = L.DoString(content)
	if err != nil {
		return seasonInfo, fmt.Errorf("执行 Lua 代码失败: %w", err)
	}
	// 获取 Lua 脚本的返回值
	lv := L.Get(-1)
	if tbl, ok := lv.(*lua.LTable); ok {
		// 获取 clock 表
		clockTable := tbl.RawGet(lua.LString("clock"))
		if clock, ok := clockTable.(*lua.LTable); ok {
			// 获取 cycles 字段
			cycles := clock.RawGet(lua.LString("cycles"))
			if cyclesValue, ok := cycles.(lua.LNumber); ok {
				seasonInfo.Cycles = int(cyclesValue)
			}
			// 获取 phase 字段
			phase := clock.RawGet(lua.LString("phase"))
			if phaseValue, ok := phase.(lua.LString); ok {
				seasonInfo.Phase.En = string(phaseValue)
			}
		}
		// 获取 seasons 表
		seasonsTable := tbl.RawGet(lua.LString("seasons"))
		if seasons, ok := seasonsTable.(*lua.LTable); ok {
			// 获取 season 字段
			season := seasons.RawGet(lua.LString("season"))
			if seasonValue, ok := season.(lua.LString); ok {
				seasonInfo.Season.En = string(seasonValue)
			}
			// 获取 elapseddaysinseason 字段
			elapsedDays := seasons.RawGet(lua.LString("elapseddaysinseason"))
			if elapsedDaysValue, ok := elapsedDays.(lua.LNumber); ok {
				seasonInfo.ElapsedDays = int(elapsedDaysValue)
			}
			//获取季节长度
			lengthsTable := seasons.RawGet(lua.LString("lengths"))
			if lengths, ok := lengthsTable.(*lua.LTable); ok {
				summer := lengths.RawGet(lua.LString("summer"))
				if summerValue, ok := summer.(lua.LNumber); ok {
					seasonInfo.SeasonLength.Summer = int(summerValue)
				}
				autumn := lengths.RawGet(lua.LString("autumn"))
				if autumnValue, ok := autumn.(lua.LNumber); ok {
					seasonInfo.SeasonLength.Autumn = int(autumnValue)
				}
				spring := lengths.RawGet(lua.LString("spring"))
				if springValue, ok := spring.(lua.LNumber); ok {
					seasonInfo.SeasonLength.Spring = int(springValue)
				}
				winter := lengths.RawGet(lua.LString("winter"))
				if winterValue, ok := winter.(lua.LNumber); ok {
					seasonInfo.SeasonLength.Winter = int(winterValue)
				}

			}
		}
	}

	if seasonInfo.Phase.En == "night" {
		seasonInfo.Phase.Zh = "夜晚"
	}
	if seasonInfo.Phase.En == "day" {
		seasonInfo.Phase.Zh = "白天"
	}
	if seasonInfo.Phase.En == "dusk" {
		seasonInfo.Phase.Zh = "黄昏"
	}

	if seasonInfo.Season.En == "summer" {
		seasonInfo.Season.Zh = "夏天"
	}
	if seasonInfo.Season.En == "autumn" {
		seasonInfo.Season.Zh = "秋天"
	}
	if seasonInfo.Season.En == "spring" {
		seasonInfo.Season.Zh = "春天"
	}
	if seasonInfo.Season.En == "winter" {
		seasonInfo.Season.Zh = "冬天"
	}

	return seasonInfo, nil
}

func GetProcessStatus(screenName string) int {
	cmd := "ps -ef | grep " + screenName + " | grep -v grep"
	err := utils.BashCMD(cmd)
	if err != nil {
		return 0
	} else {
		return 1
	}
}

func countMods(luaScript string) (int, error) {
	L := lua.NewState()
	defer L.Close()
	if err := L.DoString(luaScript); err != nil {
		return 0, fmt.Errorf("加载 Lua 文件失败: %w", err)
	}
	modsTable := L.Get(-1)
	count := 0
	if tbl, ok := modsTable.(*lua.LTable); ok {
		tbl.ForEach(func(key lua.LValue, value lua.LValue) {
			// 检查键是否是字符串，并且以 "workshop-" 开头
			if strKey, ok := key.(lua.LString); ok && strings.HasPrefix(string(strKey), "workshop-") {
				// 提取 "workshop-" 后面的数字
				count++
			}
		})
	}
	return count, nil
}
