package middleware

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func MWtoken() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Request.Header.Get("X-DMP-TOKEN")
		claims, err := utils.ValidateJWT(token, []byte(db.JwtSecret))
		if err != nil {
			logger.Logger.Warn("token验证失败", "ip", c.ClientIP())
			c.JSON(http.StatusOK, gin.H{"code": 420, "message": utils.I18n.Get(c, "token fail"), "data": nil})
			c.Abort()
			return
		}
		c.Set("username", claims.Username)
		c.Set("nickname", claims.Nickname)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// MWAdminOnly 仅管理员接口
func MWAdminOnly() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exist := c.Get("role")
		if exist && role == "admin" {
			c.Next()
			return
		}
		username, exist := c.Get("username")
		if !exist {
			username = "获取失败"
		}
		nickname, exist := c.Get("nickname")
		if !exist {
			nickname = "获取失败"
		}
		logger.Logger.Warn("越权请求", "ip", c.ClientIP(), "user", username, "nickname", nickname)
		c.JSON(http.StatusOK, gin.H{"code": 420, "message": utils.I18n.Get(c, "permission needed"), "data": nil})
		c.Abort()
		return
	}
}
