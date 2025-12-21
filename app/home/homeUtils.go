package home

import (
	"dst-management-platform-api/utils"
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"strings"
)

type AllScreens struct {
	ScreenName string `json:"screenName"`
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

func GetClusterScreens(clusterName string) []AllScreens {
	cmd := fmt.Sprintf("ps -ef | grep DST_%s | grep dontstarve_dedicated_server_nullrenderer | grep -v grep | awk '{print $14}'", clusterName)
	out, _, _ := utils.BashCMDOutput(cmd)
	screenNamesStr := strings.TrimSpace(out)

	screenNames := strings.Split(screenNamesStr, "\n")

	var allScreens []AllScreens
	for _, i := range screenNames {
		if i != "" {
			allScreens = append(allScreens, AllScreens{ScreenName: i})
		}
	}

	return allScreens
}
