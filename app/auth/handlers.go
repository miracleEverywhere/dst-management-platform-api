package auth

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

type LoginForm struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type JsonBody struct {
	LoginForm LoginForm `json:"loginForm"`
}

func handleLogin(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var loginForm JsonBody
	if err := c.ShouldBindJSON(&loginForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config, _ := utils.ReadConfig()
	// 校验用户名和密码
	if loginForm.LoginForm.Username != config.Username {
		utils.RespondWithError(c, 420, langStr)
		return
	}
	if loginForm.LoginForm.Password != config.Password {
		utils.RespondWithError(c, 421, langStr)
		return
	}

	jwtSecret := []byte(config.JwtSecret)
	token, _ := utils.GenerateJWT(config.Username, jwtSecret, 12)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "data": gin.H{"token": token}})
}

func handleUserinfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": gin.H{"username": "admin"}})
}
