package utils

import (
	"regexp"
	"strings"
)

// IsSafeString 判断字符串是否安全，主要适用于命令拼接的字符串，包含worldName screenName等
func IsSafeString(s string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\-\.]+$`, s)

	return matched
}

// IsValidGameMode 验证游戏模式字符串是否安全，防止XSS注入
// DST 游戏模式可自定义，此处校验字符集而非枚举固定值，杜绝 eval() 注入
func IsValidGameMode(mode string) bool {
	if mode == "" {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\-\.]+$`, mode)
	return matched
}

// IsSafePath 文件名、路径是否安全 防止穿越攻击
func IsSafePath(path string) bool {
	forbiddenPatterns := []string{
		"..",
		"~",
	}

	for _, pattern := range forbiddenPatterns {
		if strings.Contains(path, pattern) {
			return false
		}
	}

	return true
}
