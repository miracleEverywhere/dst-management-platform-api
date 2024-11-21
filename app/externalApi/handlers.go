package externalApi

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
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

	connectionCode := "c_connect('" + internetIp + "',11000)"
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
	modInfoList, err := getModsInfo(config.RoomSetting.Mod)
	if err != nil {
		utils.Logger.Error("获取mod信息失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("getModInfoFail", langStr), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": modInfoList})
}
