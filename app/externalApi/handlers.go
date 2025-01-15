package externalApi

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"os"
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
	var (
		internetIp string
		err        error
	)
	internetIp, err = GetInternetIP1()
	if err != nil {
		utils.Logger.Warn("调用公网ip接口1失败", "err", err)
		internetIp, err = GetInternetIP2()
		if err != nil {
			utils.Logger.Warn("调用公网ip接口2失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("getConnectionCodeFail", langStr), "data": nil})
			return
		}
	}
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	var (
		connectionCode string
		port           int
	)

	if config.RoomSetting.Ground != "" {
		port = config.RoomSetting.Base.MasterPort
	} else {
		port = config.RoomSetting.Base.CavesPort
	}

	if config.RoomSetting.Base.Password != "" {
		connectionCode = "c_connect('" + internetIp + "', " + strconv.Itoa(port) + ", '" + config.RoomSetting.Base.Password + "')"
	} else {
		connectionCode = "c_connect('" + internetIp + "', " + strconv.Itoa(port) + ")"
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": connectionCode})
}

func handleModInfoGet(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("读取配置文件失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	modInfoList, err := GetModsInfo(config.RoomSetting.Mod, langStr)
	if err != nil {
		utils.Logger.Error("获取mod信息失败", "err", err)
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

	homeDir, err := os.UserHomeDir()
	if err != nil {
		utils.Logger.Error("无法获取 home 目录", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	modPathUgc := homeDir + "/" + utils.ModDownloadPath + "/steamapps/workshop/content/322330"
	modsUgc, err := utils.GetDirs(modPathUgc)
	if err != nil {
		utils.Logger.Error("无法获取已下载的UGC MOD目录", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	modPathNotUgc := homeDir + "/" + utils.ModDownloadPath + "/not_ugc"
	modsNotUgc, err := utils.GetDirs(modPathNotUgc)
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
