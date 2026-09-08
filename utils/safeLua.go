package utils

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/gopher-lua"
)

// NewSafeLuaState 创建沙箱化 Lua 状态，只加载数据解析所需的安全模块，禁止 os/io/debug/package 等危险库
func NewSafeLuaState() *lua.LState {
	L := lua.NewState(lua.Options{
		SkipOpenLibs: true,
	})
	// 只打开数据解析所需的安全库
	for _, pair := range []struct {
		n string
		f lua.LGFunction
	}{
		{lua.BaseLibName, lua.OpenBase},
		{lua.TabLibName, lua.OpenTable},
		{lua.StringLibName, lua.OpenString},
		{lua.MathLibName, lua.OpenMath},
	} {
		if err := L.CallByParam(lua.P{
			Fn:      L.NewFunction(pair.f),
			NRet:    0,
			Protect: true,
		}, lua.LString(pair.n)); err != nil {
			panic(fmt.Sprintf("加载 Lua 安全库失败: %v", err))
		}
	}
	// 移除 base 库中的危险全局函数
	for _, name := range []string{
		"dofile", "loadfile", "load", "loadstring",
		"require", "module", "setfenv", "getfenv", "newproxy",
		"collectgarbage",
	} {
		L.SetGlobal(name, lua.LNil)
	}
	return L
}

// kleiSaveHeaderPattern 匹配 Klei 引擎写入存档时添加的头部（形如 "KLEI     1 "）。
// 该头部不是合法 Lua，会导致前端严格的 Lua 5.1 解析器报错。
var kleiSaveHeaderPattern = regexp.MustCompile(`^KLEI\s+\d+\s*`)

// NormalizeLuaConfig 归一化 DST 的 lua 配置文件内容（leveldataoverride.lua、modoverrides.lua 等），
// 仅剥离会导致前端解析失败的非法前缀，不改动其余合法内容：
//  1. UTF-8 BOM（\ufeff）；
//  2. Klei 引擎存档头（形如 "KLEI     1 "），其后才是真正的 "return { ... }"。
//
// 前端保存房间时会用严格的 Lua 5.1 解析器校验每个世界的配置，上述前缀会触发
// “世界代码配置格式错误”/“模组配置格式错误”，导致整个房间表单无法保存。
func NormalizeLuaConfig(content string) string {
	content = strings.TrimPrefix(content, "\ufeff")
	content = kleiSaveHeaderPattern.ReplaceAllString(content, "")
	return content
}
