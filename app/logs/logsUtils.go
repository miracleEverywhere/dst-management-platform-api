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

type LogInfo struct {
	Name string `json:"name"`
	Size int64  `json:"size"`
	Num  int    `json:"num"`
}

func getGroundLogsInfo(lang string) LogInfo {
	size, err := utils.GetDirSize(utils.MasterBackupLogPath)
	if err != nil {
		utils.Logger.Warn("计算日志大小失败", "err", err)
	}
	count, err := utils.CountFiles(utils.MasterBackupLogPath)
	if err != nil {
		utils.Logger.Warn("计算日志数量失败", "err", err)
	}

	var logInfo LogInfo
	if lang == "zh" {
		logInfo.Name = "地面日志"
	} else {
		logInfo.Name = "Ground"
	}
	logInfo.Size = size
	logInfo.Num = count

	return logInfo
}

func getCaveLogsInfo(lang string) LogInfo {
	size, err := utils.GetDirSize(utils.CavesBackupLogPath)
	if err != nil {
		utils.Logger.Warn("计算日志大小失败", "err", err)
	}
	count, err := utils.CountFiles(utils.CavesBackupLogPath)
	if err != nil {
		utils.Logger.Warn("计算日志数量失败", "err", err)
	}

	var logInfo LogInfo
	if lang == "zh" {
		logInfo.Name = "洞穴日志"
	} else {
		logInfo.Name = "Cave"
	}
	logInfo.Size = size
	logInfo.Num = count

	return logInfo
}

func getChatLogsInfo(world string, lang string) LogInfo {
	var (
		size  int64
		count int
		err   error
	)
	if world == "ground" {
		size, err = utils.GetDirSize(utils.MasterBackupChatLogPath)
		if err != nil {
			utils.Logger.Warn("计算日志大小失败", "err", err)
		}
		count, err = utils.CountFiles(utils.MasterBackupChatLogPath)
		if err != nil {
			utils.Logger.Warn("计算日志数量失败", "err", err)
		}
	}
	if world == "cave" {
		size, err = utils.GetDirSize(utils.CavesBackupChatLogPath)
		if err != nil {
			utils.Logger.Warn("计算日志大小失败", "err", err)
		}
		count, err = utils.CountFiles(utils.CavesBackupChatLogPath)
		if err != nil {
			utils.Logger.Warn("计算日志数量失败", "err", err)
		}
	}
	if world == "both" {
		sizeMaster, err := utils.GetDirSize(utils.MasterBackupChatLogPath)
		if err != nil {
			utils.Logger.Warn("计算日志大小失败", "err", err)
		}
		countMaster, err := utils.CountFiles(utils.MasterBackupChatLogPath)
		if err != nil {
			utils.Logger.Warn("计算日志数量失败", "err", err)
		}

		sizeCave, err := utils.GetDirSize(utils.CavesBackupChatLogPath)
		if err != nil {
			utils.Logger.Warn("计算日志大小失败", "err", err)
		}
		countCave, err := utils.CountFiles(utils.CavesBackupChatLogPath)
		if err != nil {
			utils.Logger.Warn("计算日志数量失败", "err", err)
		}

		size = sizeMaster + sizeCave
		count = countMaster + countCave
	}

	var logInfo LogInfo
	if lang == "zh" {
		logInfo.Name = "聊天日志"
	} else {
		logInfo.Name = "Chat"
	}
	logInfo.Size = size
	logInfo.Num = count

	return logInfo
}

func getAccessLogsInfo(lang string) LogInfo {
	size, err := utils.GetFileSize(utils.DMPLogPath)
	if err != nil {
		utils.Logger.Warn("计算日志大小失败", "err", err)
	}

	var logInfo LogInfo
	if lang == "zh" {
		logInfo.Name = "请求日志"
	} else {
		logInfo.Name = "Access"
	}
	logInfo.Size = size
	logInfo.Num = 1

	return logInfo
}

func getRuntimeLogsInfo(lang string) LogInfo {
	size, err := utils.GetFileSize(utils.ProcessLogFile)
	if err != nil {
		utils.Logger.Warn("计算日志大小失败", "err", err)
	}

	var logInfo LogInfo
	if lang == "zh" {
		logInfo.Name = "运行日志"
	} else {
		logInfo.Name = "Runtime"
	}
	logInfo.Size = size
	logInfo.Num = 1

	return logInfo
}
