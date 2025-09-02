package tools

import (
	"bufio"
	"dst-management-platform-api/utils"
	"fmt"
	"os"
	"regexp"
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
	if len(nums) >= 4 {
		if strings.Contains(nums[1], ".") {
			a, err := strconv.ParseFloat(nums[1], 64)
			if err != nil {
				return 0, 0, fmt.Errorf("字符串转浮点数失败")
			}
			x = int(a)
		} else {
			x, parseErr = strconv.Atoi(nums[1])
			if parseErr != nil {
				return 0, 0, fmt.Errorf("解析x坐标失败")
			}
		}

		if strings.Contains(nums[3], ".") {
			a, err := strconv.ParseFloat(nums[3], 64)
			if err != nil {
				return 0, 0, fmt.Errorf("字符串转浮点数失败")
			}
			y = int(a)
		} else {
			y, parseErr = strconv.Atoi(nums[3])
			if parseErr != nil {
				return 0, 0, fmt.Errorf("解析y坐标失败")
			}
		}
	}

	return x, y, nil
}

type item struct {
	CnName string `json:"cnName"`
	EnName string `json:"enName"`
	Code   string `json:"code"`
	Count  int    `json:"count"`
}

func countPrefabs(screenName, logPath string) []item {
	prefabs := []item{
		{
			CnName: "海象营地",
			EnName: "walrus camp",
			Code:   "walrus_camp",
		},
		{
			CnName: "杀人蜂巢",
			EnName: "wasp hive",
			Code:   "wasphive",
		},
		{
			CnName: "远古雕像",
			EnName: "ruins statue mage",
			Code:   "ruins_statue_mage",
		},
		{
			CnName: "远古月亮雕像",
			EnName: "archive moon statue",
			Code:   "archive_moon_statue",
		},
	}

	cmd1 := "print('=== world prefabs counting start ===')"
	err := utils.ScreenCMD(cmd1, screenName)
	if err != nil {
		utils.Logger.Error("统计世界失败", "err", err)
		return prefabs
	}

	for _, prefab := range prefabs {
		cmd := fmt.Sprintf("c_countprefabs('%s')", prefab.Code)
		_ = utils.ScreenCMD(cmd, screenName)
		time.Sleep(50 * time.Millisecond)
	}

	cmd2 := "print('=== world prefabs counting finish ===')"
	err = utils.ScreenCMD(cmd2, screenName)
	if err != nil {
		utils.Logger.Error("统计世界失败", "err", err)
		return prefabs
	}
	time.Sleep(100 * time.Millisecond)

	file, err := os.Open(logPath)
	if err != nil {
		utils.Logger.Error("统计世界失败", "err", err)
		return prefabs
	}

	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			utils.Logger.Error("文件关闭失败", "err", err)
		}
	}(file)

	// 逐行读取文件
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	var usefulLines []string

	var foundFinish bool
	var foundStart bool

	// 反向遍历行
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		if strings.Contains(line, cmd2) {
			foundFinish = true
			continue
		}

		if foundFinish {
			usefulLines = append(usefulLines, line)
		}

		// 检查是否包含关键字
		if strings.Contains(line, cmd1) {
			foundStart = true
			break
		}
	}

	if !foundStart {
		utils.Logger.Error("没有发现开始标记")
		return prefabs
	}

	// 正则表达式匹配模式
	pattern := `There are\s+(\d+)\s+(\w+)\s+in the world`
	re := regexp.MustCompile(pattern)

	// 查找匹配的行并提取所需字段
	for _, line := range usefulLines {
		if matches := re.FindStringSubmatch(line); matches != nil {
			for index, prefab := range prefabs {
				if prefab.Code+"s" == matches[2] {
					count, err := strconv.Atoi(matches[1])
					if err != nil {
						count = 0
					}
					prefabs[index].Count = count
				}
			}
		}
	}

	return prefabs
}

type Player struct {
	Uid        string `json:"uid"`
	NickName   string `json:"nickName"`
	Prefab     string `json:"prefab"`
	Coordinate struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"coordinate"`
}

func getPlayerPosition(screenName, logPath string, cluster utils.Cluster) []Player {

	var Players []Player

	utils.STATISTICSMutex.Lock()
	if len(utils.STATISTICS[cluster.ClusterSetting.ClusterName]) > 0 {
		players := utils.STATISTICS[cluster.ClusterSetting.ClusterName][len(utils.STATISTICS[cluster.ClusterSetting.ClusterName])-1].Players
		for _, player := range players {
			Players = append(Players, Player{
				Uid:      player.UID,
				NickName: player.NickName,
				Prefab:   player.Prefab,
			})
		}
	}
	utils.STATISTICSMutex.Unlock()

	if len(Players) == 0 {
		return []Player{}
	}

	for index, player := range Players {
		ts := time.Now().UnixNano()

		cmd := fmt.Sprintf("print('==== DMP Start %s [%d] Start DMP ====')", player.Uid, ts)
		err := utils.ScreenCMD(cmd, screenName)
		if err != nil {
			utils.Logger.Error("执行获取玩家坐标失败", "err", err)
			continue
		}

		time.Sleep(50 * time.Millisecond)

		cmd = fmt.Sprintf("print(UserToPlayer('%s').Transform:GetWorldPosition())", player.Uid)
		err = utils.ScreenCMD(cmd, screenName)
		if err != nil {
			utils.Logger.Error("执行获取玩家坐标失败", "err", err)
			continue
		}

		time.Sleep(50 * time.Millisecond)

		cmd = fmt.Sprintf("print('==== DMP End %s [%d] End DMP ====')", player.Uid, ts)
		err = utils.ScreenCMD(cmd, screenName)
		if err != nil {
			utils.Logger.Error("执行获取玩家坐标失败", "err", err)
			continue
		}

		time.Sleep(50 * time.Millisecond)

		data, err := utils.GetFileLastNLines(logPath, 100)
		var lines []string
		for i := len(data) - 1; i >= 0; i-- {
			lines = append(lines, data[i])
		}

		pattern := `(-?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?)\s+([-+]?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?)\s+(-?(?:\d+\.?\d*|\.\d+)(?:[eE][-+]?\d+)?)`
		re := regexp.MustCompile(pattern)

		var endFound bool

		for _, line := range lines {
			if strings.Contains(line, fmt.Sprintf("==== DMP End %s [%d] End DMP ====", player.Uid, ts)) {
				endFound = true
				continue
			}
			if endFound {
				endFound = false
				if matches := re.FindStringSubmatch(line); matches != nil {
					x, err := strconv.ParseFloat(matches[1], 64)
					if err != nil {
						break
					}
					y, err := strconv.ParseFloat(matches[3], 64)
					if err != nil {
						break
					}
					Players[index].Coordinate.X = int(x)
					Players[index].Coordinate.Y = int(y)
				}
			}

		}

	}

	var returnData []Player

	for _, player := range Players {
		if player.Coordinate.Y != 0 {
			returnData = append(returnData, player)
		}
	}

	return returnData
}
