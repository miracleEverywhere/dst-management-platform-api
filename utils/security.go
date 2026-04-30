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
