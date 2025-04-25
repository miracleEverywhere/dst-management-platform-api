package logs

import (
	"bufio"
	"dst-management-platform-api/utils"
	"os"
)

func getLastNLines(filename string, n int) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			utils.Logger.Error("文件关闭失败", "err", err)
		}
	}(file)

	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:] // 移除前面的行，保持最后 n 行
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func caclLogSizeCount(logPath string) (int64, int) {
	size, err := utils.GetDirSize(logPath)
	if err != nil {
		utils.Logger.Error("计算日志大小失败", "err", err, "path", logPath)
		size = 0
	}
	count, err := utils.CountFiles(logPath)
	if err != nil {
		utils.Logger.Error("计算日志数量失败", "err", err, "path", logPath)
		count = 0
	}

	return size, count
}

type LogInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Num  int    `json:"num"`
}

func getClusterLogInfo(cluster utils.Cluster, langStr string) []LogInfo {
	var err error
	var worldLogInfo, chatLogInfo, accessLogInfo, runtimeLogInfo LogInfo

	for _, world := range cluster.Worlds {
		var (
			logPath string
			size    int64
			count   int
		)
		// 世界日志
		logPath = world.GetBackupServerLogPath(cluster.ClusterSetting.ClusterName)
		size, count = caclLogSizeCount(logPath)
		worldLogInfo.Size = worldLogInfo.Size + size
		worldLogInfo.Num = worldLogInfo.Num + count
		// 聊天日志
		logPath = world.GetBackupChatLogPath(cluster.ClusterSetting.ClusterName)
		size, count = caclLogSizeCount(logPath)
		chatLogInfo.Size = chatLogInfo.Size + size
		chatLogInfo.Num = chatLogInfo.Num + count
	}

	// 请求日志
	accessLogInfo.Size, err = utils.GetFileSize(utils.DMPAccessLog)
	if err != nil {
		utils.Logger.Error("计算日志大小失败", "err", err, "path", utils.DMPAccessLog)
		accessLogInfo.Size = 0
	}

	// 运行日志
	runtimeLogInfo.Size, err = utils.GetFileSize(utils.DMPRuntimeLog)
	if err != nil {
		utils.Logger.Error("计算日志大小失败", "err", err, "path", utils.DMPRuntimeLog)
		runtimeLogInfo.Size = 0
	}

	if langStr == "zh" {
		worldLogInfo.Name = "世界日志"
		chatLogInfo.Name = "聊天日志"
		accessLogInfo.Name = "请求日志"
		runtimeLogInfo.Name = "平台日志"
	} else {
		worldLogInfo.Name = "World"
		chatLogInfo.Name = "Chat"
		accessLogInfo.Name = "Access"
		runtimeLogInfo.Name = "Runtime"
	}

	accessLogInfo.Num = 1
	runtimeLogInfo.Num = 1

	return []LogInfo{worldLogInfo, chatLogInfo, accessLogInfo, runtimeLogInfo}
}
