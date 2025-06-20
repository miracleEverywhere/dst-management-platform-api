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
	type LogForm struct {
		ClusterName string `form:"clusterName" json:"clusterName"`
		WorldName   string `form:"worldName" json:"worldName"`
		Line        int    `form:"line" json:"line"`
		Type        string `form:"type" json:"type"`
	}
	var (
		logForm LogForm
		logPath string
	)
	if err := c.ShouldBindQuery(&logForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch logForm.Type {
	case "world":
		if logForm.ClusterName == "" || logForm.WorldName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少集群名或世界名"})
			return
		}
		logPath = fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s/server_log.txt", utils.HomeDir, logForm.ClusterName, logForm.WorldName)
	case "chat":
		if logForm.ClusterName == "" || logForm.WorldName == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "缺少集群名或世界名"})
			return
		}
		logPath = fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s/server_chat_log.txt", utils.HomeDir, logForm.ClusterName, logForm.WorldName)
	case "access":
		logPath = utils.DMPAccessLog
	case "runtime":
		logPath = utils.DMPRuntimeLog
	case "steam":
		logPath = utils.SteamLog
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	logsValue, err := utils.GetFileLastNLines(logPath, logForm.Line)
	if err != nil {
		utils.Logger.Error("读取日志失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": []string{""}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": logsValue})
}

func handleHistoricalLogFileGet(c *gin.Context) {
	type LogForm struct {
		ClusterName string `form:"clusterName" json:"clusterName"`
		WorldName   string `form:"worldName" json:"worldName"`
		Type        string `form:"type" json:"type"`
	}
	type LogFileData struct {
		Label string `json:"label"`
		Value string `json:"value"`
	}

	var (
		logForm  LogForm
		logPath  string
		logFiles []string
	)

	if err := c.ShouldBindQuery(&logForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if logForm.ClusterName == "" || logForm.WorldName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "缺少集群名或世界名"})
		return
	}

	switch logForm.Type {
	case "world":
		logPath = fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s/backup/server_log", utils.HomeDir, logForm.ClusterName, logForm.WorldName)
	case "chat":
		logPath = fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s/backup/server_chat_log", utils.HomeDir, logForm.ClusterName, logForm.WorldName)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	logFiles, err := utils.GetFiles(logPath)
	if err != nil {
		utils.RespondWithError(c, 500, "zh")
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

	type ReqForm struct {
		ClusterName string `json:"clusterName" form:"clusterName"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("读取配置文件失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	cluster, err := config.GetClusterWithName(reqForm.ClusterName)
	if err != nil {
		utils.Logger.Error("获取集群失败", "err", err)
		utils.RespondWithError(c, 404, langStr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": getClusterLogInfo(cluster, langStr)})
}

func handleCleanLogsPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	type CleanLogsForm struct {
		ClusterName string   `json:"clusterName"`
		LogTypes    []string `json:"logTypes"`
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

	cluster, err := config.GetClusterWithName(cleanLogsForm.ClusterName)
	if err != nil {
		utils.Logger.Error("获取集群失败", "err", err)
		utils.RespondWithError(c, 404, langStr)
		return
	}

	var (
		code       int
		messagesZh []string
		messagesEn []string
	)
	for _, logType := range cleanLogsForm.LogTypes {
		switch logType {
		case "World":
			for _, world := range cluster.Worlds {
				logPath := world.GetBackupServerLogPath(cluster.ClusterSetting.ClusterName)
				cmd := fmt.Sprintf("rm -f %s/*", logPath)
				err = utils.BashCMD(cmd)
				if err != nil {
					utils.Logger.Error("世界日志删除失败", "err", err)
					messagesZh = append(messagesZh, "世界日志删除失败")
					messagesEn = append(messagesEn, "Clean World Logs Fail")
					code = 201
				}
			}
		case "Chat":
			for _, world := range cluster.Worlds {
				logPath := world.GetBackupChatLogPath(cluster.ClusterSetting.ClusterName)
				cmd := fmt.Sprintf("rm -f %s/*", logPath)
				err = utils.BashCMD(cmd)
				if err != nil {
					utils.Logger.Error("聊天日志删除失败", "err", err)
					messagesZh = append(messagesZh, "聊天日志删除失败")
					messagesEn = append(messagesEn, "Clean Chat Logs Fail")
					code = 201
				}
			}
		case "Access":
			err = utils.TruncAndWriteFile(utils.DMPAccessLog, "")
			if err != nil {
				utils.Logger.Error("请求日志删除失败", "err", err)
				messagesZh = append(messagesZh, "请求日志删除失败")
				messagesEn = append(messagesEn, "Clean Access Logs Fail")
				code = 201
			}
		case "Runtime":
			err = utils.TruncAndWriteFile(utils.DMPRuntimeLog, "")
			if err != nil {
				utils.Logger.Error("平台日志删除失败", "err", err)
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

func handleLogDownloadPost(c *gin.Context) {
	defer func() {
		var cmdClean = "cd /tmp/dmp_tmp_logs && rm -f *"
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

	type ReqForm struct {
		ClusterName string `json:"clusterName"`
	}
	var reqForm ReqForm

	if err := c.ShouldBindJSON(&reqForm); err != nil {
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

	cluster, err := config.GetClusterWithName(reqForm.ClusterName)
	if err != nil {
		utils.Logger.Error("获取集群失败", "err", err)
		utils.RespondWithError(c, 404, langStr)
		return
	}

	var (
		cmds    []string
		logPath string
	)
	// 创建临时压缩目录
	cmds = append(cmds, "mkdir -p /tmp/dmp_tmp_logs")
	// 添加dmp日志
	cmds = append(cmds, fmt.Sprintf("cp %s %s %s /tmp/dmp_tmp_logs", utils.DMPAccessLog, utils.DMPRuntimeLog, utils.SteamLog))
	// 添加dst日志
	for _, world := range cluster.Worlds {
		logPath = world.GetServerLogFile(cluster.ClusterSetting.ClusterName)
		cmds = append(cmds, fmt.Sprintf("cp %s /tmp/dmp_tmp_logs/%s.log", logPath, world.Name))
	}
	// 添加压缩命令
	cmds = append(cmds, "cd /tmp/dmp_tmp_logs && tar zcvf logs.tgz *")

	// 执行命令
	for _, cmd := range cmds {
		err = utils.BashCMD(cmd)
		if err != nil {
			utils.Logger.Error("创建压缩文件失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("tarFail", langStr), "data": nil})
			return
		}
	}
	// 读取文件内容
	fileData, err := os.ReadFile("/tmp/dmp_tmp_logs/logs.tgz")
	if err != nil {
		utils.Logger.Error("读取日志压缩文件失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("fileReadFail", langStr), "data": nil})
		return
	}
	fileContentBase64 := base64.StdEncoding.EncodeToString(fileData)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": fileContentBase64})
}
