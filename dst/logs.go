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
