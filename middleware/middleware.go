package middleware

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/i18n"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"errors"
	"fmt"
	"net/http"
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
			token, err = utils.GenerateJWT(user, []byte(cache.JwtSecret), constant.JwtExpirationHours)
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
	cacheDuration := constant.StaticCacheHours * time.Hour
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

// LoginRateLimit 登录接口限速，同一用户名 1 秒内只能请求一次。
// 以用户名为限流维度，避免伪造 X-Forwarded-For 绕过。
func LoginRateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		username, err := extractLoginUsername(c)

		switch {
		case errors.Is(err, errLoginBodyTooLarge):
			logger.Logger.Warnf("登录请求体过大, ip: %s", c.ClientIP())
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": i18n.I18n.Get(c, "bad request"), "data": nil})
			c.Abort()
			return
		case err != nil || username == "":
			// 无法提取用户名，交给 loginPost 统一返回 400，该路径不会查库，也不会执行密码校验
			c.Next()
			return
		}

		if !loginRateLimiter.allow(username, time.Now()) {
			tooManyRequests(c, username)
			return
		}

		c.Next()
	}
}
