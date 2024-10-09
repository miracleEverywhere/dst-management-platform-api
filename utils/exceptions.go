package utils

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func Exceptions(code int, lang string) string {
	exceptionsZH := map[int]string{
		420: "Token认证失败",
		421: "用户不存在",
		422: "密码错误",
		510: "获取主机信息失败",
	}
	exceptionsEN := map[int]string{
		420: "Token Auth Fail",
		421: "User Not Exist",
		422: "Incorrect password",
		510: "Failed to retrieve host information",
	}

	if lang == "zh" {
		return exceptionsZH[code]
	} else {
		return exceptionsEN[code]
	}
}

func RespondWithError(c *gin.Context, code int, lang string) {
	message := Exceptions(code, lang)
	c.JSON(http.StatusOK, gin.H{"code": code, "message": message})
}
