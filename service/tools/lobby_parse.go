package tools

import (
	"context"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"strconv"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

const (
	// lobbyMaxLuaScriptLen 大厅返回的 Lua 脚本最大长度，超出视为异常数据
	lobbyMaxLuaScriptLen = 512 * 1024
	// lobbyLuaParseTimeout 单个 Lua 脚本解析超时
	lobbyLuaParseTimeout = 2 * time.Second
)

// runLobbyLua 在沙箱化的 Lua 状态中执行大厅返回的脚本，并把脚本最后一个值（表）交给 handle。
// Klei 大厅的 data / players 字段都是 Lua 表脚本，使用 utils.NewSafeLuaState 禁止 os/io 等危险库，
// 同时限制脚本长度与执行时间，避免异常数据拖垮进程。
func runLobbyLua(script string, handle func(table *lua.LTable)) (ok bool) {
	defer func() {
		if recovered := recover(); recovered != nil {
			logger.Logger.Warnf("解析大厅 Lua 数据异常: %v", recovered)
			ok = false
		}
	}()

	L := utils.NewSafeLuaState()
	defer L.Close()

	ctx, cancel := context.WithTimeout(context.Background(), lobbyLuaParseTimeout)
	defer cancel()
	L.SetContext(ctx)

	if err := L.DoString(script); err != nil {
		logger.Logger.Debugf("解析大厅 Lua 数据失败, err: %v", err)
		return false
	}

	table, isTable := L.Get(-1).(*lua.LTable)
	if !isTable {
		logger.Logger.Debugf("大厅 Lua 数据不是表结构")
		return false
	}

	handle(table)

	return true
}

// parseLobbyDays 解析大厅 data 字段中的天数信息
func parseLobbyDays(script string) (days, elapsed, left int) {
	if strings.TrimSpace(script) == "" || len(script) > lobbyMaxLuaScriptLen {
		return 0, 0, 0
	}

	runLobbyLua(script, func(table *lua.LTable) {
		days = luaInt(table.RawGetString("day"))
		elapsed = luaInt(table.RawGetString("dayselapsedinseason"))
		left = luaInt(table.RawGetString("daysleftinseason"))
	})

	return days, elapsed, left
}

// parseLobbyPlayers 解析大厅 players 字段中的在线玩家；脚本为空表示当前无玩家
func parseLobbyPlayers(script string) ([]LobbyPlayer, bool) {
	players := make([]LobbyPlayer, 0)
	if strings.TrimSpace(script) == "" {
		return players, true
	}
	if len(script) > lobbyMaxLuaScriptLen {
		return players, false
	}

	ok := runLobbyLua(script, func(table *lua.LTable) {
		table.ForEach(func(_, value lua.LValue) {
			player, isTable := value.(*lua.LTable)
			if !isTable {
				return
			}
			players = append(players, LobbyPlayer{
				Name:   luaString(player.RawGetString("name")),
				Prefab: luaString(player.RawGetString("prefab")),
				Colour: luaString(player.RawGetString("colour")),
			})
		})
	})

	return players, ok
}

// parseLobbyMods 解析大厅 mods_info 字段。
// 目前 Klei 以扁平数组上报，每个模组占 5 项：workshop-<id>、名称、版本1、版本2、是否启用；
// 同时对可能的对象结构做兼容，解析失败时退化为扫描 workshop-<id>。
func parseLobbyMods(raw []any) ([]LobbyMod, []int) {
	mods := make([]LobbyMod, 0)
	modIDs := make([]int, 0)
	seen := make(map[string]struct{}, len(raw))

	appendMod := func(mod LobbyMod) {
		if mod.ID == "" {
			return
		}
		if _, ok := seen[mod.ID]; ok {
			return
		}
		seen[mod.ID] = struct{}{}
		mods = append(mods, mod)

		if id, err := strconv.Atoi(mod.ID); err == nil && id > 0 {
			modIDs = append(modIDs, id)
		}
	}

	for index := 0; index < len(raw); index++ {
		switch value := raw[index].(type) {
		case string:
			if !strings.HasPrefix(value, "workshop-") {
				continue
			}

			mod := LobbyMod{
				ID: strings.TrimPrefix(value, "workshop-"),
			}
			if index+1 < len(raw) {
				mod.Name = anyToString(raw[index+1])
			}
			if index+2 < len(raw) {
				mod.Version1 = anyToString(raw[index+2])
			}
			if index+3 < len(raw) {
				mod.Version2 = anyToString(raw[index+3])
			}
			if index+4 < len(raw) {
				if enabled, ok := raw[index+4].(bool); ok {
					mod.Enabled = enabled
				}
			}

			appendMod(mod)
			index += 4
		case map[string]any:
			mod := LobbyMod{
				ID:       strings.TrimPrefix(anyToString(value["id"]), "workshop-"),
				Name:     anyToString(value["name"]),
				Version1: anyToString(value["version1"]),
				Version2: anyToString(value["version2"]),
			}
			if enabled, ok := value["enabled"].(bool); ok {
				mod.Enabled = enabled
			}
			appendMod(mod)
		}
	}

	return mods, modIDs
}

// luaInt 读取 Lua 值中的整数，类型不符时返回 0
func luaInt(value lua.LValue) int {
	switch typed := value.(type) {
	case lua.LNumber:
		return int(typed)
	case lua.LString:
		if number, err := strconv.Atoi(strings.TrimSpace(string(typed))); err == nil {
			return number
		}
	}

	return 0
}

// luaString 读取 Lua 值中的字符串
func luaString(value lua.LValue) string {
	switch typed := value.(type) {
	case lua.LString:
		return string(typed)
	case lua.LNumber:
		return typed.String()
	}

	return ""
}

// anyToString 将 JSON 解码出的任意值转成字符串
func anyToString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(typed)
	}

	return ""
}
