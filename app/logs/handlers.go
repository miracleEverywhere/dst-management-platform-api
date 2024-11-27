package logs

import (
	"dst-management-platform-api/utils"
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
)

func handleLogGet(c *gin.Context) {
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
		logsValue, err := getLastNLines(utils.ChatLogPath, logForm.Line)
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
