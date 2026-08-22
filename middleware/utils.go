package middleware

import (
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

func tokenMissing(c *gin.Context) {
	logger.Logger.Warnf("未授权的访问, DMP已拦截, ip为: %s", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"code": 420, "message": utils.I18n.Get(c, "token fail"), "data": nil})
	c.Abort()
}

func tokenRevoked(c *gin.Context, claims utils.Claims) {
	logger.Logger.Warnf("token已被撤销, username: %s, ip: %s", claims.Username, c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"code": 420, "message": utils.I18n.Get(c, "token revoked"), "data": nil})
	c.Abort()
}

func tokenNoPermission(c *gin.Context, username, nickname string) {
	logger.Logger.Warnf("越权请求, ip: %v, user: %v, nickname: %v", c.ClientIP(), username, nickname)
	c.JSON(http.StatusOK, gin.H{"code": 201, "message": utils.I18n.Get(c, "permission needed"), "data": nil})
	c.Abort()
}

func tooManyRequests(c *gin.Context, ip string) {
	logger.Logger.Warnf("登录频率过高, IP: %s", ip)
	c.JSON(http.StatusOK, gin.H{"code": 429, "message": utils.I18n.Get(c, "too many requests"), "data": nil})
	c.Abort()
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

var loginRateLimiter = &loginRateLimitCache{}

type loginRateLimitCache struct {
	mu          sync.Mutex
	items       map[string]time.Time
	lastCleanup time.Time
}

// 判断是否刷新token
func shouldRefreshToken(exp time.Time) bool {
	remainingTime := time.Until(exp)

	//logger.Logger.Debugf("token剩余有效时间还剩: %.2f小时", remainingTime.Hours())

	totalDuration := time.Duration(utils.JwtExpirationHours) * time.Hour

	// 当剩余时间小于总有效期的 1/2 时刷新
	return remainingTime > 0 && remainingTime < totalDuration/2
}
