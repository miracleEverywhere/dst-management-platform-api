package externalApi

import (
	"dst-management-platform-api/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func handleVersionGet(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	dstVersion, err := GetDSTVersion()
	if err != nil {
		utils.Logger.Error("获取饥荒版本失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("getVersionFail", langStr), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": dstVersion})
}

func handleConnectionCodeGet(c *gin.Context) {
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
	if cluster.Worlds == nil {
		var cc string
		if langStr == "zh" {
			cc = "未发现可用的世界，无法获取直连代码"
		} else {
			cc = "No valid World found, can NOT generate connection code"
		}
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": cc, "data": nil})
		return
	}

	var (
		hasMaster bool
		port      int
	)
	for _, world := range cluster.Worlds {
		if world.IsMaster {
			hasMaster = true
			port = world.ServerPort
			break
		}
	}
	if !hasMaster {
		port = cluster.Worlds[0].ServerPort
	}

	var connectionCode string

	// 先查询数据库是否有自定义配置
	if cluster.CustomConnectCode.Ip != "" {
		if cluster.ClusterSetting.Password == "" {
			connectionCode = fmt.Sprintf("c_connect('%s', %d)", cluster.CustomConnectCode.Ip, cluster.CustomConnectCode.Port)
		} else {
			connectionCode = fmt.Sprintf("c_connect('%s', %d, '%s')", cluster.CustomConnectCode.Ip, cluster.CustomConnectCode.Port, cluster.ClusterSetting.Password)
		}
		// 直接返回自定义直连代码
		utils.Logger.Info("发现自定义直连代码，直接返回")
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": connectionCode})
		return
	}

	// 无自定义配置，查询数据库中有无缓存好的公网IP
	if config.InternetIp != "" {
		if cluster.ClusterSetting.Password == "" {
			connectionCode = fmt.Sprintf("c_connect('%s', %d)", config.InternetIp, port)
		} else {
			connectionCode = fmt.Sprintf("c_connect('%s', %d, '%s')", config.InternetIp, port, cluster.ClusterSetting.Password)
		}
		// 直接返回直连代码
		utils.Logger.Info("发现缓存的直连代码，直接返回")
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": connectionCode})
		return
	}

	// 无缓存好的公网IP，实时获取公网IP，并写入数据库
	internetIp, err := GetInternetIP1()
	if err != nil {
		utils.Logger.Warn("调用公网ip接口1失败", "err", err)
		internetIp, err = GetInternetIP2()
		if err != nil {
			utils.Logger.Warn("调用公网ip接口2失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("getConnectionCodeFail", langStr), "data": nil})
			return
		}
	}

	if cluster.ClusterSetting.Password == "" {
		connectionCode = fmt.Sprintf("c_connect('%s', %d)", internetIp, port)
	} else {
		connectionCode = fmt.Sprintf("c_connect('%s', %d, '%s')", internetIp, port, cluster.ClusterSetting.Password)
	}

	config.InternetIp = internetIp
	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("配置文件写入失败，无影响，记录异常", "err", err)
	}

	utils.Logger.Info("调用公网接口获取直连代码")

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": connectionCode})
}

func handleModInfoGet(c *gin.Context) {
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

	modInfoList, netErr, _ := GetModsInfo(cluster.Mod, langStr)
	if netErr != nil {
		utils.Logger.Error("获取mod信息失败", "err", netErr)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("getModInfoFail", langStr), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": modInfoList})
}

func handleModSearchGet(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	type SearchForm struct {
		SearchType string `form:"searchType" json:"searchType"`
		SearchText string `form:"searchText" json:"searchText"`
		Page       int    `form:"page" json:"page"`
		PageSize   int    `form:"pageSize" json:"pageSize"`
	}
	var searchForm SearchForm
	if err := c.ShouldBindQuery(&searchForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if searchForm.SearchType == "id" {
		id, err := strconv.Atoi(searchForm.SearchText)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("invalidModID", langStr), "data": nil})
			return
		}
		data, err := SearchModById(id, langStr)
		if err != nil {
			utils.Logger.Error("获取mod信息失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("getModInfoFail", langStr), "data": nil})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
		return
	}
	if searchForm.SearchType == "text" {
		data, err := SearchMod(searchForm.Page, searchForm.PageSize, searchForm.SearchText, langStr)
		if err != nil {
			utils.Logger.Error("获取mod信息失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("getModInfoFail", langStr), "data": nil})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
}

func handleDownloadedModInfoGet(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	modPathUgc := utils.ModUgcDownloadPath
	modsUgc, err := utils.GetDirs(modPathUgc, false)
	if err != nil {
		utils.Logger.Error("无法获取已下载的UGC MOD目录", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	modPathNotUgc := utils.ModNoUgcDownloadPath
	modsNotUgc, err := utils.GetDirs(modPathNotUgc, false)
	if err != nil {
		utils.Logger.Error("无法获取已下载的非UGC MOD目录", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	mods := append(modsNotUgc, modsUgc...)

	modInfo, err := GetDownloadedModInfo(mods, langStr)
	if err != nil {
		utils.RespondWithError(c, 500, langStr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": modInfo})
}

func handleLobbyCheckPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	type ReqForm struct {
		ClusterName string   `json:"clusterName"`
		Regions     []string `json:"regions"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
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

	var urls []string
	for _, region := range reqForm.Regions {
		urls = append(urls, utils.GetDSTRoomsApi(region))
	}

	rooms, err := CheckDstLobbyRoom(urls, cluster.ClusterSetting.Name)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("checkLobbyFail", langStr), "data": false})
		return
	}

	for _, room := range rooms {
		if room.MaxConnections == cluster.ClusterSetting.PlayerNum {
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": true})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": false})
}
