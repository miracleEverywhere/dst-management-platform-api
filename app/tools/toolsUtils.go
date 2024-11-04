package tools

import (
	"dst-management-platform-api/utils"
	"fmt"
	"os"
	"sort"
	"syscall"
	"time"
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

// FileInfo 结构体，包含文件名和创建时间
type FileInfo struct {
	Name    string    `json:"name"`
	ModTime time.Time `json:"modTime"`
}

// FileInfoList 用于排序的切片
type FileInfoList []FileInfo

func (f FileInfoList) Len() int {
	return len(f)
}

func (f FileInfoList) Less(i, j int) bool {
	// 反向排序：创建时间较新的文件排在前面
	return f[i].ModTime.After(f[j].ModTime)
}

func (f FileInfoList) Swap(i, j int) {
	f[i], f[j] = f[j], f[i]
}

func getBackupFiles() (FileInfoList, error) {
	entries, err := os.ReadDir(utils.BackupPath)
	if err != nil {
		fmt.Println("读取目录时出错:", err)
		return FileInfoList{}, err
	}
	// 创建 FileInfoList 切片
	var fileInfoList FileInfoList

	// 遍历文件并添加到 FileInfoList
	for _, entry := range entries {
		if !entry.IsDir() {
			// 获取文件信息
			info, err := entry.Info()
			if err != nil {
				fmt.Printf("获取文件 %s 信息时出错: %v\n", entry.Name(), err)
				continue
			}
			fileInfoList = append(fileInfoList, FileInfo{
				Name:    info.Name(),
				ModTime: info.ModTime(),
			})
		}
	}

	// 按照创建时间排序
	sort.Sort(fileInfoList)

	return fileInfoList, nil
}
