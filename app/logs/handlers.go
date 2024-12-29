package logs

import (
	"dst-management-platform-api/utils"
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
)

func handleLogGet(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	type LogForm struct {
		Line int    `form:"line" json:"line"`
		Type string `form:"type" json:"type"`
	}
	var logForm LogForm
	if err := c.ShouldBindQuery(&logForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	switch logForm.Type {
	case "ground":
		logsValue, err := getLastNLines(utils.MasterLogPath, logForm.Line)
		if err != nil {
			utils.Logger.Error("读取日志失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": []string{""}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": logsValue})
	case "caves":
		logsValue, err := getLastNLines(utils.CavesLogPath, logForm.Line)
		if err != nil {
			utils.Logger.Error("读取日志失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": []string{""}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": logsValue})
	case "chat":
		config, err := utils.ReadConfig()
		if err != nil {
			utils.Logger.Error("配置文件读取失败", "err", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}
		var logsValue []string
		if config.RoomSetting.Ground != "" {
			logsValue, err = getLastNLines(utils.MasterChatLogPath, logForm.Line)
		} else {
			logsValue, err = getLastNLines(utils.CavesChatLogPath, logForm.Line)
		}

		if err != nil {
			utils.Logger.Error("读取日志失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": []string{""}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": logsValue})
	case "dmp":
		logsValue, err := getLastNLines(utils.DMPLogPath, logForm.Line)
		if err != nil {
			utils.Logger.Error("读取日志失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": []string{""}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": logsValue})
	case "runtime":
		logsValue, err := getLastNLines(utils.ProcessLogFile, logForm.Line)
		if err != nil {
			utils.Logger.Error("读取日志失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": []string{""}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": logsValue})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
	}
}

func handleProcessLogPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	err := utils.BashCMD("tar zcvf logs.tgz dmpProcess.log")
	if err != nil {
		utils.Logger.Error("打包日志压缩文件失败")
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("tarFail", langStr), "data": nil})
		return
	}
	// 读取文件内容
	fileData, err := os.ReadFile("./logs.tgz")
	if err != nil {
		utils.Logger.Error("读取备份文件失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("fileReadFail", langStr), "data": nil})
		return
	}

	defer func() {
		err := utils.BashCMD("rm -f logs.tgz")
		if err != nil {
			utils.Logger.Error("日志压缩文件删除失败")
		}
	}()

	fileContentBase64 := base64.StdEncoding.EncodeToString(fileData)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": fileContentBase64})
}

func handleHistoricalLogFileGet(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	type LogForm struct {
		Type string `form:"type" json:"type"`
	}
	type LogFileData struct {
		Label string `json:"label"`
		Value string `json:"value"`
	}
	var logForm LogForm
	if err := c.ShouldBindQuery(&logForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	switch logForm.Type {
	case "chat":
		var (
			logFiles []string
			logPath  string
		)
		if config.RoomSetting.Ground != "" {
			logPath = utils.MasterBackupChatLogPath
		} else {
			logPath = utils.CavesBackupChatLogPath
		}
		logFiles, err = utils.GetFiles(logPath)
		if err != nil {
			utils.RespondWithError(c, 500, langStr)
			return
		}

		var data []LogFileData

		for _, i := range logFiles {
			var logFileData LogFileData
			logFileData.Label = i
			logFileData.Value = logPath + "/" + i
			data = append(data, logFileData)
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
	case "ground":
		logFiles, err := utils.GetFiles(utils.MasterBackupLogPath)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": []LogFileData{}})
			return
		}

		var data []LogFileData

		for _, i := range logFiles {
			var logFileData LogFileData
			logFileData.Label = i
			logFileData.Value = utils.MasterBackupLogPath + "/" + i
			data = append(data, logFileData)
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
	case "caves":
		logFiles, err := utils.GetFiles(utils.CavesBackupLogPath)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": []LogFileData{}})
			return
		}

		var data []LogFileData

		for _, i := range logFiles {
			var logFileData LogFileData
			logFileData.Label = i
			logFileData.Value = utils.CavesBackupLogPath + "/" + i
			data = append(data, logFileData)
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
	}
}

func handleHistoricalLogGet(c *gin.Context) {
	type LogForm struct {
		File string `form:"file" json:"file"`
	}
	var logForm LogForm
	if err := c.ShouldBindQuery(&logForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	data, err := utils.GetFileAllContent(logForm.File)
	if err != nil {
		if err != nil {
			utils.Logger.Error("读取日志失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": ""})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}
