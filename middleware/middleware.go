package middleware

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func TokenCheck() gin.HandlerFunc {
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

// AdminOnly 仅管理员接口
func AdminOnly() gin.HandlerFunc {
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
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": utils.I18n.Get(c, "permission needed"), "data": nil})
		c.Abort()
		return
	}
}

// CacheControl 缓存控制中间件
func CacheControl() gin.HandlerFunc {
	cacheDuration := 24 * time.Hour
	return func(c *gin.Context) {
		// 只对静态资源文件设置缓存
		if isStaticAsset(c.Request.URL.Path) {
			// 设置缓存头
			c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", int(cacheDuration.Seconds())))

			// 可选：设置过期时间
			expires := time.Now().Add(cacheDuration).UTC().Format("Mon, 02 Jan 2006 15:04:05 GMT")
			c.Header("Expires", expires)
		}

		c.Next()
	}
}

// 判断是否为静态资源文件
func isStaticAsset(path string) bool {
	staticExtensions := []string{".js", ".css", ".jpg", ".jpeg", ".png", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot"}
	for _, ext := range staticExtensions {
		if strings.HasSuffix(path, ext) {
			return true
		}
	}
	return false
}
