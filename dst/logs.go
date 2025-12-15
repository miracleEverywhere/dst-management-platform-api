package dst

import (
	"bufio"
	"dst-management-platform-api/logger"
	"fmt"
	"os"
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

	return getLogFileLastNLines(logPath, lines)
}

func getLogFileLastNLines(filename string, n int) []string {
	file, err := os.Open(filename)
	if err != nil {
		logger.Logger.Error("读取日志失败", "path", filename, "err", err)
		return []string{}
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logger.Logger.Error("文件关闭失败", "err", err)
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
		logger.Logger.Error("读取日志失败", "path", filename, "err", err)
		return []string{}
	}

	return lines
}
