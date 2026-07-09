package utils

import (
	"fmt"

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
