package cache

import "sync"

var (
	// 自定义游戏启动命令
	customGameStartupCmd     string
	customGameStartupCmdLock sync.RWMutex
)

func GetCustomGameStartupCmd() string {
	customGameStartupCmdLock.RLock()
	defer customGameStartupCmdLock.RUnlock()
	return customGameStartupCmd
}

func SetCustomGameStartupCmd(cmd string) {
	customGameStartupCmdLock.Lock()
	defer customGameStartupCmdLock.Unlock()
	customGameStartupCmd = cmd
}
