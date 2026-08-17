package middleware

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

// TokenCheck 从 header(X-DMP-TOKEN) query(token) 中获取jwt并进行校验
func TokenCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		var (
			err      error
			hasToken bool
		)

		// 从header中获取
		token := c.Request.Header.Get("X-DMP-TOKEN")

		if token == "" {
			token = c.Query("token")
			if token != "" {
				hasToken = true
			}
		} else {
			hasToken = true
		}

		if !hasToken {
			tokenMissing(c)
			return
		}

		claims, err := utils.ValidateJWT(token, []byte(cache.JwtSecret))
		if err != nil {
			tokenMissing(c)
			return
		}

		// 校验 token 版本号，检查是否已被撤销
		if !cache.ValidateTokenVersion(claims.Username, claims.TokenVersion) {
			tokenRevoked(c, *claims)
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
			token, err = utils.GenerateJWT(user, []byte(cache.JwtSecret), utils.JwtExpirationHours)
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
		username := c.GetString("username")
		if username == "" {
			username = "获取失败"
		}
		nickname := c.GetString("nickname")
		if nickname == "" {
			nickname = "获取失败"
		}

		tokenNoPermission(c, username, nickname)
		return
	}
}

// CacheControl 缓存控制中间件
func CacheControl() gin.HandlerFunc {
	cacheDuration := utils.StaticCacheHours * time.Hour
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
			tooManyRequests(c, ip)
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
