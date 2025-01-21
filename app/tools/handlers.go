package tools

import (
	"dst-management-platform-api/app/externalApi"
	"dst-management-platform-api/app/setting"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/utils"
	"encoding/base64"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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
		utils.Logger.Error("获取系统信息失败", "err", err)
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
			utils.Logger.Error("删除文件失败", "err", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}
	}

	// 创建或打开文件
	file, err := os.OpenFile(scriptPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0775)
	if err != nil {
		utils.Logger.Error("打开文件失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			utils.Logger.Error("关闭文件失败", "err", err)
		}
	}(file)

	// 写入内容
	content := []byte(utils.ShInstall)
	_, err = file.Write(content)
	if err != nil {
		utils.Logger.Error("写入文件失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	// 异步执行脚本
	go func() {
		cmd := exec.Command("/bin/bash", scriptPath) // 使用 /bin/bash 执行脚本
		e := cmd.Run()
		if e != nil {
			utils.Logger.Error("执行安装脚本失败", "err", e)
		}
	}()

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("installing", langStr), "data": nil})
}

func handleGetInstallStatus(c *gin.Context) {
	content, err := os.ReadFile("/tmp/install_status")
	utils.Logger.Error("读取文件失败", "err", err)
	status := string(content)
	statusSlice := strings.Split(status, "\t")
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": gin.H{
		"process": statusSlice[0], "zh": statusSlice[1], "en": statusSlice[2],
	}})
}

func handleAnnounceGet(c *gin.Context) {
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}
	if config.AutoAnnounce == nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": []string{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": config.AutoAnnounce})
}

func handleAnnouncePost(c *gin.Context) {
	defer scheduler.ReloadScheduler()
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
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	for _, announce := range config.AutoAnnounce {
		if announce.Name == announceForm.Name {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("duplicatedName", langStr), "data": nil})
			return
		}
	}
	config.AutoAnnounce = append(config.AutoAnnounce, announceForm)
	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("createSuccess", langStr), "data": nil})
}

func handleAnnounceDelete(c *gin.Context) {
	defer scheduler.ReloadScheduler()
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
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	// 删除，遍历不添加
	for i := 0; i < len(config.AutoAnnounce); i++ {
		if config.AutoAnnounce[i].Name == announceForm.Name {
			config.AutoAnnounce = append(config.AutoAnnounce[:i], config.AutoAnnounce[i+1:]...)
			i--
		}
	}
	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("deleteSuccess", langStr), "data": nil})
}

func handleAnnouncePut(c *gin.Context) {
	defer scheduler.ReloadScheduler()
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
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	for index, announce := range config.AutoAnnounce {
		if announce.Name == announceForm.Name {
			config.AutoAnnounce[index] = announceForm
			err = utils.WriteConfig(config)
			if err != nil {
				utils.Logger.Error("配置文件写入失败", "err", err)
				utils.RespondWithError(c, 500, langStr)
				return
			}
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("updateSuccess", langStr), "data": nil})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("updateFail", langStr), "data": nil})
}

func handleUpdateGet(c *gin.Context) {
	dstVersion, err := externalApi.GetDSTVersion()
	if err != nil {
		utils.Logger.Error("获取饥荒版本失败", "err", err)
	}
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": gin.H{
		"version":       dstVersion,
		"updateSetting": config.AutoUpdate,
	}})
}

func handleUpdatePut(c *gin.Context) {
	defer scheduler.ReloadScheduler()
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
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	config.AutoUpdate.Time = updateForm.Time
	config.AutoUpdate.Enable = updateForm.Enable
	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("updateSuccess", langStr), "data": nil})
}

func handleBackupGet(c *gin.Context) {
	type BackFiles struct {
		Name       string `json:"name"`
		CreateTime string `json:"createTime"`
		Size       int64  `json:"size"`
	}
	var tmp []BackFiles
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}
	backupFiles, err := getBackupFiles()
	if err != nil {
		utils.Logger.Error("备份文件获取", "err", err)
	}
	for _, i := range backupFiles {
		var a BackFiles
		a.Name = i.Name
		a.CreateTime = i.ModTime.Format("2006-01-02 15:04:05")
		a.Size = i.Size
		tmp = append(tmp, a)
	}
	diskUsage, err := utils.DiskUsage()
	if err != nil {
		utils.Logger.Error("磁盘使用率获取失败", "err", err)
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": gin.H{
		"backupSetting": config.AutoBackup,
		"backupFiles":   tmp,
		"diskUsage":     diskUsage,
	}})
}

func handleBackupPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	err := utils.BackupGame()
	if err != nil {
		utils.Logger.Error("游戏备份失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("backupFail", langStr), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("backupSuccess", langStr), "data": nil})
}

func handleBackupPut(c *gin.Context) {
	defer scheduler.ReloadScheduler()
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
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	config.AutoBackup.Time = backupForm.Time
	config.AutoBackup.Enable = backupForm.Enable
	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

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
		utils.Logger.Error("备份文件删除失败", "err", err)
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
		utils.Logger.Error("恢复游戏失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("restoreFail", langStr), "data": nil})
		return
	}
	err = setting.WriteDatabase()
	if err != nil {
		utils.Logger.Error("恢复存档文件写入数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("restoreSuccessSaveFail", langStr), "data": nil})
		return
	}

	// 写入dedicated_server_mods_setup.lua
	err = setting.DstModsSetup()
	if err != nil {
		utils.Logger.Error("mod配置保存失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("saveFail", langStr), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("restoreSuccess", langStr), "data": nil})
}

func handleBackupDownload(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	type DownloadForm struct {
		Filename string `json:"filename"`
	}
	var downloadForm DownloadForm
	if err := c.ShouldBindJSON(&downloadForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	filePath := filepath.Join(utils.BackupPath, downloadForm.Filename)
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("fileNotFound", langStr), "data": nil})
		return
	}
	// 读取文件内容
	fileData, err := os.ReadFile(filePath)
	if err != nil {
		utils.Logger.Error("读取备份文件失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("fileReadFail", langStr), "data": nil})
		return
	}

	fileContentBase64 := base64.StdEncoding.EncodeToString(fileData)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": fileContentBase64})
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
		err := utils.RemoveFile(filePath)
		if err != nil {
			utils.Logger.Error("删除文件失败", "err", err, "file", filePath)
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("deleteSuccess", langStr), "data": nil})
}

func handleStatisticsGet(c *gin.Context) {
	type stats struct {
		Num       int   `json:"num"`
		Timestamp int64 `json:"timestamp"`
	}
	var data []stats
	for _, i := range utils.STATISTICS {
		var j stats
		j.Num = i.Num
		j.Timestamp = i.Timestamp
		data = append(data, j)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}

func handleKeepaliveGet(c *gin.Context) {
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": gin.H{
		"enable": config.Keepalive.Enable,
	}})
}

func handleKeepalivePut(c *gin.Context) {
	defer scheduler.ReloadScheduler()
	type UpdateForm struct {
		Enable bool `json:"enable"`
	}
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var updateForm UpdateForm
	if err := c.ShouldBindJSON(&updateForm); err != nil {
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
	config.Keepalive.Enable = updateForm.Enable
	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("updateSuccess", langStr), "data": nil})
}

func handleReplaceDSTSOFile(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	err := utils.ReplaceDSTSOFile()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("replaceFail", langStr), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("replaceSuccess", langStr), "data": nil})
}

func handleCreateTokenPost(c *gin.Context) {
	type ApiForm struct {
		ExpiredTime int64 `json:"expiredTime"`
	}

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var apiForm ApiForm
	if err := c.ShouldBindJSON(&apiForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	now := time.Now()
	nowTimestamp := now.UnixMilli()
	hours := (apiForm.ExpiredTime - nowTimestamp) / (60 * 60 * 1000)

	jwtSecret := []byte(config.JwtSecret)
	token, _ := utils.GenerateJWT(config.Username, jwtSecret, int(hours))

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("createTokenSuccess", langStr), "data": token})
}

func handleAnnouncedGet(c *gin.Context) {
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "error", "data": 0})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "error", "data": config.AnnouncedID})
}

func handleAnnouncedPost(c *gin.Context) {
	type AnnouncedForm struct {
		ID int `json:"id"`
	}
	var announcedForm AnnouncedForm
	if err := c.ShouldBindJSON(&announcedForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": err, "data": nil})
		return
	}

	config.AnnouncedID = announcedForm.ID
	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": err, "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "", "data": nil})
}

func handleMetricsGet(c *gin.Context) {
	type MetricsForm struct {
		// TimeRange 必须是分钟数
		TimeRange int `form:"timeRange" json:"timeRange"`
	}
	var metricsForm MetricsForm
	if err := c.ShouldBindQuery(&metricsForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	metricsLength := len(utils.SYS_METRICS)
	var metrics []utils.SysMetrics

	switch metricsForm.TimeRange {
	case 30:
		if metricsLength <= 60 {
			metrics = utils.SYS_METRICS
		} else {
			metrics = utils.SYS_METRICS[len(utils.SYS_METRICS)-60:]
		}
	case 60:
		if metricsLength <= 120 {
			metrics = utils.SYS_METRICS
		} else {
			metrics = utils.SYS_METRICS[len(utils.SYS_METRICS)-120:]
		}
	case 180:
		if metricsLength <= 360 {
			metrics = utils.SYS_METRICS
		} else {
			metrics = utils.SYS_METRICS[len(utils.SYS_METRICS)-360:]
		}
	default:
		metrics = utils.SYS_METRICS
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "error", "data": metrics})
}

func handleVersionGet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": utils.VERSION})
}
