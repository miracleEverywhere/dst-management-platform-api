package logs

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
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
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": []string{""}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": logsValue})
	case "caves":
		logsValue, err := getLastNLines(utils.CavesLogPath, logForm.Line)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": []string{""}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": logsValue})
	case "chat":
		logsValue, err := getLastNLines(utils.ChatLogPath, logForm.Line)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": []string{""}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": logsValue})
	case "dmp":
		logsValue, err := getLastNLines(utils.DMPLogPath, logForm.Line)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": []string{""}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": logsValue})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
	}
}
