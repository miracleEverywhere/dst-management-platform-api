package utils

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func exceptions(code int, lang string) string {
	exceptionsZH := map[int]string{
		420: "Token认证失败",
		421: "用户不存在",
		422: "密码错误",
		500: "服务器内部错误",
		510: "获取主机信息失败",
		511: "执行命令失败",
	}
	exceptionsEN := map[int]string{
		420: "Token Auth Fail",
		421: "User Not Exist",
		422: "Incorrect password",
		500: "Internal server error",
		510: "Failed to retrieve host information",
		511: "Failed to execute command",
	}

	if lang == "zh" {
		return exceptionsZH[code]
	} else {
		return exceptionsEN[code]
	}
}

func RespondWithError(c *gin.Context, code int, lang string) {
	message := exceptions(code, lang)
	c.JSON(http.StatusOK, gin.H{"code": code, "message": message})
}
