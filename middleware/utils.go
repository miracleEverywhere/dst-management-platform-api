package middleware

import (
	"bytes"
	"crypto/sha256"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/i18n"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/gin-gonic/gin"
)

func tokenMissing(c *gin.Context) {
	logger.Logger.Warnf("未授权的访问, DMP已拦截, ip为: %s", c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"code": 420, "message": i18n.I18n.Get(c, "token fail"), "data": nil})
	c.Abort()
}

func tokenRevoked(c *gin.Context, claims utils.Claims) {
	logger.Logger.Warnf("token已被撤销, username: %s, ip: %s", claims.Username, c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"code": 420, "message": i18n.I18n.Get(c, "token revoked"), "data": nil})
	c.Abort()
}

func tokenNoPermission(c *gin.Context, username, nickname string) {
	logger.Logger.Warnf("越权请求, ip: %v, user: %v, nickname: %v", c.ClientIP(), username, nickname)
	c.JSON(http.StatusOK, gin.H{"code": 201, "message": i18n.I18n.Get(c, "permission needed"), "data": nil})
	c.Abort()
}

func tooManyRequests(c *gin.Context, username string) {
	logger.Logger.Warnf("登录频率过高, user: %s, ip: %s", truncateUsername(username), c.ClientIP())
	c.JSON(http.StatusOK, gin.H{"code": 429, "message": i18n.I18n.Get(c, "too many requests"), "data": nil})
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

var loginRateLimiter = &loginRateLimitCache{items: make(map[string]time.Time)}

type loginRateLimitCache struct {
	mu          sync.Mutex
	items       map[string]time.Time // key为用户名的sha256摘要，value为上次放行时间
	lastCleanup time.Time
}

// allow 判断该用户名本次请求是否放行，放行时记录时间戳
func (m *loginRateLimitCache) allow(username string, now time.Time) bool {
	key := loginRateLimitKey(username)

	m.mu.Lock()
	defer m.mu.Unlock()

	// 定期清理，或在条目数达到上限时立即清理，防止内存泄漏
	if now.Sub(m.lastCleanup) >= constant.LoginRateLimitCleanupInterval || len(m.items) >= constant.LoginRateLimitMaxEntries {
		m.cleanup(now)
	}

	lastTime, exists := m.items[key]
	if exists && now.Sub(lastTime) < constant.LoginRateLimitInterval {
		return false
	}

	if !exists && len(m.items) >= constant.LoginRateLimitMaxEntries {
		// 条目已满时放行，避免攻击者用随机用户名把正常用户锁在门外
		logger.Logger.Warnf("登录限流条目已达上限 %d，跳过记录", constant.LoginRateLimitMaxEntries)
		return true
	}

	m.items[key] = now
	return true
}

// cleanup 清理已失效的限流记录
func (m *loginRateLimitCache) cleanup(now time.Time) {
	for k, v := range m.items {
		if now.Sub(v) >= constant.LoginRateLimitInterval {
			delete(m.items, k)
		}
	}
	m.lastCleanup = now
}

// loginRateLimitKey 使用定长摘要作为key，避免超长用户名撑大内存
func loginRateLimitKey(username string) string {
	sum := sha256.Sum256([]byte(username))
	return hex.EncodeToString(sum[:])
}

// truncateUsername 截断并过滤日志中的用户名，避免污染日志
func truncateUsername(username string) string {
	const maxRunes = 64

	runes := []rune(username)
	if len(runes) > maxRunes {
		runes = append(runes[:maxRunes], []rune("...")...)
	}

	for i, r := range runes {
		if unicode.IsControl(r) {
			runes[i] = '?'
		}
	}

	return string(runes)
}

// 判断是否刷新token
func shouldRefreshToken(exp time.Time) bool {
	remainingTime := time.Until(exp)

	//logger.Logger.Debugf("token剩余有效时间还剩: %.2f小时", remainingTime.Hours())

	totalDuration := time.Duration(constant.JwtExpirationHours) * time.Hour

	// 当剩余时间小于总有效期的 1/2 时刷新
	return remainingTime > 0 && remainingTime < totalDuration/2
}

var errLoginBodyTooLarge = errors.New("login body too large")

// extractLoginUsername 读取并回填登录请求体，仅提取用户名
func extractLoginUsername(c *gin.Context) (string, error) {
	if c.Request.Body == nil {
		return "", io.EOF
	}

	body, err := io.ReadAll(io.LimitReader(c.Request.Body, constant.LoginBodyMaxBytes+1))
	if err != nil {
		return "", err
	}
	if len(body) > constant.LoginBodyMaxBytes {
		return "", errLoginBodyTooLarge
	}

	// 回填请求体，供 loginPost 的 ShouldBindJSON 再次读取
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	var form struct {
		Username string `json:"username"`
	}
	if err = json.Unmarshal(body, &form); err != nil {
		return "", err
	}

	return form.Username, nil
}
