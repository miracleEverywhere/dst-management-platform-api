package utils

import (
	"github.com/gin-gonic/gin"
	"net/http"
)

func Exceptions(code int, lang string) string {
	exceptionsZH := map[int]string{
		410: "Token过期",
		411: "Token认证失败",
		412: "非法Token",
		413: "请先登录",
		420: "用户不存在",
		421: "密码错误",
	}
	exceptionsEN := map[int]string{
		410: "Token Expired",
		411: "Token Auth Fail",
		412: "Invalid Token",
		413: "Please Login First",
		420: "User Not Exist",
		421: "Incorrect password",
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
