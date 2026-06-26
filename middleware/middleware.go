package middleware

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

func TokenCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Request.Header.Get("X-DMP-TOKEN")
		claims, err := utils.ValidateJWT(token, []byte(db.JwtSecret))
		if err != nil {
			logger.Logger.Warnf("未授权的访问, DMP已拦截, ip为: %s", c.ClientIP())
			c.JSON(http.StatusOK, gin.H{"code": 420, "message": utils.I18n.Get(c, "token fail"), "data": nil})
			c.Abort()
			return
		}

		// 校验 token 版本号，检查是否已被撤销
		if !db.ValidateTokenVersion(claims.Username, claims.TokenVersion) {
			logger.Logger.Warnf("token已被撤销, username: %s, ip: %s", claims.Username, c.ClientIP())
			c.JSON(http.StatusOK, gin.H{"code": 420, "message": utils.I18n.Get(c, "token revoked"), "data": nil})
			c.Abort()
			return
		}

		c.Set("username", claims.Username)
		c.Set("nickname", claims.Nickname)
		c.Set("role", claims.Role)

		// token还有1/2有效期时，刷新token
		if shouldRefreshToken(claims.ExpiresAt.Time) {
			logger.Logger.Info("token有效期小于阈值，刷新token")
			user := models.User{
				Username:     claims.Username,
				Nickname:     claims.Nickname,
				Role:         claims.Role,
				TokenVersion: claims.TokenVersion,
			}
			token, err = utils.GenerateJWT(user, []byte(db.JwtSecret), utils.JwtExpirationHours)
			if err != nil {
				logger.Logger.Errorf("刷新Token失败：%v", err)
			} else {
				c.Header("X-DMP-NEW-TOKEN", token)
			}
		}

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
		logger.Logger.Warnf("越权请求, ip: %v, user: %v, nickname: %v", c.ClientIP(), username, nickname)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": utils.I18n.Get(c, "permission needed"), "data": nil})
		c.Abort()
		return
	}
}

// CacheControl 缓存控制中间件
func CacheControl() gin.HandlerFunc {
	cacheDuration := 48 * time.Hour
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

var loginRateLimiter = &loginRateLimitCache{}

type loginRateLimitCache struct {
	mu          sync.Mutex
	items       map[string]time.Time
	lastCleanup time.Time
}

// LoginRateLimit 登录接口限速，同一IP 1秒内只能请求一次
func LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		now := time.Now()

		loginRateLimiter.mu.Lock()

		// 定期清理过期条目，防止内存泄漏
		if loginRateLimiter.lastCleanup.Add(5 * time.Minute).Before(now) {
			for k, v := range loginRateLimiter.items {
				if now.Sub(v) > time.Second {
					delete(loginRateLimiter.items, k)
				}
			}
			loginRateLimiter.lastCleanup = now
		}

		lastTime, exists := loginRateLimiter.items[ip]
		if exists && now.Sub(lastTime) < time.Second {
			loginRateLimiter.mu.Unlock()
			logger.Logger.Warnf("登录频率过高, IP: %s", ip)
			c.JSON(http.StatusOK, gin.H{"code": 429, "message": utils.I18n.Get(c, "too many requests"), "data": nil})
			c.Abort()
			return
		}
		if loginRateLimiter.items == nil {
			loginRateLimiter.items = make(map[string]time.Time)
		}
		loginRateLimiter.items[ip] = now
		loginRateLimiter.mu.Unlock()

		c.Next()
	}
}

// 判断是否刷新token
func shouldRefreshToken(exp time.Time) bool {
	remainingTime := time.Until(exp)

	logger.Logger.Debugf("token剩余有效时间还剩: %.2f小时", remainingTime.Hours())

	totalDuration := time.Duration(utils.JwtExpirationHours) * time.Hour

	// 当剩余时间小于总有效期的 1/2 时刷新
	return remainingTime > 0 && remainingTime < totalDuration/2
}
