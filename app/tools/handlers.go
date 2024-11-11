package tools

import (
	"dst-management-platform-api/app/home"
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
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	scriptPath := "install.sh"

	// 检查文件是否存在，如果存在则删除
	if _, err := os.Stat(scriptPath); err == nil {
		err := os.Remove(scriptPath)
		if err != nil {
			fmt.Println("Error removing file:", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}
	}

	// 创建或打开文件
	file, err := os.OpenFile(scriptPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0775)
	if err != nil {
		fmt.Println("Error opening file:", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	defer file.Close()

	// 写入内容
	content := []byte(utils.ShInstall)
	_, err = file.Write(content)
	if err != nil {
		fmt.Println("Error writing to file:", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	// 异步执行脚本
	go func() {
		cmd := exec.Command("/bin/bash", scriptPath) // 使用 /bin/bash 执行脚本
		e := cmd.Run()
		if e != nil {
			fmt.Println("Error executing script:", e)
		}
	}()

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("installing", langStr), "data": nil})
}

func handleGetInstallStatus(c *gin.Context) {
	content, _ := os.ReadFile("/tmp/install_status")
	status := string(content)
	statusSlice := strings.Split(status, "\t")
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": gin.H{
		"process": statusSlice[0], "zh": statusSlice[1], "en": statusSlice[2],
	}})
}

func handleAnnounceGet(c *gin.Context) {
	config, _ := utils.ReadConfig()
	if config.AutoAnnounce == nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": []string{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": config.AutoAnnounce})
}

func handleAnnouncePost(c *gin.Context) {
	defer reloadScheduler()
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var announceForm utils.AutoAnnounce
	if err := c.ShouldBindJSON(&announceForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config, _ := utils.ReadConfig()
	for _, announce := range config.AutoAnnounce {
		if announce.Name == announceForm.Name {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("duplicatedName", langStr), "data": nil})
			return
		}
	}
	config.AutoAnnounce = append(config.AutoAnnounce, announceForm)
	utils.WriteConfig(config)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("createSuccess", langStr), "data": nil})
}

func handleAnnounceDelete(c *gin.Context) {
	// 捕获函数退出时执行重启操作
	//defer func() {
	//	// 在函数返回后执行重启程序，但确保响应已经发送
	//	go func() {
	//		err := restartMyself()
	//		if err != nil {
	//			fmt.Println("重启失败:", err)
	//		}
	//	}()
	//}()
	defer reloadScheduler()
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var announceForm utils.AutoAnnounce
	if err := c.ShouldBindJSON(&announceForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config, _ := utils.ReadConfig()
	// 删除，遍历不添加
	for i := 0; i < len(config.AutoAnnounce); i++ {
		if config.AutoAnnounce[i].Name == announceForm.Name {
			config.AutoAnnounce = append(config.AutoAnnounce[:i], config.AutoAnnounce[i+1:]...)
			i--
		}
	}
	utils.WriteConfig(config)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("deleteSuccess", langStr), "data": nil})
}

func handleAnnouncePut(c *gin.Context) {
	defer reloadScheduler()
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var announceForm utils.AutoAnnounce
	if err := c.ShouldBindJSON(&announceForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config, _ := utils.ReadConfig()
	for index, announce := range config.AutoAnnounce {
		if announce.Name == announceForm.Name {
			config.AutoAnnounce[index] = announceForm
			utils.WriteConfig(config)
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("updateSuccess", langStr), "data": nil})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("updateFail", langStr), "data": nil})
}

func handleUpdateGet(c *gin.Context) {
	dstVersion, _ := home.GetDSTVersion()
	config, _ := utils.ReadConfig()

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": gin.H{
		"version":       dstVersion,
		"updateSetting": config.AutoUpdate,
	}})
}

func handleUpdatePut(c *gin.Context) {
	defer reloadScheduler()
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var updateForm utils.AutoUpdate
	if err := c.ShouldBindJSON(&updateForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config, _ := utils.ReadConfig()
	config.AutoUpdate.Time = updateForm.Time
	config.AutoUpdate.Enable = updateForm.Enable
	utils.WriteConfig(config)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("updateSuccess", langStr), "data": nil})
}

func handleBackupGet(c *gin.Context) {
	type BackFiles struct {
		Name       string `json:"name"`
		CreateTime string `json:"createTime"`
		Size       int64  `json:"size"`
	}
	var tmp []BackFiles
	config, _ := utils.ReadConfig()
	backupFiles, _ := getBackupFiles()
	for _, i := range backupFiles {
		var a BackFiles
		a.Name = i.Name
		a.CreateTime = i.ModTime.Format("2006-01-02 15:04:05")
		a.Size = i.Size
		tmp = append(tmp, a)
	}
	diskUsage, _ := utils.DiskUsage()
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": gin.H{
		"backupSetting": config.AutoBackup,
		"backupFiles":   tmp,
		"diskUsage":     diskUsage,
	}})
}

func handleBackupPut(c *gin.Context) {
	defer reloadScheduler()
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var backupForm utils.AutoBackup
	if err := c.ShouldBindJSON(&backupForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config, _ := utils.ReadConfig()
	config.AutoBackup.Time = backupForm.Time
	config.AutoBackup.Enable = backupForm.Enable
	utils.WriteConfig(config)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("updateSuccess", langStr), "data": nil})
}

func handleBackupDelete(c *gin.Context) {
	type DeleteForm struct {
		Name string `json:"name"`
	}
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var deleteForm DeleteForm
	if err := c.ShouldBindJSON(&deleteForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filePath := utils.BackupPath + "/" + deleteForm.Name
	err := utils.RemoveFile(filePath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("deleteFail", langStr), "data": nil})
	} else {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("deleteSuccess", langStr), "data": nil})
	}
}

func handleBackupRestore(c *gin.Context) {
	type RestoreForm struct {
		Name string `json:"name"`
	}
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var restoreForm RestoreForm
	if err := c.ShouldBindJSON(&restoreForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filePath := utils.BackupPath + "/" + restoreForm.Name
	err := utils.RecoveryGame(filePath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("restoreFail", langStr), "data": nil})
	} else {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("restoreSuccess", langStr), "data": nil})
	}
}

func handleMultiDelete(c *gin.Context) {
	type MultiDeleteForm struct {
		Names []string `json:"names"`
	}
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var multiDeleteForm MultiDeleteForm
	if err := c.ShouldBindJSON(&multiDeleteForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	for _, file := range multiDeleteForm.Names {
		filePath := utils.BackupPath + "/" + file
		_ = utils.RemoveFile(filePath)
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("deleteSuccess", langStr), "data": nil})
}

func handleDownloadModManualPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	modList := utils.GetModList()
	utils.DownloadMod(modList)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("deleteSuccess", langStr), "data": nil})
}

func handleStatisticsGet(c *gin.Context) {
	config, _ := utils.ReadConfig()
	data := config.Statistics
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}
