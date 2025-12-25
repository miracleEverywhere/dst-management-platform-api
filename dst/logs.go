package dst

import (
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
)

func (g *Game) getLogContent(logType string, id, lines int) []string {
	var logPath string

	switch logType {
	case "game":
		world, err := g.getWorldByID(id)
		if err != nil {
			return []string{}
		}
		logPath = fmt.Sprintf("%s/server_log.txt", world.worldPath)
		logger.Logger.Debug(logPath)
	case "chat":
		for _, world := range g.worldSaveData {
			if g.worldUpStatus(world.ID) {
				logPath = fmt.Sprintf("%s/server_chat_log.txt", world.worldPath)
				break
			}
		}
	default:
		return []string{}
	}

	logger.Logger.Debug(logPath)
	if logPath == "" {
		return []string{}
	}

	return utils.GetFileLastNLines(logPath, lines)
}

func (g *Game) historyFileList(logType string, id int) []string {
	var logPath string

	switch logType {
	case "game":
		world, err := g.getWorldByID(id)
		if err != nil {
			return []string{}
		}
		logPath = fmt.Sprintf("%s/backup/server_log", world.worldPath)
		logger.Logger.Debug(logPath)
	case "chat":
		for _, world := range g.worldSaveData {
			if g.worldUpStatus(world.ID) {
				logPath = fmt.Sprintf("%s/backup/server_chat_log", world.worldPath)
				break
			}
		}
	default:
		return []string{}
	}

	files, err := utils.GetFiles(logPath)
	if err != nil {
		return []string{}
	}

	return files
}

func (g *Game) historyFileContent(logType, logfileName string, id int) string {
	var logPath string

	switch logType {
	case "game":
		world, err := g.getWorldByID(id)
		if err != nil {
			return ""
		}
		logPath = fmt.Sprintf("%s/backup/server_log/%s", world.worldPath, logfileName)
		logger.Logger.Debug(logPath)
	case "chat":
		for _, world := range g.worldSaveData {
			if g.worldUpStatus(world.ID) {
				logPath = fmt.Sprintf("%s/backup/server_chat_log/%s", world.worldPath, logfileName)
				break
			}
		}
	default:
		return ""
	}

	content, err := utils.GetFileAllContent(logPath)
	if err != nil {
		return ""
	}

	return content
}

type LogInfo struct {
	Game    int64 `json:"game"`
	Chat    int64 `json:"chat"`
	Steam   int64 `json:"steam"`
	Access  int64 `json:"access"`
	Runtime int64 `json:"runtime"`
}

func (g *Game) logsInfo() LogInfo {
	var logInfo LogInfo
	for _, world := range g.worldSaveData {
		size, err := utils.GetDirSize(fmt.Sprintf("%s/backup/server_log", world.worldPath))
		if err == nil {
			logInfo.Game = logInfo.Game + size
		}
		size, err = utils.GetDirSize(fmt.Sprintf("%s/backup/server_chat_log", world.worldPath))
		if err == nil {
			logInfo.Chat = logInfo.Chat + size
		}
	}
	steamSize, err := utils.GetFileSize("Steam/logs/bootstrap_log.txt")
	if err == nil {
		logInfo.Steam = logInfo.Steam + steamSize
	}
	accessSize, err := utils.GetFileSize("logs/access.log")
	if err == nil {
		logInfo.Access = logInfo.Access + accessSize
	}
	runtimeSize, err := utils.GetFileSize("logs/runtime.log")
	if err == nil {
		logInfo.Runtime = logInfo.Runtime + runtimeSize
	}

	return logInfo
}

type CleanLogs struct {
	Game    bool `json:"game"`
	Chat    bool `json:"chat"`
	Steam   bool `json:"steam"`
	Access  bool `json:"access"`
	Runtime bool `json:"runtime"`
}

func (g *Game) logsClean(cleanLogs *CleanLogs) bool {
	allSuccess := true

	if cleanLogs.Game {
		for _, world := range g.worldSaveData {
			err := utils.RemoveDir(fmt.Sprintf("%s/backup/server_log", world.worldPath))
			if err != nil {
				allSuccess = false
				logger.Logger.Error("删除游戏日志失败", "err", err)
			}
		}
	}
	if cleanLogs.Chat {
		for _, world := range g.worldSaveData {
			err := utils.RemoveDir(fmt.Sprintf("%s/backup/server_chat_log", world.worldPath))
			if err != nil {
				allSuccess = false
				logger.Logger.Error("删除聊天日志失败", "err", err)
			}
		}
	}
	if cleanLogs.Steam {
		err := utils.TruncAndWriteFile("Steam/logs/bootstrap_log.txt", "")
		if err != nil {
			allSuccess = false
			logger.Logger.Error("删除Steam日志失败", "err", err)
		}
	}
	if cleanLogs.Access {
		err := utils.TruncAndWriteFile("logs/access.log", "")
		if err != nil {
			allSuccess = false
			logger.Logger.Error("删除请求日志失败", "err", err)
		}
	}
	if cleanLogs.Runtime {
		err := utils.TruncAndWriteFile("logs/runtime.log", "")
		if err != nil {
			allSuccess = false
			logger.Logger.Error("删除运行日志失败", "err", err)
		}
	}

	return allSuccess
}

func (g *Game) logsList(admin bool) []string {
	var files []string

	for _, world := range g.worldSaveData {
		files = append(files, fmt.Sprintf("%s/server_log.txt", world.worldPath))
	}

	if admin {
		files = append(files, "logs/access.log", "logs/runtime.log")
	}

	return files
}
