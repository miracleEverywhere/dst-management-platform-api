package tools

import (
	"os"
	"syscall"
)

func restartMyself() error {
	// 获取当前可执行文件的路径
	argv0, err := os.Executable()
	if err != nil {
		return err
	}

	// 创建一个新的进程，使用 syscall.Exec 直接替换当前进程
	// 注意：这里直接使用 exec 来保持 PID 不变，实现优雅重启
	return syscall.Exec(argv0, os.Args, os.Environ())
}
