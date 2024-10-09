package tools

import (
	"dst-management-platform-api/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
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
