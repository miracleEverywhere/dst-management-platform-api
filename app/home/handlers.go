package home

import (
	"dst-management-platform-api/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func handleRoomInfoGet(c *gin.Context) {
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

	type Data struct {
		ClusterSetting utils.ClusterSetting `json:"clusterSetting"`
		SeasonInfo     metaInfo             `json:"seasonInfo"`
		ModsCount      int                  `json:"modsCount"`
		Players        []string             `json:"players"`
	}
	type Response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    Data   `json:"data"`
	}

	modsCount, err := countMods(cluster.Mod)
	if err != nil {
		utils.Logger.Error("读取mod数量失败", "err", err)
	}

	var (
		filePath   string
		sessionErr error
		seasonInfo metaInfo
		players    []string
	)
	for _, world := range cluster.Worlds {
		sessionPath := world.GetSessionPath(cluster.ClusterSetting.ClusterName)
		filePath, sessionErr = FindLatestMetaFile(sessionPath)
		if sessionErr == nil {
			break
		}
	}

	if sessionErr != nil {
		seasonInfo, _ = getMetaInfo("")
		utils.Logger.Error("查询session-meta文件失败", "err", sessionErr)
	} else {
		seasonInfo, err = getMetaInfo(filePath)
		if err != nil {
			utils.Logger.Error("获取meta文件内容失败", "err", err)
		}
	}

	if len(utils.STATISTICS[cluster.ClusterSetting.ClusterName]) > 0 {
		Players := utils.STATISTICS[cluster.ClusterSetting.ClusterName][len(utils.STATISTICS[cluster.ClusterSetting.ClusterName])-1].Players
		for _, player := range Players {
			players = append(players, player.NickName)
		}
	}

	data := Data{
		ClusterSetting: cluster.ClusterSetting,
		SeasonInfo:     seasonInfo,
		ModsCount:      modsCount,
		Players:        players,
	}

	response := Response{
		Code:    200,
		Message: "success",
		Data:    data,
	}

	c.JSON(http.StatusOK, response)
}

func handleSystemInfoGet(c *gin.Context) {
	type Data struct {
		Cpu    float64 `json:"cpu"`
		Memory float64 `json:"memory"`
	}
	type Response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    Data   `json:"data"`
	}

	var err error
	var SysInfoResponse Response
	SysInfoResponse.Code = 200
	SysInfoResponse.Message = "success"
	SysInfoResponse.Data.Cpu, err = utils.CpuUsage()
	if err != nil {
		utils.Logger.Error("获取Cpu使用率失败", "err", err)
	}
	SysInfoResponse.Data.Memory, err = utils.MemoryUsage()
	if err != nil {
		utils.Logger.Error("获取内存使用率失败", "err", err)
	}

	c.JSON(http.StatusOK, SysInfoResponse)
}

func handleWorldInfoGet(c *gin.Context) {
	type ReqForm struct {
		ClusterName string `json:"clusterName" form:"clusterName"`
	}
	type WorldStat struct {
		ID       int     `json:"id"`
		Stat     bool    `json:"stat"`
		World    string  `json:"world"`
		IsMaster bool    `json:"isMaster"`
		Type     string  `json:"type"`
		Cpu      float64 `json:"cpu"`
		Mem      float64 `json:"mem"`
		MemSize  float64 `json:"memSize"`
		DiskUsed int64   `json:"diskUsed"`
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

	var worldInfo []WorldStat

	for _, world := range cluster.Worlds {
		stat, cpu, mem, memSize, diskUsed := world.GetProcessStatus(cluster.ClusterSetting.ClusterName)

		status := WorldStat{
			ID:       world.ID,
			Stat:     stat,
			World:    world.Name,
			IsMaster: world.IsMaster,
			Type:     world.GetWorldType(),
			Cpu:      cpu,
			Mem:      mem,
			MemSize:  memSize,
			DiskUsed: diskUsed,
		}

		worldInfo = append(worldInfo, status)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": worldInfo})
}

func handleExecPost(c *gin.Context) {
	type ReqForm struct {
		ClusterName string      `json:"clusterName"`
		WorldName   string      `json:"worldName"`
		Type        string      `json:"type"`
		ExtraData   interface{} `json:"extraData"`
	}
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
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

	switch reqForm.Type {
	case "switch":
		world, err := config.GetWorldWithName(reqForm.ClusterName, reqForm.WorldName)
		if err != nil {
			utils.RespondWithError(c, 404, langStr)
			return
		}
		if world.GetStatus() {
			_ = world.StopGame()
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("shutdownSuccess", langStr), "data": nil})
			return
		} else {
			err = world.StartGame(reqForm.ClusterName, cluster.Mod, cluster.SysSetting.Bit64)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("startupFail", langStr), "data": nil})
				return
			}
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("startupSuccess", langStr), "data": nil})
			return
		}
	case "startup":
		defer func() {
			time.Sleep(10 * time.Second)
			_ = utils.BashCMD("screen -wipe")
		}()
		err = utils.StartClusterAllWorlds(cluster)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("startupFail", langStr), "data": nil})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("startupSuccess", langStr), "data": nil})
		return
	case "shutdown":
		defer func() {
			time.Sleep(10 * time.Second)
			_ = utils.BashCMD("screen -wipe")
		}()
		err = utils.StopClusterAllWorlds(cluster)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("shutdownFail", langStr), "data": nil})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("shutdownSuccess", langStr), "data": nil})
		return
	case "update":
		go func() {
			utils.DstUpdating = true
			_ = utils.StopAllClusters(config.Clusters)
			_ = utils.BashCMD(utils.GetDSTUpdateCmd())
			for _, _cluster := range config.Clusters {
				if !_cluster.ClusterSetting.Status {
					continue
				}
				_ = utils.StartClusterAllWorlds(_cluster)
			}
			utils.DstUpdating = false
		}()
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("updating", langStr), "data": nil})
		return
	case "restart":
		_ = utils.StopClusterAllWorlds(cluster)
		err = utils.StartClusterAllWorlds(cluster)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("restartFail", langStr), "data": nil})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("restartSuccess", langStr), "data": nil})
		return
	case "rollback":
		days := func() string {
			// 只存在float64的情况
			switch v := reqForm.ExtraData.(type) {
			case float64:
				return fmt.Sprintf("%d", int64(v))
			default:
				return ""
			}
		}()
		cmd := fmt.Sprintf("c_rollback(%s)", days)
		for _, world := range cluster.Worlds {
			if world.GetStatus() {
				err = utils.ScreenCMD(cmd, world.ScreenName)
				if err != nil {
					utils.Logger.Error("回档命令执行失败，尝试下一个世界", "err", err, "world", world.Name)
					continue
				}
				c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("rollbackSuccess", langStr), "data": nil})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("rollbackFail", langStr), "data": nil})
		return
	case "reset":
		cmd := "c_regenerateworld()"
		for _, world := range cluster.Worlds {
			if world.GetStatus() {
				err = utils.ScreenCMD(cmd, world.ScreenName)
				if err != nil {
					utils.Logger.Error("重置世界命令执行失败，尝试下一个世界", "err", err, "world", world.Name)
					continue
				}
				c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("resetSuccess", langStr), "data": nil})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("resetFail", langStr), "data": nil})
		return
	case "delete":
		_ = utils.StopClusterAllWorlds(cluster)
		for _, world := range cluster.Worlds {
			err = utils.RemoveDir(world.GetSavePath(cluster.ClusterSetting.ClusterName))
			if err != nil {
				utils.Logger.Error("删除世界失败，尝试下一个世界", "err", err, "world", world.Name)
				continue
			}
		}
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("deleteFail", langStr), "data": nil})
			return
		} else {
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("deleteSuccess", langStr), "data": nil})
			return
		}
	case "announce":
		message := func() string {
			// 只存在string的情况
			switch v := reqForm.ExtraData.(type) {
			case string:
				return v
			default:
				return ""
			}
		}()
		cmd := fmt.Sprintf("c_announce('%s')", message)
		for _, world := range cluster.Worlds {
			if world.GetStatus() {
				err = utils.ScreenCMD(cmd, world.ScreenName)
				if err != nil {
					utils.Logger.Error("公告命令执行失败，尝试下一个世界", "err", err, "world", world.Name)
					continue
				}
				c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("announceSuccess", langStr), "data": nil})
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("announceFail", langStr), "data": nil})
		return
	case "console":
		cmd := func() string {
			// 只存在string的情况
			switch v := reqForm.ExtraData.(type) {
			case string:
				return v
			default:
				return ""
			}
		}()
		world, err := config.GetWorldWithName(reqForm.ClusterName, reqForm.WorldName)
		if err != nil {
			utils.RespondWithError(c, 404, langStr)
			return
		}
		err = utils.ScreenCMD(cmd, world.ScreenName)
		if err != nil {
			utils.Logger.Error("console命令执行失败", "err", err, "world", world.Name)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("execFail", langStr), "data": nil})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("execSuccess", langStr), "data": nil})
		return
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
}

func handleGetClusterAllScreensGet(c *gin.Context) {
	type ReqForm struct {
		ClusterName string `json:"clusterName" form:"clusterName"`
	}

	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if reqForm.ClusterName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	screenNames := GetClusterScreens(reqForm.ClusterName)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": screenNames})
}

func handleKillScreenManualPost(c *gin.Context) {
	defer func() {
		time.Sleep(500 * time.Millisecond)
		_ = utils.BashCMD("screen -wipe")
	}()

	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	type ReqForm struct {
		ScreenName string `json:"screenName" form:"screenName"`
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

	killCMD := fmt.Sprintf("ps -ef | grep %s | grep -v grep | awk '{print $2}' | xargs kill -9", reqForm.ScreenName)
	err = utils.BashCMD(killCMD)
	if err != nil {
		utils.Logger.Error("执行命令失败", "err", err, "cmd", killCMD)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("shutdownFail", langStr), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("shutdownSuccess", langStr), "data": nil})
}

func handleGetIsUpdatingGet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": utils.DstUpdating})
}
