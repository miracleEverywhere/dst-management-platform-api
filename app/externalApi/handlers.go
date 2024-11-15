package externalApi

import (
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
		internetIp, err = GetInternetIP2()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("getConnectionCodeFail", langStr), "data": nil})
			return
		}
	}

	connectionCode := "c_connect('" + internetIp + "',11000)"
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": connectionCode})
}
