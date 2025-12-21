package utils

import (
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"sync"
	"time"
)

func ValidateJWT(tokenString string, jwtSecret []byte) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		Logger.Warn("JWT验证失败")
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func MWlang() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := c.Request.Header.Get("X-I18n-Lang")
		c.Set("lang", lang)

		c.Next()
	}
}

func MWtoken() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Request.Header.Get("authorization")
		lang := c.Request.Header.Get("X-I18n-Lang")
		config, err := ReadConfig()
		if err != nil {
			Logger.Error("配置文件打开失败", "err", err)
			RespondWithError(c, 500, lang)
			c.Abort()
			return
		}
		tokenSecret := config.JwtSecret
		claims, err := ValidateJWT(token, []byte(tokenSecret))
		if err != nil {
			RespondWithError(c, 420, lang)
			c.Abort()
			return
		}
		c.Set("username", claims.Username)
		c.Set("nickname", claims.Nickname)
		c.Set("role", claims.Role)
		c.Next()
	}
}

// IPRateLimiter IP限流器结构体
type IPRateLimiter struct {
	ips           map[string]*requestInfo
	mutex         sync.RWMutex
	maxRequests   int           // 每个时间窗口内允许的最大请求数
	windowSize    time.Duration // 时间窗口大小
	cleanupTicker *time.Ticker  // 定期清理的ticker
}

type requestInfo struct {
	count       int       // 当前窗口内的请求计数
	windowStart time.Time // 当前窗口的开始时间
}

// NewIPRateLimiter 创建IP限流器实例
func NewIPRateLimiter(maxRequests int, windowSize time.Duration) *IPRateLimiter {
	limiter := &IPRateLimiter{
		ips:         make(map[string]*requestInfo),
		maxRequests: maxRequests,
		windowSize:  windowSize,
	}

	// 启动定期清理goroutine
	limiter.startCleanupRoutine()

	return limiter
}

// startCleanupRoutine 启动定期清理过期IP记录的goroutine
func (l *IPRateLimiter) startCleanupRoutine() {
	l.cleanupTicker = time.NewTicker(l.windowSize * 2) // 两倍窗口时间清理一次

	go func() {
		for range l.cleanupTicker.C {
			l.cleanupExpiredRecords()
		}
	}()
}

// cleanupExpiredRecords 清理过期的IP记录
func (l *IPRateLimiter) cleanupExpiredRecords() {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	now := time.Now()
	for ip, info := range l.ips {
		if now.Sub(info.windowStart) > l.windowSize*2 {
			delete(l.ips, ip)
		}
	}
}

// MWIPLimiter 返回Gin中间件函数
func (l *IPRateLimiter) MWIPLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()

		l.mutex.Lock()
		defer l.mutex.Unlock()

		// 获取或初始化该IP的记录
		info, exists := l.ips[ip]
		if !exists {
			l.ips[ip] = &requestInfo{
				count:       1,
				windowStart: time.Now(),
			}
			c.Next()
			return
		}

		// 检查是否在当前时间窗口内
		if time.Since(info.windowStart) > l.windowSize {
			// 新窗口，重置计数
			info.count = 1
			info.windowStart = time.Now()
			c.Next()
			return
		}

		// 增加计数并检查是否超过限制
		info.count++
		if info.count > l.maxRequests {
			lang, _ := c.Get("lang")
			langStr := "zh" // 默认语言
			if strLang, ok := lang.(string); ok {
				langStr = strLang
			}
			RespondWithError(c, 429, langStr)
			c.Abort()
			return
		}

		c.Next()
	}
}

// Stop 停止限流器的清理goroutine
func (l *IPRateLimiter) Stop() {
	if l.cleanupTicker != nil {
		l.cleanupTicker.Stop()
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

		Logger.Info("越权请求已中断")
		lang := c.Request.Header.Get("X-I18n-Lang")
		RespondWithError(c, 425, lang)
		c.Abort()
		return
	}
}

// MWUserCheck 用户状态检查
func MWUserCheck() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := c.Request.Header.Get("X-I18n-Lang")
		username, exist := c.Get("username")
		if exist {
			usernameStr, ok := username.(string)
			if ok {
				UserCacheMutex.Lock()
				user := UserCache[usernameStr]
				UserCacheMutex.Unlock()
				if len(user.Username) != 0 {
					if !user.Disabled {
						c.Next()
						return
					} else {
						RespondWithError(c, 423, lang)
						c.Abort()
						return
					}
				} else {
					RespondWithError(c, 421, lang)
					c.Abort()
					return
				}
			}

		}

		RespondWithError(c, 500, lang)
		c.Abort()
		return
	}
}

func MWDownloadToken() gin.HandlerFunc {
	return func(c *gin.Context) {
		// js window.open不允许携带header，采用params
		// 需要验证token和cluster
		queries := c.Request.URL.Query()
		authorization := queries["authorization"]
		lang := queries["lang"][0]

		if len(authorization) != 1 {
			RespondWithError(c, 425, lang)
			c.Abort()
			return
		}
		clusterNames := queries["clusterName"]
		if len(clusterNames) != 1 {
			RespondWithError(c, 425, lang)
			c.Abort()
			return
		}

		token := authorization[0]
		clusterName := clusterNames[0]

		config, err := ReadConfig()
		if err != nil {
			Logger.Error("配置文件打开失败", "err", err)
			RespondWithError(c, 500, lang)
			c.Abort()
			return
		}
		tokenSecret := config.JwtSecret
		claims, err := ValidateJWT(token, []byte(tokenSecret))
		if err != nil {
			RespondWithError(c, 420, lang)
			c.Abort()
			return
		}

		if claims.Role != "admin" {
			user := config.GetUserWithUsername(claims.Username)
			if !Contains(user.ClusterPermission, clusterName) {
				RespondWithError(c, 425, lang)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}
