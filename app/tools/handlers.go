package tools

import (
	"dst-management-platform-api/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

func handleOSInfoGet(c *gin.Context) {
	osInfo, err := utils.GetOSInfo()
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	if err != nil {
		utils.RespondWithError(c, 510, langStr)
		fmt.Println(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": osInfo})
}

func handleInstall(c *gin.Context) {
	scriptPath := "install.sh"

	// 检查文件是否存在，如果存在则删除
	if _, err := os.Stat(scriptPath); err == nil {
		err := os.Remove(scriptPath)
		if err != nil {
			fmt.Println("Error removing file:", err)
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to remove existing file", "data": nil})
			return
		}
	}

	// 创建或打开文件
	file, err := os.OpenFile(scriptPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0775)
	if err != nil {
		fmt.Println("Error opening file:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to open file", "data": nil})
		return
	}
	defer file.Close()

	// 写入内容
	content := []byte(utils.ShInstall)
	_, err = file.Write(content)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": "Failed to write to file", "data": nil})
		return
	}

	// 异步执行脚本
	go func() {
		cmd := exec.Command("/bin/bash", scriptPath) // 使用 /bin/bash 执行脚本
		e := cmd.Run()
		if e != nil {
			fmt.Println("Error executing script:", e)
			// 这里可以将错误记录到日志文件，或发送通知等
		} else {
			// 执行成功也可以记录
			fmt.Println("Script executed successfully")
		}
	}()

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "Script is being executed", "data": nil})
}

func handleGetInstallStatus(c *gin.Context) {
	content, _ := os.ReadFile("/tmp/install_status")
	status := string(content)
	statusSlice := strings.Split(status, "\t")
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": gin.H{
		"process": statusSlice[0], "zh": statusSlice[1], "en": statusSlice[2],
	}})
}
