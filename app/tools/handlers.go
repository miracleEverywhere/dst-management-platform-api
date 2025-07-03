package tools

import (
	"dst-management-platform-api/app/setting"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/utils"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/ssh"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/websocket"
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
		utils.DstInstalling = true
		cmd := exec.Command("/bin/bash", scriptPath) // 使用 /bin/bash 执行脚本
		e := cmd.Run()
		if e != nil {
			utils.Logger.Error("执行安装脚本失败", "err", e)
		}
		utils.DstInstalling = false
	}()

	// 返回成功响应
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("installing", langStr), "data": nil})
}

func handleGetInstallStatus(c *gin.Context) {
	content, err := os.ReadFile("/tmp/install_status")
	if err != nil {
		utils.Logger.Error("读取文件失败", "err", err)
	}
	status := string(content)
	statusSlice := strings.Split(status, "\t")

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": gin.H{
		"process": statusSlice[0], "zh": statusSlice[1], "en": statusSlice[2],
	}})
}

func handleGetIsInstallingGet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": utils.DstInstalling})
}

func handleAnnounceGet(c *gin.Context) {
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
		utils.RespondWithError(c, 500, "zh")
		return
	}

	cluster, err := config.GetClusterWithName(reqForm.ClusterName)
	if err != nil {
		utils.Logger.Error("获取集群失败", "err", err)
		utils.RespondWithError(c, 404, "zh")
		return
	}

	if cluster.SysSetting.AutoAnnounce == nil {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": []string{}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": cluster.SysSetting.AutoAnnounce})
}

func handleAnnouncePost(c *gin.Context) {
	defer scheduler.ReloadScheduler()
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	type ReqForm struct {
		AutoAnnounce utils.AutoAnnounce `json:"autoAnnounce"`
		ClusterName  string             `json:"clusterName" form:"clusterName"`
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
		utils.RespondWithError(c, 404, langStr)
		return
	}

	for _, announce := range cluster.SysSetting.AutoAnnounce {
		if announce.Name == reqForm.AutoAnnounce.Name {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("duplicatedName", langStr), "data": nil})
			return
		}
	}
	cluster.SysSetting.AutoAnnounce = append(cluster.SysSetting.AutoAnnounce, reqForm.AutoAnnounce)
	for index, dbCluster := range config.Clusters {
		if cluster.ClusterSetting.ClusterName == dbCluster.ClusterSetting.ClusterName {
			config.Clusters[index] = cluster
			err = utils.WriteConfig(config)
			if err != nil {
				utils.Logger.Error("配置文件写入失败", "err", err)
				utils.RespondWithError(c, 500, langStr)
				return
			}
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("createSuccess", langStr), "data": nil})
			return
		}
	}

	utils.RespondWithError(c, 404, langStr)
}

func handleAnnounceDelete(c *gin.Context) {
	defer scheduler.ReloadScheduler()
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	type ReqForm struct {
		AutoAnnounce utils.AutoAnnounce `json:"autoAnnounce"`
		ClusterName  string             `json:"clusterName" form:"clusterName"`
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
		utils.RespondWithError(c, 404, langStr)
		return
	}
	// 删除，遍历不添加
	for i := 0; i < len(cluster.SysSetting.AutoAnnounce); i++ {
		if cluster.SysSetting.AutoAnnounce[i].Name == reqForm.AutoAnnounce.Name {
			cluster.SysSetting.AutoAnnounce = append(cluster.SysSetting.AutoAnnounce[:i], cluster.SysSetting.AutoAnnounce[i+1:]...)
			i--
		}
	}
	for index, dbCluster := range config.Clusters {
		if cluster.ClusterSetting.ClusterName == dbCluster.ClusterSetting.ClusterName {
			config.Clusters[index] = cluster
			err = utils.WriteConfig(config)
			if err != nil {
				utils.Logger.Error("配置文件写入失败", "err", err)
				utils.RespondWithError(c, 500, langStr)
				return
			}
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("deleteSuccess", langStr), "data": nil})
			return
		}
	}

	utils.RespondWithError(c, 404, langStr)
}

func handleAnnouncePut(c *gin.Context) {
	defer scheduler.ReloadScheduler()
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	type ReqForm struct {
		AutoAnnounce utils.AutoAnnounce `json:"autoAnnounce"`
		ClusterName  string             `json:"clusterName" form:"clusterName"`
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
		utils.RespondWithError(c, 404, langStr)
		return
	}

	for index, announce := range cluster.SysSetting.AutoAnnounce {
		if announce.Name == reqForm.AutoAnnounce.Name {
			cluster.SysSetting.AutoAnnounce[index] = reqForm.AutoAnnounce
			for dbIndex, dbCluster := range config.Clusters {
				if cluster.ClusterSetting.ClusterName == dbCluster.ClusterSetting.ClusterName {
					config.Clusters[dbIndex] = cluster
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
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("updateFail", langStr), "data": nil})
}

func handleBackupGet(c *gin.Context) {
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
		utils.RespondWithError(c, 500, "zh")
		return
	}

	cluster, err := config.GetClusterWithName(reqForm.ClusterName)
	if err != nil {
		utils.Logger.Error("获取集群失败", "err", err)
		utils.RespondWithError(c, 404, "zh")
		return
	}

	type BackFiles struct {
		Name       string `json:"name"`
		CreateTime string `json:"createTime"`
		Size       int64  `json:"size"`
	}
	var tmp []BackFiles

	backupFiles, err := getBackupFiles(cluster)
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
		"backupFiles": tmp,
		"diskUsage":   diskUsage,
	}})
}

func handleBackupPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	type ReqForm struct {
		ClusterName string `json:"clusterName" form:"clusterName"`
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
		utils.RespondWithError(c, 404, langStr)
		return
	}

	err = utils.BackupGame(cluster)
	if err != nil {
		utils.Logger.Error("游戏备份失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("backupFail", langStr), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("backupSuccess", langStr), "data": nil})
}

func handleBackupDelete(c *gin.Context) {
	type DeleteForm struct {
		ClusterName string `json:"clusterName" form:"clusterName"`
		Name        string `json:"name"`
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

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	cluster, err := config.GetClusterWithName(deleteForm.ClusterName)
	if err != nil {
		utils.RespondWithError(c, 404, langStr)
		return
	}

	filePath := cluster.GetBackupPath() + "/" + deleteForm.Name
	err = utils.RemoveFile(filePath)
	if err != nil {
		utils.Logger.Error("备份文件删除失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("deleteFail", langStr), "data": nil})
	} else {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("deleteSuccess", langStr), "data": nil})
	}
}

func handleBackupRestore(c *gin.Context) {
	defer func() {
		setting.ClearFiles()
	}()

	type RestoreForm struct {
		ClusterName string `json:"clusterName" form:"clusterName"`
		Name        string `json:"name"`
	}
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var (
		restoreForm         RestoreForm
		cluster             utils.Cluster
		currentClusterIndex = -1
		backupClusterIndex  = -1
	)
	if err := c.ShouldBindJSON(&restoreForm); err != nil {
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

	for index, cl := range config.Clusters {
		if cl.ClusterSetting.ClusterName == restoreForm.ClusterName {
			currentClusterIndex = index
			break
		}
	}

	if currentClusterIndex == -1 {
		utils.RespondWithError(c, 404, langStr)
		return
	}

	cluster = config.Clusters[currentClusterIndex]

	// 关闭当前服务器
	_ = utils.StopClusterAllWorlds(cluster)

	filePath := cluster.GetBackupPath() + "/" + restoreForm.Name

	// 解压tgz文件
	cmd := fmt.Sprintf("tar zxf %s -C %s", filePath, utils.ImportFileUploadPath)
	err = utils.BashCMD(cmd)
	if err != nil {
		utils.Logger.Error("解压失败", "err", err, "cmd", cmd)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("restoreFail", langStr), "data": nil})
		return
	}

	// 还原备份文件
	cmd = fmt.Sprintf("rm -rf %s", cluster.GetMainPath())
	err = utils.BashCMD(cmd)
	if err != nil {
		utils.Logger.Error("删除旧集群文件失败", "err", err, "cmd", cmd)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("restoreFail", langStr), "data": nil})
		return
	}
	cmd = fmt.Sprintf("mv %s%s %s", utils.ImportFileUploadPath, cluster.GetMainPath(), cluster.GetMainPath())
	err = utils.BashCMD(cmd)
	if err != nil {
		utils.Logger.Error("创建新集群文件失败", "err", err, "cmd", cmd)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("restoreFail", langStr), "data": nil})
		return
	}
	// 读取备份的配置文件
	backupConfig, err := utils.ReadBackupConfig(utils.ImportFileUploadPath + "/DstMP.sdb")
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	for index, backupCluster := range backupConfig.Clusters {
		if backupCluster.ClusterSetting.ClusterName == cluster.ClusterSetting.ClusterName {
			backupClusterIndex = index
			break
		}
	}

	if backupClusterIndex == -1 {
		utils.Logger.Error("旧配置文件中没有找到对应的集群")
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("restoreFailOldClusterNotFound", langStr), "data": nil})
		return
	}

	config.Clusters[currentClusterIndex] = backupConfig.Clusters[backupClusterIndex]

	cluster, err = config.GetClusterWithName(restoreForm.ClusterName)
	if err != nil {
		utils.RespondWithError(c, 404, langStr)
		return
	}

	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("写入配置文件失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	err = cluster.ClearDstFiles()
	if err != nil {
		utils.Logger.Error("删除旧集群脏数据失败")
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("restoreSuccess", langStr), "data": nil})
}

/* func handleBackupDownload(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	type DownloadForm struct {
		ClusterName string `json:"clusterName" form:"clusterName"`
		Filename    string `json:"filename"`
	}
	var downloadForm DownloadForm
	if err := c.ShouldBindJSON(&downloadForm); err != nil {
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

	cluster, err := config.GetClusterWithName(downloadForm.ClusterName)
	if err != nil {
		utils.RespondWithError(c, 404, langStr)
		return
	}

	filePath := filepath.Join(cluster.GetBackupPath(), downloadForm.Filename)
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
} */

func handleMultiDelete(c *gin.Context) {
	type MultiDeleteForm struct {
		ClusterName string   `json:"clusterName" form:"clusterName"`
		Names       []string `json:"names"`
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

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	cluster, err := config.GetClusterWithName(multiDeleteForm.ClusterName)
	if err != nil {
		utils.RespondWithError(c, 404, langStr)
		return
	}

	for _, file := range multiDeleteForm.Names {
		filePath := cluster.GetBackupPath() + "/" + file
		err := utils.RemoveFile(filePath)
		if err != nil {
			utils.Logger.Error("删除文件失败", "err", err, "file", filePath)
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("deleteSuccess", langStr), "data": nil})
}

func handleStatisticsGet(c *gin.Context) {
	type ReqForm struct {
		ClusterName string `json:"clusterName" form:"clusterName"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var statistics []utils.Statistics
	for key, _ := range utils.STATISTICS {
		if key == reqForm.ClusterName {
			statistics = utils.STATISTICS[key]
		}
	}

	if len(statistics) == 0 {
		utils.RespondWithError(c, 404, "zh")
		return
	}

	type stats struct {
		Num       int   `json:"num"`
		Timestamp int64 `json:"timestamp"`
	}
	type GanttRowItem struct {
		BeginDate string `json:"beginDate"`
		EndDate   string `json:"endDate"`
		ID        string `json:"id"`
		Label     string `json:"label"`
	}

	type Data struct {
		Stats []stats                   `json:"stats"`
		Gantt map[string][]GanttRowItem `json:"gantt"`
		Pie   map[string]int64          `json:"pie"`
	}

	var (
		data          Data
		gantt         = make(map[string][]GanttRowItem)
		activePlayers = make(map[string]bool)
	)
	for i, stat := range statistics {
		// 折线图
		var j stats
		j.Num = stat.Num
		j.Timestamp = stat.Timestamp
		data.Stats = append(data.Stats, j)

		// 甘特图
		currentName := make(map[string]bool)
		// 构建当前时间点的nickname集合
		for _, players := range stat.Players {
			currentName[players.NickName] = true
		}
		// 处理新出现的nickname(beginTime)
		for nickname := range currentName {
			if !activePlayers[nickname] {
				// 如果nickname之前不活跃，现在活跃，开始新的时间段
				if _, exists := gantt[nickname]; !exists {
					gantt[nickname] = []GanttRowItem{}
				}
				gantt[nickname] = append(gantt[nickname], GanttRowItem{
					BeginDate: utils.TimestampToTimestring(stat.Timestamp),
				})
			}
		}

		// 处理不活跃(离线)nickname，即endDate
		for nickname := range activePlayers {
			if !currentName[nickname] {
				if ranges, exists := gantt[nickname]; exists && len(ranges) > 0 {
					lastIdx := len(ranges) - 1
					if ranges[lastIdx].EndDate == "" {
						// 确保未设置endDate
						ranges[lastIdx].EndDate = utils.TimestampToTimestring(stat.Timestamp)
						gantt[nickname] = ranges
					}
				}
			}
		}

		// 如果当前时间点还有活跃nickname，就为所有活跃的nickname设置endDate
		if i == len(statistics)-1 {
			for nickname := range currentName {
				if ranges, exists := gantt[nickname]; exists && len(ranges) > 0 {
					lastIdx := len(ranges) - 1
					if ranges[lastIdx].EndDate == "" {
						ranges[lastIdx].EndDate = utils.TimestampToTimestring(stat.Timestamp)
						gantt[nickname] = ranges
					}
				}
			}
		}

		// 更新活跃nickname集合
		activePlayers = currentName
	}

	for key, value := range gantt {
		for index, row := range value {
			gantt[key][index].ID = fmt.Sprintf("%s-%d", key, index)
			beginT, err := time.Parse("2006-01-02 15:04", row.BeginDate)
			if err != nil {
				utils.Logger.Error("时间转换错误", "err", err)
			}
			endT, err := time.Parse("2006-01-02 15:04", row.EndDate)
			if err != nil {
				utils.Logger.Error("时间转换错误", "err", err)
			}
			duration := endT.Sub(beginT)

			gantt[key][index].Label = fmt.Sprintf("%.0f", duration.Minutes())
		}
	}

	data.Gantt = gantt
	data.Pie = utils.PlayTimeCount[reqForm.ClusterName]

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
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

	jwtSecret := []byte(config.JwtSecret)
	usernameValue, _ := c.Get("username")
	username := fmt.Sprintf("%v", usernameValue)

	for _, user := range config.Users {
		if user.Username == username {
			token, _ := utils.GenerateJWT(user, jwtSecret, int(apiForm.ExpiredTime))
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("createTokenSuccess", langStr), "data": token})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("createTokenFail", langStr), "data": nil})
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

	metricsLength := len(utils.SYSMETRICS)
	var metrics []utils.SysMetrics

	switch metricsForm.TimeRange {
	case 30:
		if metricsLength <= 60 {
			metrics = utils.SYSMETRICS
		} else {
			metrics = utils.SYSMETRICS[len(utils.SYSMETRICS)-60:]
		}
	case 60:
		if metricsLength <= 120 {
			metrics = utils.SYSMETRICS
		} else {
			metrics = utils.SYSMETRICS[len(utils.SYSMETRICS)-120:]
		}
	case 180:
		if metricsLength <= 360 {
			metrics = utils.SYSMETRICS
		} else {
			metrics = utils.SYSMETRICS[len(utils.SYSMETRICS)-360:]
		}
	default:
		metrics = utils.SYSMETRICS
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "error", "data": metrics})
}

func handleVersionGet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": utils.VERSION})
}

func handleWebSSHGet(c *gin.Context) {
	var upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	type WSMessage struct {
		Type string `json:"type"`
		Data string `json:"data"`
		Cols int    `json:"cols"`
		Rows int    `json:"rows"`
	}

	ip := c.Query("ip")
	port := c.DefaultQuery("port", "22")
	username := c.Query("username")
	password := c.Query("password")
	token := c.Query("token")

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件打开失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}
	tokenSecret := config.JwtSecret
	claims, err := utils.ValidateJWT(token, []byte(tokenSecret))
	if err != nil {
		utils.RespondWithError(c, 420, "zh")
		return
	}

	if claims.Role != "admin" {
		utils.RespondWithError(c, 425, "zh")
		return
	}

	if ip == "" || username == "" || password == "" {
		utils.Logger.Warn("webssh：必要信息为空")
		c.JSON(http.StatusBadRequest, gin.H{"error": "err"})
		return
	}

	passwordBase64, _ := base64.StdEncoding.DecodeString(password)

	// aes加解密有问题，需要处理
	passwordBytes, err := utils.AesDecrypt(passwordBase64, utils.GetAesKey())
	if err != nil {
		utils.Logger.Warn("aes解密失败", "err", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "err"})
		return
	}

	password = string(passwordBytes)

	address := net.JoinHostPort(ip, port)

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		utils.Logger.Error("WS upgrade 错误", "err", err)
		return
	}
	utils.Logger.Info("WebSocket连接已建立")

	defer func(conn *websocket.Conn) {
		err := conn.Close()
		if err != nil {
			utils.Logger.Warn("WS 关闭失败", "err", err)
		}
	}(conn)

	sshConfig := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         30 * time.Second,
	}

	sshConn, err := ssh.Dial("tcp", address, sshConfig)
	if err != nil {
		utils.Logger.Error("WS dial 错误", "err", err)
		return
	}
	defer func(sshConn *ssh.Client) {
		err := sshConn.Close()
		if err != nil {
			utils.Logger.Warn("ssh 关闭失败", "err", err)
		}
	}(sshConn)

	session, err := sshConn.NewSession()
	if err != nil {
		utils.Logger.Error("ssh session 错误", "err", err)
		return
	}
	defer func(session *ssh.Session) {
		err := session.Close()
		if err != nil {
			utils.Logger.Warn("ssh session 关闭失败", "err", err)
		}
	}(session)

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	cols := 80
	rows := 40
	err = session.RequestPty("xterm", rows, cols, modes)
	if err != nil {
		utils.Logger.Error("request pty 错误", "err", err)
		return
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		utils.Logger.Error("stdin pipe 错误", "err", err)
		return
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		utils.Logger.Error("stdout pipe 错误", "err", err)
		return
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		utils.Logger.Error("stderr pipe 错误", "err", err)
		return
	}

	err = session.Shell()
	if err != nil {
		utils.Logger.Error("启动 shell 失败", "err", err)
		return
	}

	go func() {
		for {
			_, msg, err := conn.ReadMessage()
			if err != nil {
				utils.Logger.Warn("WS 读取错误", "err", err)
				return
			}

			var wsMsg WSMessage
			err = json.Unmarshal(msg, &wsMsg)
			if err != nil {
				utils.Logger.Warn("ws json unmarshal 错误", "err", err)
				continue
			}

			switch wsMsg.Type {
			case "input":
				_, err = stdin.Write([]byte(wsMsg.Data))
				if err != nil {
					utils.Logger.Warn("ssh stdin 错误", "err", err)
				}
			case "resize":
				cols = wsMsg.Cols
				rows = wsMsg.Rows
				err = session.WindowChange(rows, cols)
				if err != nil {
					utils.Logger.Warn("window resize 错误", "err", err)
				}
			}
		}
	}()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				utils.Logger.Error("ssh stdout 读取错误", "err", err)
				return
			}
			outputMsg := WSMessage{
				Type: "output",
				Data: string(buf[:n]),
			}
			err = conn.WriteJSON(outputMsg)
			if err != nil {
				utils.Logger.Warn("ssh stdout 写入错误", "err", err)
			}
		}
	}()

	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := stderr.Read(buf)
			if err != nil {
				utils.Logger.Error("ssh stderr 读取错误", "err", err)
				return
			}
			outputMsg := WSMessage{
				Type: "output",
				Data: string(buf[:n]),
			}
			err = conn.WriteJSON(outputMsg)
			if err != nil {
				utils.Logger.Warn("ssh stderr 写入错误", "err", err)
			}
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
}
