package auth

import (
	"dst-management-platform-api/utils"
	"fmt"
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
	var loginForm JsonBody
	if err := c.ShouldBindJSON(&loginForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config, _ := utils.ReadConfig()
	if loginForm.LoginForm.Username != config.Username {
		fmt.Println(loginForm.LoginForm.Username)
		fmt.Println(config)
		c.JSON(http.StatusOK, gin.H{"code": 411, "message": "用户名错误"})
		return
	}
	if loginForm.LoginForm.Password != config.Password {
		c.JSON(http.StatusOK, gin.H{"code": 411, "message": "密码错误"})
		return
	}

	jwtSecret := []byte(config.JwtSecret)
	token, _ := utils.GenerateJWT(config.Username, jwtSecret, 12)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "data": gin.H{"token": token}})
}

func handleUserinfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": gin.H{"username": "admin"}})
}
