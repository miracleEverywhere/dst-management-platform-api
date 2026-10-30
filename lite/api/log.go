package api

import (
	"dst-management-platform-api/lite/utils"
	"net/http"
	"os"
)

const (
	// logTailLines 日志接口固定返回的最后行数。
	// 接口只接收 logType 和 worldName，行数由服务端固定，读盘量和内存占用因此可预期。
	logTailLines = 100

	logTypeGame = "game"
	logTypeChat = "chat"
)

// logGet 获取当前日志：game 对应 server_log.txt，chat 对应 server_chat_log.txt。
// 聊天日志由主世界写入，主世界文件不存在或为空时回退到洞穴。
func logGet(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	logType := query.Get("logType")
	worldName := query.Get("worldName")

	if logType != logTypeGame && logType != logTypeChat {
		writeJSON(w, 400, msgBadRequest, nil)
		return
	}

	// 只允许读取数据库中配置过的世界，避免 worldName 被拼成任意路径
	utils.DBMu.RLock()
	groundName := utils.DB.Ground.Name
	caveName := utils.DB.Cave.Name
	utils.DBMu.RUnlock()

	if worldName != groundName && worldName != caveName {
		writeJSON(w, 400, msgBadRequest, nil)
		return
	}

	var logPath string
	switch logType {
	case logTypeGame:
		logPath = utils.WorldLogPath(worldName)
	case logTypeChat:
		logPath = utils.WorldChatLogPath(groundName)
		if !logFileHasContent(logPath) {
			logPath = utils.WorldChatLogPath(caveName)
		}
	}

	if logPath == "" {
		writeJSON(w, 400, msgBadRequest, nil)
		return
	}

	lines, err := utils.ReadLastLines(logPath, logTailLines)
	if err != nil {
		if !os.IsNotExist(err) {
			utils.Logger.Error("读取日志失败: " + err.Error())
		}
		lines = []string{}
	}

	writeJSON(w, 200, msgSuccess, lines)
}

// logFileHasContent 判断日志文件是否存在且非空。
func logFileHasContent(filePath string) bool {
	if filePath == "" {
		return false
	}

	info, err := os.Stat(filePath)

	return err == nil && info.Size() > 0
}
