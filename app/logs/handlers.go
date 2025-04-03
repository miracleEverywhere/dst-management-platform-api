package logs

import (
	"dst-management-platform-api/utils"
	"encoding/base64"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"strings"
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

func handleLogDownloadPost(c *gin.Context) {
	defer func() {
		var cmdClean = "cd /tmp && rm -f *.log logs.tgz"
		err := utils.BashCMD(cmdClean)
		if err != nil {
			utils.Logger.Error("清理日志文件失败", "err", err)
		}
	}()

	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	var cmdPrepare = "cp ~/dmp.log /tmp && cp ~/dmpProcess.log /tmp"
	var cmdTar = "cd /tmp && tar zcvf logs.tgz dmp.log dmpProcess.log"

	if config.RoomSetting.Ground != "" {
		cmdPrepare = cmdPrepare + " && cp " + utils.MasterLogPath + " /tmp/ground.log"
		cmdTar += " ground.log"
	}

	if config.RoomSetting.Cave != "" {
		cmdPrepare = cmdPrepare + " && cp " + utils.CavesLogPath + " /tmp/cave.log"
		cmdTar += " cave.log"
	}
	fmt.Println(cmdPrepare)
	fmt.Println(cmdTar)
	err = utils.BashCMD(cmdPrepare)
	if err != nil {
		utils.Logger.Error("整理日志文件失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("tarFail", langStr), "data": nil})
		return
	}
	err = utils.BashCMD(cmdTar)
	if err != nil {
		utils.Logger.Error("打包日志压缩文件失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("tarFail", langStr), "data": nil})
		return
	}
	// 读取文件内容
	fileData, err := os.ReadFile("/tmp/logs.tgz")
	if err != nil {
		utils.Logger.Error("读取日志压缩文件失败", "err", err)
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

func handleGetLogInfoGet(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	var (
		logInfos []LogInfo
		world    string
	)

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("读取配置文件失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	if config.RoomSetting.Ground != "" {
		if config.RoomSetting.Cave != "" {
			world = "both"
		} else {
			world = "ground"
		}
	} else {
		world = "cave"
	}

	logInfos = append(logInfos, getGroundLogsInfo(langStr))
	logInfos = append(logInfos, getCaveLogsInfo(langStr))
	logInfos = append(logInfos, getChatLogsInfo(world, langStr))
	logInfos = append(logInfos, getAccessLogsInfo(langStr))
	logInfos = append(logInfos, getRuntimeLogsInfo(langStr))

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": logInfos})
}

func handleCleanLogsPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	type CleanLogsForm struct {
		LogTypes []string `json:"logTypes"`
	}
	var cleanLogsForm CleanLogsForm
	if err := c.ShouldBindJSON(&cleanLogsForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	var (
		code       int
		messagesZh []string
		messagesEn []string
	)
	for _, logType := range cleanLogsForm.LogTypes {
		switch logType {
		case "Ground":
			cmd := "rm -rf " + utils.MasterBackupLogPath + "/*"
			err = utils.BashCMD(cmd)
			if err != nil {
				utils.Logger.Error("地面日志删除失败", "err", err)
				messagesZh = append(messagesZh, "地面日志删除失败")
				messagesEn = append(messagesEn, "Clean Ground Logs Fail")
				code = 201
			}
		case "Cave":
			cmd := "rm -rf " + utils.CavesBackupLogPath + "/*"
			err = utils.BashCMD(cmd)
			if err != nil {
				utils.Logger.Error("洞穴日志删除失败", "err", err)
				messagesZh = append(messagesZh, "洞穴日志删除失败")
				messagesEn = append(messagesEn, "Clean Cave Logs Fail")
				code = 201
			}
		case "Chat":
			if config.RoomSetting.Ground != "" {
				cmd := "rm -rf " + utils.MasterBackupChatLogPath + "/*"
				err = utils.BashCMD(cmd)
				if err != nil {
					utils.Logger.Error("地面聊天日志删除失败", "err", err)
					messagesZh = append(messagesZh, "地面聊天日志删除失败")
					messagesEn = append(messagesEn, "Clean Ground Chat Logs Fail")
					code = 201
				}
			}
			if config.RoomSetting.Cave != "" {
				cmd := "rm -rf " + utils.CavesBackupChatLogPath + "/*"
				err = utils.BashCMD(cmd)
				if err != nil {
					utils.Logger.Error("洞穴聊天日志删除失败", "err", err)
					messagesZh = append(messagesZh, "洞穴聊天日志删除失败")
					messagesEn = append(messagesEn, "Clean Cave Chat Logs Fail")
					code = 201
				}
			}
		case "Access":
			err = utils.TruncAndWriteFile(utils.DMPLogPath, "")
			if err != nil {
				utils.Logger.Error("请求日志删除失败", "err", err)
				messagesZh = append(messagesZh, "请求日志删除失败")
				messagesEn = append(messagesEn, "Clean Access Logs Fail")
				code = 201
			}
		case "Runtime":
			err = utils.TruncAndWriteFile(utils.ProcessLogFile, "")
			if err != nil {
				utils.Logger.Error("运行日志删除失败", "err", err)
				messagesZh = append(messagesZh, "运行日志删除失败")
				messagesEn = append(messagesEn, "Clean Runtime Logs Fail")
				code = 201
			}
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
	}

	if code != 201 {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("cleanSuccess", langStr), "data": nil})
	} else {
		var message string
		if langStr == "zh" {
			message = strings.Join(messagesZh, "，")
		} else {
			message = strings.Join(messagesEn, ", ")
		}
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message, "data": nil})
	}
}
