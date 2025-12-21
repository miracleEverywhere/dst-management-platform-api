package utils

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func exceptions(code int, lang string) string {
	exceptionsZH := map[int]string{
		404: "集群资源不存在",
		420: "Token认证失败",
		421: "用户不存在",
		422: "密码错误",
		423: "该用户已被禁用",
		424: "旧密码错误",
		425: "非法请求",
		429: "请求过于频繁，请稍后再试",
		500: "服务器内部错误",
		510: "获取主机信息失败",
		511: "执行命令失败",
	}
	exceptionsEN := map[int]string{
		404: "Resources Not Found",
		420: "Token Auth Fail",
		421: "User Not Exist",
		422: "Incorrect password",
		423: "User is Not enabled",
		424: "Invalid old password",
		425: "Invalided request, No Permission",
		429: "Request rate limit exceeded. Please try again later",
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
	c.JSON(http.StatusOK, gin.H{"code": code, "message": message, "data": nil})
}
