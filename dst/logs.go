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
