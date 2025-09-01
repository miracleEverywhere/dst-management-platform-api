package tools

import (
	"bufio"
	"dst-management-platform-api/utils"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
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

func getBackupFiles(cluster utils.Cluster) (FileInfoList, error) {
	backupPath := cluster.GetBackupPath()
	entries, err := os.ReadDir(backupPath)
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

func getCoordinate(cmd, screenName, logPath string) (int, int, error) {
	err := utils.ScreenCMD(cmd, screenName)
	if err != nil {
		return 0, 0, err
	}

	time.Sleep(100 * time.Millisecond)

	// 打开文件
	file, err := os.Open(logPath)
	if err != nil {
		return 0, 0, err
	}
	defer file.Close()

	// 使用缓冲读取器
	scanner := bufio.NewScanner(file)
	var lines []string
	var targetLineIndex int = -1

	// 先扫描文件并将所有行存入内存（适用于可以放入内存的文件）
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)
		if strings.Contains(line, cmd) {
			// 记录最后一个匹配行的索引
			targetLineIndex = len(lines) - 1
		}
	}

	if targetLineIndex == -1 {
		return 0, 0, fmt.Errorf("未找到坐标信息")
	}

	// 检查是否有足够的后续行
	if targetLineIndex+3 >= len(lines) {
		return 0, 0, fmt.Errorf("找到目标行但没有足够的后续行")
	}

	// 提取坐标的三行
	coordLines := lines[targetLineIndex+1 : targetLineIndex+4]
	var x, y int
	var parseErr error

	// 解析第三行坐标
	nums := strings.Fields(coordLines[2])
	utils.Logger.Info(strconv.Itoa(len(nums)))
	utils.Logger.Info(nums[1])
	utils.Logger.Info(nums[3])
	if len(nums) >= 4 {
		x, parseErr = strconv.Atoi(nums[1])
		if parseErr != nil {
			return 0, 0, fmt.Errorf("解析x坐标失败")
		}
		y, parseErr = strconv.Atoi(nums[3])
		if parseErr != nil {
			return 0, 0, fmt.Errorf("解析y坐标失败")
		}
	}

	return x, y, nil
}
