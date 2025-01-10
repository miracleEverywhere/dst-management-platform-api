package tools

import (
	"dst-management-platform-api/utils"
	"os"
	"sort"
	"time"
)

// FileInfo 结构体，包含文件名和创建时间
type FileInfo struct {
	Name    string    `json:"name"`
	ModTime time.Time `json:"modTime"`
	Size    int64     `json:"size"`
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
		utils.Logger.Error("读取目录时出错", "err", err)
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
				utils.Logger.Error("获取文件信息时出错", "err", err, "file", entry.Name())
				continue
			}
			fileInfoList = append(fileInfoList, FileInfo{
				Name:    info.Name(),
				ModTime: info.ModTime(),
				Size:    info.Size(),
			})
		}
	}

	// 按照创建时间排序
	sort.Sort(fileInfoList)

	return fileInfoList, nil
}
