package setting

import (
	"dst-management-platform-api/app/externalApi"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/tealeg/xlsx"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

func handleClustersGet(c *gin.Context) {
	username, _ := c.Get("username")
	role, _ := c.Get("role")
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	type ClusterItem struct {
		ClusterName        string   `json:"clusterName"`
		ClusterDisplayName string   `json:"clusterDisplayName"`
		Worlds             []string `json:"worlds"`
		Status             bool     `json:"status"`
	}
	var data []ClusterItem

	if role == "admin" {
		// 管理员返回所有cluster
		for _, cluster := range config.Clusters {
			var worlds []string
			for _, world := range cluster.Worlds {
				worlds = append(worlds, world.Name)
			}
			data = append(data, ClusterItem{
				ClusterName:        cluster.ClusterSetting.ClusterName,
				ClusterDisplayName: cluster.ClusterSetting.ClusterDisplayName,
				Worlds:             worlds,
				Status:             cluster.ClusterSetting.Status,
			})
		}
	} else {
		for i, user := range config.Users {
			if user.Username == username {
				for _, clusterName := range config.Users[i].ClusterPermission {
					cluster, err := config.GetClusterWithName(clusterName)
					if err != nil {
						continue
					} else {
						var worlds []string
						for _, world := range cluster.Worlds {
							worlds = append(worlds, world.Name)
						}
						data = append(data, ClusterItem{
							ClusterName:        cluster.ClusterSetting.ClusterName,
							ClusterDisplayName: cluster.ClusterSetting.ClusterDisplayName,
							Worlds:             worlds,
							Status:             cluster.ClusterSetting.Status,
						})
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}

func handleAllClustersGet(c *gin.Context) {
	type ClusterItem struct {
		ClusterName        string   `json:"clusterName"`
		ClusterDisplayName string   `json:"clusterDisplayName"`
		Worlds             []string `json:"worlds"`
	}
	var data []ClusterItem

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	for _, cluster := range config.Clusters {
		var worlds []string
		for _, world := range cluster.Worlds {
			worlds = append(worlds, world.Name)
		}
		data = append(data, ClusterItem{
			ClusterName:        cluster.ClusterSetting.ClusterName,
			ClusterDisplayName: cluster.ClusterSetting.ClusterDisplayName,
			Worlds:             worlds,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}

func handleClustersWorldPortGet(c *gin.Context) {
	type ResponseCluster struct {
		ClusterName        string `json:"clusterName"`
		ClusterDisplayName string `json:"clusterDisplayName"`
		WorldPort          []int  `json:"worldPort"`
	}
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	var responseClusters []ResponseCluster
	for _, cluster := range config.Clusters {
		var worldPort []int
		for _, world := range cluster.Worlds {
			worldPort = append(worldPort, world.ServerPort)
			worldPort = append(worldPort, world.SteamMasterPort)
			worldPort = append(worldPort, world.SteamAuthenticationPort)
		}
		// master port一致 不重复添加
		if cluster.Worlds != nil {
			worldPort = append(worldPort, cluster.Worlds[0].ShardMasterPort)
		}

		responseClusters = append(responseClusters, ResponseCluster{
			ClusterName:        cluster.ClusterSetting.ClusterName,
			ClusterDisplayName: cluster.ClusterSetting.ClusterDisplayName,
			WorldPort:          worldPort,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": responseClusters})
}

func handleClusterGet(c *gin.Context) {
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

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": cluster})
}

func handleClusterPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	username, _ := c.Get("username")
	role, _ := c.Get("role")

	type ReqForm struct {
		ClusterName        string `json:"clusterName"`
		ClusterDisplayName string `json:"clusterDisplayName"`
	}
	var reqFrom ReqForm
	if err := c.ShouldBindJSON(&reqFrom); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	for _, cluster := range config.Clusters {
		if cluster.ClusterSetting.ClusterName == reqFrom.ClusterName {
			c.JSON(http.StatusOK, gin.H{
				"code":    201,
				"message": response("clusterExisted", langStr),
				"data":    nil,
			})
			return
		}
	}

	var cluster utils.Cluster
	cluster.ClusterSetting.ClusterName = reqFrom.ClusterName
	cluster.ClusterSetting.ClusterDisplayName = reqFrom.ClusterDisplayName
	cluster.ClusterSetting.Status = true
	cluster.ClusterSetting.ConsoleEnabled = true
	cluster.SysSetting = utils.SysSetting{
		AutoRestart: utils.AutoRestart{
			Enable: true,
			Time:   "06:47:19",
		},
		AutoAnnounce: nil,
		AutoBackup: utils.AutoBackup{
			Enable: true,
			Time:   "06:13:57",
		},
		Keepalive: utils.Keepalive{
			Enable:    true,
			Frequency: 30,
		},
		Bit64:    false,
		TickRate: 15,
	}

	config.Clusters = append(config.Clusters, cluster)

	// 添加对应的用户权限
	if role != "admin" {
		for userIndex, user := range config.Users {
			if user.Username == username {
				if user.ClusterCreationProhibited {
					c.JSON(http.StatusOK, gin.H{
						"code":    201,
						"message": response("clusterCreationProhibited", langStr),
						"data":    nil,
					})
					return
				}
				config.Users[userIndex].ClusterPermission = append(config.Users[userIndex].ClusterPermission, reqFrom.ClusterName)
			}
		}
	}

	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("写入配置文件失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	// 重新载入定时任务
	defer func() {
		scheduler.ReloadScheduler()
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": response("createSuccess", langStr),
		"data":    nil,
	})
}

func handleClusterPut(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	type ReqForm struct {
		ClusterName        string `json:"clusterName"`
		ClusterDisplayName string `json:"clusterDisplayName"`
	}
	var reqFrom ReqForm
	if err := c.ShouldBindJSON(&reqFrom); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	for index, cluster := range config.Clusters {
		if cluster.ClusterSetting.ClusterName == reqFrom.ClusterName {
			cluster.ClusterSetting.ClusterDisplayName = reqFrom.ClusterDisplayName
			config.Clusters[index] = cluster
			err = utils.WriteConfig(config)
			if err != nil {
				utils.Logger.Error("写入配置文件失败", "err", err)
				utils.RespondWithError(c, 500, "zh")
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"code":    200,
				"message": response("configUpdateSuccess", langStr),
				"data":    nil,
			})
			return
		}
	}

	utils.RespondWithError(c, 404, langStr)
}

func handleClusterDelete(c *gin.Context) {
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
		utils.RespondWithError(c, 404, "zh")
		return
	}

	// 生成新数据库配置
	var (
		newClusters []utils.Cluster
		newUsers    []utils.User
		cmd         string
	)
	for _, dbCluster := range config.Clusters {
		if dbCluster.ClusterSetting.ClusterName != reqForm.ClusterName {
			newClusters = append(newClusters, dbCluster)
		}
	}
	for _, user := range config.Users {
		newPermission := utils.RemoveSliceOne(user.ClusterPermission, reqForm.ClusterName)
		user.ClusterPermission = newPermission
		newUsers = append(newUsers, user)
	}
	config.Clusters = newClusters
	config.Users = newUsers

	// 停止游戏进程
	_ = utils.StopClusterAllWorlds(cluster)

	// 删除游戏配置
	cmd = fmt.Sprintf("rm -rf %s", cluster.GetMainPath())
	err = utils.BashCMD(cmd)
	if err != nil {
		utils.Logger.Warn("删除游戏配置文件失败", "err", err, "cmd", cmd)
	}

	// 删除mod缓存
	cmd = fmt.Sprintf("rm -rf %s", cluster.GetModUgcPathRoot())
	err = utils.BashCMD(cmd)
	if err != nil {
		utils.Logger.Warn("删除mod缓存失败", "err", err, "cmd", cmd)
	}

	// 删除统计信息缓存
	delete(utils.STATISTICS, cluster.ClusterSetting.ClusterName)

	// 更新数据库
	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("写入配置文件失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	// 更新用户信息缓存
	for _, user := range config.Users {
		utils.UserCache[user.Username] = user
	}

	// 重新载入定时任务
	defer func() {
		scheduler.ReloadScheduler()
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": response("deleteSuccess", langStr),
		"data":    nil,
	})
}

func handleClusterStatusPut(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	type ReqForm struct {
		ClusterName string `json:"clusterName"`
		Status      bool   `json:"status"`
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

	clusterIndex, cluster, err := config.GetClusterAndIndexWithName(reqForm.ClusterName)
	if err != nil {
		utils.Logger.Error("获取集群失败", "err", err)
		utils.RespondWithError(c, 404, "zh")
		return
	}

	if reqForm.Status {
		// 启用集群

		// 修改集群状态
		cluster.ClusterSetting.Status = true
		// 启用定时任务
		cluster.SysSetting.AutoRestart.Enable = true
		for _, announce := range cluster.SysSetting.AutoAnnounce {
			announce.Enable = true
		}
		cluster.SysSetting.AutoBackup.Enable = true
		cluster.SysSetting.Keepalive.Enable = true
		cluster.SysSetting.ScheduledStartStop.Enable = true

		// 启动世界
		_ = utils.StartClusterAllWorlds(cluster)
	} else {
		// 关闭集群

		// 修改集群状态
		cluster.ClusterSetting.Status = false
		// 关闭世界
		_ = utils.StopClusterAllWorlds(cluster)
		// 禁用定时任务
		cluster.SysSetting.AutoRestart.Enable = false
		for _, announce := range cluster.SysSetting.AutoAnnounce {
			announce.Enable = false
		}
		cluster.SysSetting.AutoBackup.Enable = false
		cluster.SysSetting.Keepalive.Enable = false
		cluster.SysSetting.ScheduledStartStop.Enable = false
	}

	// 更新数据库
	config.Clusters[clusterIndex] = cluster
	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("写入配置文件失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	// 重新载入定时任务
	defer func() {
		scheduler.ReloadScheduler()
	}()

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": response("executed", langStr),
		"data":    nil,
	})
}

func handleClusterSavePost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	var reqCluster utils.Cluster
	if err := c.ShouldBindJSON(&reqCluster); err != nil {
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
	cluster, err := config.GetClusterWithName(reqCluster.ClusterSetting.ClusterName)
	if err != nil {
		utils.Logger.Error("获取集群失败", "err", err)
		utils.RespondWithError(c, 404, "zh")
		return
	}

	KillInvalidScreen(cluster.Worlds, reqCluster.Worlds)

	err = SaveSetting(reqCluster)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    201,
			"message": response("saveFail", langStr),
			"data":    nil,
		})
		return
	}

	err = reqCluster.ClearDstFiles()
	if err != nil {
		utils.Logger.Error("删除旧集群脏数据失败")
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": response("saveSuccess", langStr),
		"data":    nil,
	})
}

func handleClusterSaveRestartPost(c *gin.Context) {
	defer func() {
		time.Sleep(10 * time.Second)
		_ = utils.BashCMD("screen -wipe")
	}()

	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	var reqCluster utils.Cluster
	if err := c.ShouldBindJSON(&reqCluster); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_ = utils.BashCMD("screen -wipe")

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("读取配置文件失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}
	cluster, err := config.GetClusterWithName(reqCluster.ClusterSetting.ClusterName)
	if err != nil {
		utils.Logger.Error("获取集群失败", "err", err)
		utils.RespondWithError(c, 404, "zh")
		return
	}

	KillInvalidScreen(cluster.Worlds, reqCluster.Worlds)

	err = SaveSetting(reqCluster)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    201,
			"message": response("saveFail", langStr),
			"data":    nil,
		})
		return
	}

	err = reqCluster.ClearDstFiles()
	if err != nil {
		utils.Logger.Error("删除旧集群脏数据失败")
	}

	_ = utils.StopClusterAllWorlds(reqCluster)
	err = utils.StartClusterAllWorlds(reqCluster)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    201,
			"message": response("saveSuccessRestartFail", langStr),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("restartSuccess", langStr), "data": nil})
}

func handleClusterSaveRegeneratePost(c *gin.Context) {
	defer func() {
		time.Sleep(10 * time.Second)
		_ = utils.BashCMD("screen -wipe")
	}()

	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	var reqCluster utils.Cluster
	if err := c.ShouldBindJSON(&reqCluster); err != nil {
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
	cluster, err := config.GetClusterWithName(reqCluster.ClusterSetting.ClusterName)
	if err != nil {
		utils.Logger.Error("获取集群失败", "err", err)
		utils.RespondWithError(c, 404, "zh")
		return
	}

	KillInvalidScreen(cluster.Worlds, reqCluster.Worlds)

	err = SaveSetting(reqCluster)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    201,
			"message": response("saveFail", langStr),
			"data":    nil,
		})
		return
	}

	err = utils.StartClusterAllWorlds(reqCluster)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    201,
			"message": response("saveSuccessRestartFail", langStr),
			"data":    nil,
		})
		return
	}

	err = reqCluster.ClearDstFiles()
	if err != nil {
		utils.Logger.Error("删除旧集群脏数据失败")
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("generateSuccess", langStr), "data": nil})
}

func handleImportPost(c *gin.Context) {
	defer func() {
		ClearFiles()
	}()

	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, err.Error())
		return
	}
	clusterName := c.PostForm("clusterName")
	if clusterName == "" {
		c.JSON(http.StatusBadRequest, "缺少集群名")
		return
	}

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}
	cluster, err := config.GetClusterWithName(clusterName)
	if err != nil {
		utils.Logger.Error("获取集群失败", "err", err)
		utils.RespondWithError(c, 404, "zh")
		return
	}

	if cluster.Worlds != nil {
		utils.Logger.Info("被导入的集群中的世界不为空")
		c.JSON(http.StatusOK, gin.H{
			"code":    201,
			"message": responseImportError("worldNotEmpty", langStr),
			"data":    nil,
		})
		return
	}

	//保存文件
	savePath := utils.ImportFileUploadPath + file.Filename
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		utils.Logger.Error("文件保存失败", "err", err)
		c.JSON(http.StatusOK, gin.H{
			"code":    201,
			"message": responseImportError("zipFileSave", langStr),
			"data":    nil,
		})
		return
	}
	//执行导入
	result, msg, cluster, lists, dstFiles := DoImport(file.Filename, cluster, langStr)
	if !result {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": responseImportError(msg, langStr), "data": nil})
		return
	}
	//写入三个名单
	clusterPath := cluster.GetMainPath() + "/"
	err = utils.EnsureDirExists(clusterPath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    201,
			"message": responseImportError("clusterDirCreateFail", langStr),
			"data":    nil,
		})
		return
	}
	for key, value := range lists {
		err = utils.EnsureFileExists(clusterPath + key)
		if err != nil {
			utils.Logger.Error("创建"+key+"文件失败", "err", err)
			continue
		}
		err = utils.WriteLinesFromSlice(clusterPath+key, value)
		if err != nil {
			utils.Logger.Error("写入"+key+"文件失败", "err", err)
			continue
		}
	}
	//写入文件
	err = SaveSetting(cluster)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    201,
			"message": response("importSuccessSaveFail", langStr),
			"data":    nil,
		})
		return
	}
	//写入 save/ 和 backup/
	for worldName, dirPaths := range dstFiles {
		clusterFilePath := fmt.Sprintf("%s/%s", cluster.GetMainPath(), worldName)
		for _, dirPath := range dirPaths {
			cmd := fmt.Sprintf("cp -rf %s %s", dirPath, clusterFilePath)
			err = utils.BashCMD(cmd)
			if err != nil {
				utils.Logger.Error("复制游戏数据失败", "err", err, "dir", clusterFilePath)
				c.JSON(http.StatusOK, gin.H{
					"code":    201,
					"message": responseImportError("copyFileFail", langStr),
					"data":    nil,
				})
				return
			}
		}

	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("uploadSuccess", langStr), "data": nil})
}

func handlePlayerListGet(c *gin.Context) {
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

	type PlayersInfo struct {
		UID      string `json:"uid"`
		NickName string `json:"nickName"`
		Prefab   string `json:"prefab"`
		Age      int    `json:"age"`
	}
	type PlayerList struct {
		Players   []PlayersInfo          `json:"players"`
		AdminList []string               `json:"adminList"`
		BlockList []string               `json:"blockList"`
		WhiteList []string               `json:"whiteList"`
		UidMap    map[string]interface{} `json:"uidMap"`
	}

	adminListPath := cluster.GetAdminListFile()
	blockListPath := cluster.GetBlockListFile()
	whiteListPath := cluster.GetWhiteListFile()

	adminList := getList(adminListPath)
	blockList := getList(blockListPath)
	whiteList := getList(whiteListPath)

	uidMap, _ := utils.ReadUidMap(cluster)

	var (
		playList PlayerList
		players  []utils.Players
	)

	if len(utils.STATISTICS[cluster.ClusterSetting.ClusterName]) > 0 {
		players = utils.STATISTICS[cluster.ClusterSetting.ClusterName][len(utils.STATISTICS[cluster.ClusterSetting.ClusterName])-1].Players
	}

	for _, player := range players {
		uid := player.UID
		age, _, err := GetPlayerAgePrefab(uid, cluster)
		if err != nil {
			utils.Logger.Error("玩家游戏时长获取失败")
		}
		var playerInfo PlayersInfo
		playerInfo.UID = uid
		playerInfo.NickName = player.NickName
		playerInfo.Prefab = player.Prefab
		playerInfo.Age = age

		playList.Players = append(playList.Players, playerInfo)
	}

	playList.AdminList = adminList
	playList.BlockList = blockList
	playList.WhiteList = whiteList
	playList.UidMap = uidMap

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": playList})
}

func handlePlayerListChangePost(c *gin.Context) {
	type ReqForm struct {
		ClusterName string `json:"clusterName"`
		Uid         string `json:"uid"`
		Type        string `json:"type"`
		ListName    string `json:"listName"`
	}
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var (
		reqForm        ReqForm
		uidList        []string
		err            error
		messageSuccess string
		messageFail    string
	)
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

	switch reqForm.ListName {
	case "admin":
		if reqForm.Type == "add" {
			messageSuccess = "addAdmin"
			messageFail = "addAdminFail"
			uidList, err = utils.ReadLinesToSlice(cluster.GetAdminListFile())
			if err != nil {
				utils.Logger.Info("未获取到管理员名单，跳过", "err", err)
			}
			uidList = append(uidList, reqForm.Uid)
			err = utils.WriteLinesFromSlice(cluster.GetAdminListFile(), uidList)
			if err != nil {
				utils.Logger.Info("写入管理员名单失败", "err", err)
				c.JSON(http.StatusOK, gin.H{
					"code":    201,
					"message": response(messageFail, langStr),
					"data":    nil,
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"code":    200,
				"message": response(messageSuccess, langStr),
				"data":    nil,
			})
			return
		} else {
			messageSuccess = "deleteAdmin"
			messageFail = "deleteAdminFail"
			uidList, err = utils.ReadLinesToSlice(cluster.GetAdminListFile())
			if err != nil {
				utils.Logger.Info("未获取到管理员名单", "err", err)
				c.JSON(http.StatusOK, gin.H{
					"code":    201,
					"message": response(messageFail, langStr),
					"data":    nil,
				})
				return
			}
			// 删除指定行
			for i := 0; i < len(uidList); i++ {
				if uidList[i] == reqForm.Uid {
					uidList = append(uidList[:i], uidList[i+1:]...)
					break
				}
			}
			err = utils.WriteLinesFromSlice(cluster.GetAdminListFile(), uidList)
			if err != nil {
				utils.Logger.Info("写入管理员名单失败", "err", err)
				c.JSON(http.StatusOK, gin.H{
					"code":    201,
					"message": response(messageFail, langStr),
					"data":    nil,
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"code":    200,
				"message": response(messageSuccess, langStr),
				"data":    nil,
			})
			return
		}
	case "block":
		if reqForm.Type == "add" {
			messageSuccess = "addWhite"
			messageFail = "addWhiteFail"
			uidList, err = utils.ReadLinesToSlice(cluster.GetBlockListFile())
			if err != nil {
				utils.Logger.Info("未获取到黑名单，跳过", "err", err)
			}
			uidList = append(uidList, reqForm.Uid)
			err = utils.WriteLinesFromSlice(cluster.GetBlockListFile(), uidList)
			if err != nil {
				utils.Logger.Info("写入黑名单失败", "err", err)
				c.JSON(http.StatusOK, gin.H{
					"code":    201,
					"message": response(messageFail, langStr),
					"data":    nil,
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"code":    200,
				"message": response(messageSuccess, langStr),
				"data":    nil,
			})
			return
		} else {
			messageSuccess = "addWhite"
			messageFail = "addWhiteFail"
			uidList, err = utils.ReadLinesToSlice(cluster.GetBlockListFile())
			if err != nil {
				utils.Logger.Info("未获取到黑名单", "err", err)
				c.JSON(http.StatusOK, gin.H{
					"code":    201,
					"message": response(messageFail, langStr),
					"data":    nil,
				})
				return
			}
			// 删除指定行
			for i := 0; i < len(uidList); i++ {
				if uidList[i] == reqForm.Uid {
					uidList = append(uidList[:i], uidList[i+1:]...)
					break
				}
			}
			err = utils.WriteLinesFromSlice(cluster.GetBlockListFile(), uidList)
			if err != nil {
				utils.Logger.Info("写入黑名单失败", "err", err)
				c.JSON(http.StatusOK, gin.H{
					"code":    201,
					"message": response(messageFail, langStr),
					"data":    nil,
				})
				return
			}
			c.JSON(http.StatusOK, gin.H{
				"code":    200,
				"message": response(messageSuccess, langStr),
				"data":    nil,
			})
			return
		}
	case "white":
		if reqForm.Type == "add" {
			messageSuccess = "addWhite"
			messageFail = "addWhiteFail"
			uidList, err = utils.ReadLinesToSlice(cluster.GetWhiteListFile())
			if err != nil {
				utils.Logger.Info("未获取到白名单，跳过", "err", err)
			}
			uidList = append(uidList, reqForm.Uid)
			err = utils.WriteLinesFromSlice(cluster.GetWhiteListFile(), uidList)
			if err != nil {
				utils.Logger.Info("写入白名单失败", "err", err)
				c.JSON(http.StatusOK, gin.H{
					"code":    201,
					"message": response(messageFail, langStr),
					"data":    nil,
				})
				return
			}
		} else {
			messageSuccess = "deleteWhite"
			messageFail = "deleteWhiteFail"
			uidList, err = utils.ReadLinesToSlice(cluster.GetWhiteListFile())
			if err != nil {
				utils.Logger.Info("未获取到白名单", "err", err)
				c.JSON(http.StatusOK, gin.H{
					"code":    201,
					"message": response(messageFail, langStr),
					"data":    nil,
				})
				return
			}
			// 删除指定行
			for i := 0; i < len(uidList); i++ {
				if uidList[i] == reqForm.Uid {
					uidList = append(uidList[:i], uidList[i+1:]...)
					break
				}
			}
			err = utils.WriteLinesFromSlice(cluster.GetWhiteListFile(), uidList)
			if err != nil {
				utils.Logger.Info("写入白名单失败", "err", err)
				c.JSON(http.StatusOK, gin.H{
					"code":    201,
					"message": response(messageFail, langStr),
					"data":    nil,
				})
				return
			}
		}

		clusterIniFileContent := clusterTemplate(cluster)
		err = utils.TruncAndWriteFile(cluster.GetIniFile(), clusterIniFileContent)
		if err != nil {
			utils.Logger.Error("写入cluster.ini失败", "err", err)
			c.JSON(http.StatusOK, gin.H{
				"code":    201,
				"message": response(messageFail, langStr),
				"data":    nil,
			})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": response(messageSuccess, langStr),
			"data":    nil,
		})
		return
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
	}
}

func handleHistoryPlayerGet(c *gin.Context) {
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
	type Player struct {
		UID      string      `json:"uid"`
		Nickname interface{} `json:"nickname"`
		Prefab   string      `json:"prefab"`
		Age      int         `json:"age"`
	}

	uidMap, _ := utils.ReadUidMap(cluster)
	if len(uidMap) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": []Player{}})
		return
	}

	var playerList []Player
	for uid, nickname := range uidMap {
		age, prefab, err := GetPlayerAgePrefab(uid, cluster)
		if err != nil {
			utils.Logger.Error("获取历史玩家信息失败", "err", err, "UID", uid)
		}
		var player Player
		player.UID = uid
		player.Nickname = nickname
		player.Age = age
		player.Prefab = prefab
		playerList = append(playerList, player)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": playerList})
}

func handleHistoryPlayerCleanPost(c *gin.Context) {
	type ReqForm struct {
		ClusterName string `json:"clusterName"`
	}
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var (
		reqForm ReqForm
		err     error
	)
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

	err = utils.TruncAndWriteFile(cluster.GetUIDMapFile(), "{}")
	if err != nil {
		utils.Logger.Error("清空uid_map文件失败", "err", err)
		c.JSON(http.StatusOK, gin.H{
			"code":    201,
			"message": response("cleanHistoryPlayersFail", langStr),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": response("cleanHistoryPlayersSuccess", langStr),
		"data":    nil,
	})
}

func handleBlockUpload(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("uploadFail", langStr), "data": nil})
		return
	}
	clusterName := c.PostForm("clusterName")
	if clusterName == "" {
		c.JSON(http.StatusBadRequest, "缺少集群名")
		return
	}

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}
	cluster, err := config.GetClusterWithName(clusterName)
	if err != nil {
		utils.Logger.Error("获取集群失败", "err", err)
		utils.RespondWithError(c, 404, "zh")
		return
	}

	//保存文件
	savePath := utils.ImportFileUploadPath + file.Filename
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		utils.Logger.Error("文件保存失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("uploadFail", langStr), "data": nil})
		return
	}

	// 打开Excel文件
	xlsFile, err := xlsx.OpenFile(savePath)
	if err != nil {
		utils.Logger.Error("无法打开文件: %s", err)
	}

	blockList := getList(cluster.GetBlockListFile())

	// 遍历所有工作表
	for _, sheet := range xlsFile.Sheets {
		// 遍历工作表中的所有行
		for _, row := range sheet.Rows {
			// 获取A列（索引为0）的单元格
			if len(row.Cells) > 0 {
				cell := row.Cells[0]
				// 将单元格的值添加到字符串切片中
				blockList = append(blockList, cell.String())
			}
		}
	}

	blockList = utils.UniqueSliceKeepOrderString(blockList)
	err = utils.WriteLinesFromSlice(cluster.GetBlockListFile(), blockList)
	if err != nil {
		utils.Logger.Error("写入黑名单失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("uploadFail", langStr), "data": nil})
		return
	}
	_ = utils.BashCMD("rm -f " + savePath)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("uploadSuccess", langStr), "data": nil})
}

func handlePlayerAction(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	type ReqForm struct {
		ClusterName string `json:"clusterName"`
		Type        string `json:"type"`
		Uid         string `json:"uid"`
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

	var cmd string

	switch reqForm.Type {
	case "kick":
		cmd = cluster.GetKickCmd(reqForm.Uid)
	case "kill":
		cmd = cluster.GetKillCmd(reqForm.Uid)
	case "respawn":
		cmd = cluster.GetRespawnCmd(reqForm.Uid)
	case "despawn":
		cmd = cluster.GetDespawnCmd(reqForm.Uid)
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	for _, world := range cluster.Worlds {
		if world.IsMaster {
			err = utils.ScreenCMD(cmd, world.ScreenName)
			if err != nil {
				utils.Logger.Error("执行玩家操作命令失败", "err", err, "cmd", cmd)
				break
			} else {
				c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("executed", langStr), "data": nil})
				return
			}
		}
	}

	for _, world := range cluster.Worlds {
		err = utils.ScreenCMD(cmd, world.ScreenName)
		if err == nil {
			break
		}
	}

	if err != nil {
		utils.Logger.Error("执行玩家操作命令失败", "err", err, "cmd", cmd)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("executedFail", langStr), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("executed", langStr), "data": nil})
}

func handleModSettingFormatGet(c *gin.Context) {
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
		utils.RespondWithError(c, 500, "zh")
		return
	}

	cluster, err := config.GetClusterWithName(reqForm.ClusterName)
	if err != nil {
		utils.Logger.Error("获取集群失败", "err", err)
		utils.RespondWithError(c, 404, "zh")
		return
	}

	luaScript := cluster.Mod

	modInfo, netErr, ModOverridesToStructErr := externalApi.GetModsInfo(luaScript, langStr)
	if netErr != nil && ModOverridesToStructErr != nil {
		utils.RespondWithError(c, 500, langStr)
		return
	}

	var responseData []utils.ModFormattedData

	modStruct, err := utils.ModOverridesToStruct(luaScript)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("complicatedMod", langStr), "data": nil})
		return
	}

	for _, i := range modStruct {
		item := utils.ModFormattedData{
			ID: i.ID,
			Name: func() string {
				for _, j := range modInfo {
					if i.ID == j.ID {
						return j.Name
					}
				}
				return ""
			}(),
			Enable:               i.Enabled,
			ConfigurationOptions: i.ConfigurationOptions,
			FileUrl: func() string {
				for _, j := range modInfo {
					if i.ID == j.ID {
						return j.FileUrl
					}
				}
				return ""
			}(),
			PreviewUrl: func() string {
				for _, j := range modInfo {
					if i.ID == j.ID {
						return j.PreviewUrl
					}
				}
				return ""
			}(),
		}
		responseData = append(responseData, item)
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": responseData})
}

func handleModConfigOptionsGet(c *gin.Context) {
	type ModConfigurationsForm struct {
		ID          int    `form:"id" json:"id"`
		ClusterName string `json:"clusterName" form:"clusterName"`
	}
	var modConfigurationsForm ModConfigurationsForm
	if err := c.ShouldBindQuery(&modConfigurationsForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("读取配置文件失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	cluster, err := config.GetClusterWithName(modConfigurationsForm.ClusterName)
	if err != nil {
		utils.Logger.Error("获取集群失败", "err", err)
		utils.RespondWithError(c, 404, "zh")
		return
	}

	type ModConfig struct {
		ID            int                         `json:"id"`
		ConfigOptions []utils.ConfigurationOption `json:"configOptions"`
	}

	var (
		modConfig      ModConfig
		modInfoLuaFile string
	)

	modID := modConfigurationsForm.ID

	if modID == 1 {
		// 禁用客户端模组配置
		modConfig.ID = 1
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": modConfig})
		return
	}

	modInfoLuaFile = fmt.Sprintf("%s/%d/modinfo.lua", cluster.GetModUgcPath()[0], modID)

	isUgcMod, err := utils.FileDirectoryExists(modInfoLuaFile)
	if err != nil {
		utils.RespondWithError(c, 500, langStr)
		return
	}

	if !isUgcMod {
		modInfoLuaFile = fmt.Sprintf("%s/workshop-%d/modinfo.lua", cluster.GetModNoUgcPath(), modID)
		exist, err := utils.FileDirectoryExists(modInfoLuaFile)
		if err != nil {
			utils.RespondWithError(c, 500, langStr)
			return
		}
		if !exist {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("needDownload", langStr), "data": nil})
			return
		}
	}

	luaScript, _ := utils.GetFileAllContent(modInfoLuaFile)
	modConfig.ID = modID
	modConfig.ConfigOptions = utils.GetModConfigOptions(luaScript, langStr)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": modConfig})
}

func handleModConfigChangePost(c *gin.Context) {
	type ModFormattedDataForm struct {
		ModFormattedData []utils.ModFormattedData `json:"modFormattedData"`
		ClusterName      string                   `json:"clusterName"`
	}
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	var modFormattedDataForm ModFormattedDataForm
	if err := c.ShouldBindJSON(&modFormattedDataForm); err != nil {
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

	cluster, err := config.GetClusterWithName(modFormattedDataForm.ClusterName)
	if err != nil {
		utils.RespondWithError(c, 404, langStr)
		return
	}

	luaString, err := utils.ParseToLua(modFormattedDataForm.ModFormattedData)
	if err != nil {
		utils.Logger.Warn(err.Error())
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("complicatedMod", langStr), "data": nil})
		return
	}

	cluster.Mod = luaString

	// 保存
	err = SaveSetting(cluster)
	if err != nil {
		utils.Logger.Error("MOD配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("configUpdateSuccess", langStr), "data": nil})
}

func handleModDownloadPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	type ModDownloadForm struct {
		ID      int    `json:"id"`
		FileURL string `json:"file_url"`
	}
	var modDownloadForm ModDownloadForm
	if err := c.ShouldBindJSON(&modDownloadForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	go func() {
		if modDownloadForm.FileURL == "" {
			// 下载前删除
			dirPath := fmt.Sprintf("%s/steamapps/workshop/content/322330/%d", utils.ModDownloadPath, modDownloadForm.ID)
			err := utils.RemoveDir(dirPath)
			if err != nil {
				utils.Logger.Info("Mod目录删除失败", "err", err)
			}
			err = utils.RemoveFile(utils.ModUgcAcfFile)
			if err != nil {
				utils.Logger.Info("Acf文件删除失败", "err", err)
			}
			cmd := utils.GenerateModDownloadCMD(modDownloadForm.ID)
			err = utils.BashCMD(cmd)
			if err != nil {
				utils.Logger.Error("MOD下载失败", "err", err)
			}
		} else {
			// externalApi.DownloadMod 会先删除再下载
			err := externalApi.DownloadMod(modDownloadForm.FileURL, modDownloadForm.ID)
			if err != nil {
				utils.Logger.Error("MOD下载失败", "err", err)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("downloading", langStr), "data": nil})
}

func handleModDownloadProcessGet(c *gin.Context) {
	type ReqForm struct {
		ID   int    `json:"id" form:"id"`
		Size string `json:"size" form:"size"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	size, err := strconv.ParseInt(reqForm.Size, 10, 64)
	if err != nil {
		utils.Logger.Error("模组大小转换失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "fail", "data": -1})
		return
	}

	dirPath := fmt.Sprintf("%s/%d", utils.ModUgcDownloadPath, reqForm.ID)

	currentSize, err := utils.GetDirSize(dirPath)
	if err != nil {
		utils.Logger.Error("模组大小计算失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "fail", "data": -1})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "fail", "data": currentSize / size})
}

func handleSyncModPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	type ReqForm struct {
		ClusterName string `json:"clusterName"`
		Uid         string `json:"uid"`
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

	err = utils.SyncMods(cluster)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("syncModFail", langStr), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("syncModSuccess", langStr), "data": nil})
}

func handleDeleteDownloadedModPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	type DeleteForm struct {
		ISUGC bool `json:"isUgc"`
		ID    int  `json:"id"`
	}

	var deleteForm DeleteForm
	if err := c.ShouldBindJSON(&deleteForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := utils.DeleteDownloadedMod(deleteForm.ISUGC, deleteForm.ID)
	if err != nil {
		utils.Logger.Error("删除已下载的MOD失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("deleteModFail", langStr), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("deleteModSuccess", langStr), "data": nil})
}

func handleEnableModPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	type EnableForm struct {
		ISUGC       bool   `json:"isUgc"`
		ID          int    `json:"id"`
		ClusterName string `json:"clusterName"`
	}

	var enableForm EnableForm
	if err := c.ShouldBindJSON(&enableForm); err != nil {
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

	cluster, err := config.GetClusterWithName(enableForm.ClusterName)
	if err != nil {
		utils.RespondWithError(c, 404, langStr)
		return
	}

	// 读取modinfo.lua
	var (
		modInfoLuaFile   string
		modDirPath       string
		modFormattedData []utils.ModFormattedData
	)

	if len(cluster.Worlds) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("gameServerNotCreated", langStr), "data": []string{}})
		return
	}

	// 复制mod文件至指定的dst目录
	if enableForm.ISUGC {
		modDirPath = fmt.Sprintf("%s/%d", utils.ModUgcDownloadPath, enableForm.ID)
		modInfoLuaFile = modDirPath + "/modinfo.lua"
		// MacOS 不执行复制
		if utils.Platform != "darwin" {
			for _, world := range cluster.Worlds {
				dstModPath := world.GetDstModPath(cluster.ClusterSetting.ClusterName)
				_ = utils.EnsureDirExists(dstModPath)
				err = utils.RemoveDir(dstModPath + "/" + strconv.Itoa(enableForm.ID))
				if err != nil {
					utils.Logger.Warn("删除旧MOD文件失败", "err", err)
				}
				cmd := fmt.Sprintf("cp -rf %s %s/", modDirPath, dstModPath)
				err = utils.BashCMD(cmd)
				if err != nil {
					utils.Logger.Error("复制MOD文件失败", "err", err, "cmd", cmd)
				}
			}
		}
	} else {
		modDirPath = fmt.Sprintf("%s/%d", utils.ModNoUgcDownloadPath, enableForm.ID)
		modInfoLuaFile = modDirPath + "/modinfo.lua"
		// MacOS 不执行复制
		if utils.Platform != "darwin" {
			_ = utils.EnsureDirExists(cluster.GetModNoUgcPath())
			err = utils.RemoveDir(cluster.GetModNoUgcPath() + "/workshop-" + strconv.Itoa(enableForm.ID))
			if err != nil {
				utils.Logger.Warn("删除旧MOD文件失败", "err", err, "cmd", enableForm.ID)
			}
			cmd := fmt.Sprintf("cp -rf %s %s/workshop-%d", modDirPath, cluster.GetModNoUgcPath(), enableForm.ID)
			err = utils.BashCMD(cmd)
			if err != nil {
				utils.Logger.Error("复制MOD文件失败", "err", err, "cmd", cmd)
			}
		}
	}

	luaScript, _ := utils.GetFileAllContent(modInfoLuaFile)

	// 获取新modoverrides.lua
	modOverrides, err := utils.AddModDefaultConfig(luaScript, enableForm.ID, langStr, cluster)
	if err != nil {
		utils.Logger.Error("添加模组失败，可能是添加了含有复杂配置的模组", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("complicatedModAdd", langStr), "data": nil})
		return
	}

	for _, mod := range modOverrides {
		modFormattedData = append(modFormattedData, utils.ModFormattedData{
			ID:                   mod.ID,
			Enable:               mod.Enabled,
			ConfigurationOptions: mod.ConfigurationOptions,
		})
	}

	// 需要转一次json，否则会出现新mod的default变量无法添加
	a, _ := json.Marshal(modFormattedData)
	var b []utils.ModFormattedData
	_ = json.Unmarshal(a, &b)
	modOverridesLua, err := utils.ParseToLua(b)
	if err != nil {
		utils.Logger.Error("添加模组失败，可能是添加了含有复杂配置的模组", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("complicatedModAdd", langStr), "data": nil})
		return
	}

	// 写入数据库
	cluster.Mod = modOverridesLua
	err = SaveSetting(cluster)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("enableModSuccess", langStr), "data": nil})
}

func handleDisableModPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	type DisableForm struct {
		ISUGC       bool   `json:"isUgc"`
		ID          int    `json:"id"`
		ClusterName string `json:"clusterName"`
	}

	var disableForm DisableForm
	if err := c.ShouldBindJSON(&disableForm); err != nil {
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

	cluster, err := config.GetClusterWithName(disableForm.ClusterName)
	if err != nil {
		utils.RespondWithError(c, 404, langStr)
		return
	}

	// 读取modinfo.lua
	modOverrides, err := utils.ModOverridesToStruct(cluster.Mod)
	if err != nil {
		utils.Logger.Error("禁用模组失败，可能是添加了含有复杂配置的模组", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("complicatedModAdd", langStr), "data": nil})
		return
	}

	var newModOverrides []utils.ModOverrides
	for _, mod := range modOverrides {
		if mod.ID != disableForm.ID {
			newModOverrides = append(newModOverrides, mod)
		}
	}

	// 需要转一次json，否则会出现新mod的default变量无法添加
	a, _ := json.Marshal(newModOverrides)
	var b []utils.ModOverrides
	_ = json.Unmarshal(a, &b)
	var modFormattedData []utils.ModFormattedData
	for _, mod := range b {
		modFormattedData = append(modFormattedData, utils.ModFormattedData{
			ID:                   mod.ID,
			Enable:               mod.Enabled,
			ConfigurationOptions: mod.ConfigurationOptions,
		})
	}
	newModOverridesLua, err := utils.ParseToLua(modFormattedData)
	if err != nil {
		utils.Logger.Error("禁用模组失败，可能是添加了含有复杂配置的模组", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("complicatedModAdd", langStr), "data": nil})
		return
	}

	// 写入数据
	cluster.Mod = newModOverridesLua
	err = SaveSetting(cluster)
	if err != nil {
		utils.Logger.Error("文件写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("deleteModSuccess", langStr), "data": newModOverridesLua})
}

func handleMacOSModExportPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	err := utils.RemoveDir(utils.MacModExportPath)
	if err != nil {
		utils.Logger.Error("删除目录失败", "err", err, "dir", utils.MacModExportPath)
	}

	var cpCmds []string

	modPathUgc := utils.ModUgcDownloadPath
	modsUgc, err := utils.GetDirs(modPathUgc, false)
	if err != nil {
		utils.Logger.Error("无法获取已下载的UGC MOD目录", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	for _, i := range modsUgc {
		cmd := fmt.Sprintf("cp -rf %s/%s %s/workshop-%s", modPathUgc, i, utils.MacModExportPath, i)
		cpCmds = append(cpCmds, cmd)
	}

	modPathNotUgc := utils.ModNoUgcDownloadPath
	modsNotUgc, err := utils.GetDirs(modPathNotUgc, false)
	if err != nil {
		utils.Logger.Error("无法获取已下载的非UGC MOD目录", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	for _, i := range modsNotUgc {
		cmd := fmt.Sprintf("cp -rf %s/%s %s/workshop-%s", modPathNotUgc, i, utils.MacModExportPath, i)
		cpCmds = append(cpCmds, cmd)
	}

	err = utils.BashCMD("mkdir -p " + utils.MacModExportPath)
	if err != nil {
		utils.Logger.Error("创建mod导出目录失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	for _, cmd := range cpCmds {
		err = utils.BashCMD(cmd)
		if err != nil {
			utils.Logger.Error("复制mod失败", "err", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("exportSuccess", langStr), "data": nil})
}

func handleModUpdatePost(c *gin.Context) {
	// 同步阻塞接口，耗时较长
	// 执行逻辑为 删除mod下载目录的旧mod，下载新mod到下载目录，删除dst旧mod，复制下载目录的mod到dst的mod目录
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	type UpdateForm struct {
		ID          int    `json:"id"`
		ISUGC       bool   `json:"isUgc"`
		FileURL     string `json:"fileURL"`
		ClusterName string `json:"clusterName"`
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

	cluster, err := config.GetClusterWithName(updateForm.ClusterName)
	if err != nil {
		utils.RespondWithError(c, 404, langStr)
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		utils.Logger.Error("无法获取 home 目录", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	// 删除，非UGC会在下载前自动删除
	var modDirPath string
	if updateForm.ISUGC {
		modDirPath = fmt.Sprintf("%s/%d", utils.ModUgcDownloadPath, updateForm.ID)
	}

	err = utils.RemoveDir(modDirPath)
	if err != nil {
		utils.Logger.Warn("删除旧MOD失败")
	}

	// 下载
	if updateForm.ISUGC {
		cmd := utils.GenerateModDownloadCMD(updateForm.ID)
		err := utils.BashCMD(cmd)
		if err != nil {
			utils.Logger.Error("MOD下载失败，MOD更新终止", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("updateModFail", langStr), "data": nil})
			return
		}
	} else {
		err := externalApi.DownloadMod(updateForm.FileURL, updateForm.ID)
		if err != nil {
			utils.Logger.Error("MOD下载失败，MOD更新终止", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("updateModFail", langStr), "data": nil})
			return
		}
	}

	// 删除 dst mod，复制新 mod 文件
	if updateForm.ISUGC {
		for _, modPath := range cluster.GetModUgcPath() {
			//删除旧MOD
			err = utils.RemoveDir(modPath + "/" + strconv.Itoa(updateForm.ID))
			if err != nil {
				utils.Logger.Warn("删除旧MOD文件失败", "err", err)
			}
			cmd := fmt.Sprintf("cp -rf %s %s/", modDirPath, modPath)
			err = utils.BashCMD(cmd)
			if err != nil {
				utils.Logger.Error("复制MOD文件失败", "err", err, "cmd", cmd)
				c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("updateModFail", langStr), "data": nil})
				return
			}
		}
	} else {
		modDirPath = fmt.Sprintf("%s/%s/%d", homeDir, cluster.GetModNoUgcPath(), updateForm.ID)
		err = utils.RemoveDir(cluster.GetModNoUgcPath() + "/workshop-" + strconv.Itoa(updateForm.ID))
		if err != nil {
			utils.Logger.Warn("删除旧MOD文件失败", "err", err)
		}
		cmd := fmt.Sprintf("cp -rf %s %s/workshop-%d", modDirPath, cluster.GetModNoUgcPath(), updateForm.ID)
		err = utils.BashCMD(cmd)
		if err != nil {
			utils.Logger.Error("复制MOD文件失败", "err", err, "cmd", cmd)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("updateModFail", langStr), "data": nil})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("updateModSuccess", langStr), "data": nil})

}

func handleAddClientModsDisabledConfig(c *gin.Context) {
	type ReqForm struct {
		ClusterName string `json:"clusterName"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
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

	cluster, err := config.GetClusterWithName(reqForm.ClusterName)
	if err != nil {
		utils.RespondWithError(c, 404, langStr)
		return
	}

	if cluster.ClusterSetting.Name == "" {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("gameServerNotCreated", langStr), "data": nil})
		return
	}

	//预防配置中 return { ... } 大括号不换行的情况
	re := regexp.MustCompile(`return\s*{`)
	text := re.ReplaceAllString(cluster.Mod, "return {\n")
	modFileLines := strings.Split(text, "\n")
	modFileLines[0] += "\n  client_mods_disabled={configuration_options={}, enabled=true},\n"
	cluster.Mod = strings.Join(modFileLines, "\n")

	err = SaveSetting(cluster)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("enableModSuccess", langStr), "data": nil})
}

func handleDeleteClientModsDisabledConfig(c *gin.Context) {
	type ReqForm struct {
		ClusterName string `json:"clusterName"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
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

	cluster, err := config.GetClusterWithName(reqForm.ClusterName)
	if err != nil {
		utils.RespondWithError(c, 404, langStr)
		return
	}

	if cluster.ClusterSetting.Name == "" {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("gameServerNotCreated", langStr), "data": nil})
		return
	}

	// 定义正则表达式来匹配目标内容
	re := regexp.MustCompile(`\s*client_mods_disabled=\s*\{(\s*configuration_options=\s*\{(\s*)*},?\s*enabled=true\s*)},?`)

	luaScript := cluster.Mod
	luaScript = re.ReplaceAllString(luaScript, "")
	cluster.Mod = luaScript

	err = SaveSetting(cluster)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("disableModSuccess", langStr), "data": nil})
}

func handleSystemSettingGet(c *gin.Context) {
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

	var systemResponse System

	systemResponse.SysSetting = cluster.SysSetting
	systemResponse.SchedulerSetting = config.SchedulerSetting

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": systemResponse})
}

func handleSystemSettingPut(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	type ReqForm struct {
		Settings    System `json:"settings"`
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

	var (
		bit64Changed    bool
		tickRateChanged bool
	)
	if cluster.SysSetting.Bit64 != reqForm.Settings.SysSetting.Bit64 {
		bit64Changed = true
	}
	if cluster.SysSetting.TickRate != reqForm.Settings.SysSetting.TickRate {
		tickRateChanged = true
	}

	cluster.SysSetting = reqForm.Settings.SysSetting
	config.SchedulerSetting = reqForm.Settings.SchedulerSetting

	for index, dbCluster := range config.Clusters {
		if dbCluster.ClusterSetting.ClusterName == cluster.ClusterSetting.ClusterName {
			config.Clusters[index] = cluster
			break
		}
	}

	if tickRateChanged {
		if cluster.Worlds != nil {
			err = SaveSetting(cluster)
			if err != nil {
				utils.Logger.Error("设置Tick Rate失败", "err", err)
			}
		}
	}

	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("写入文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	if bit64Changed {
		if cluster.SysSetting.Bit64 {
			// 安装64位依赖
			go utils.ExecBashScript("tmp.sh", utils.Install64Dependency)
		} else {
			// 安装32位依赖
			go utils.ExecBashScript("tmp.sh", utils.Install32Dependency)
		}
	}

	// 重新载入定时任务
	defer func() {
		scheduler.ReloadScheduler()
	}()

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("configUpdateSuccess", langStr), "data": nil})
}
