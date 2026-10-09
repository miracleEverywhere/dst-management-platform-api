package utils

import (
	"net/url"
	"regexp"
	"strings"
)

var (
	IsSafeStringRe          = regexp.MustCompile(`^[a-zA-Z0-9_\-.]+$`)
	IsValidGameStartupCmdRe = regexp.MustCompile(`^[a-zA-Z0-9_\-./\\: ]+$`)
)

// IsSafeString 判断字符串是否安全，主要适用于命令拼接的字符串，包含worldName screenName等
// 除字符集外，额外排除纯点号（如 ".."、"..."）和以 "-" 开头的字符串，
// 避免拼接成路径时产生目录穿越、或被当成命令/程序参数
func IsSafeString(s string) bool {
	if !IsSafeStringRe.MatchString(s) {
		return false
	}
	if strings.Trim(s, ".") == "" {
		return false
	}

	return !strings.HasPrefix(s, "-")
}

// ShellQuote 将字符串包装成单引号形式的shell参数，用于必须拼接bash命令的场景，
// 防止字符串中的shell元字符被解释执行
func ShellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// IsValidGameMode 验证游戏模式字符串是否安全，防止XSS注入
// DST 游戏模式可自定义，此处校验字符集而非枚举固定值，杜绝 eval() 注入
func IsValidGameMode(mode string) bool {
	return IsSafeStringRe.MatchString(mode)
}

// IsValidGameStartupCmd 判断自定义启动命令是否安全
func IsValidGameStartupCmd(cmd string) bool {
	return IsValidGameStartupCmdRe.MatchString(cmd)
}

// IsValidURL 判断 URL 是否为合法的 webhook URL，防止 SSRF 攻击
// 合法的 webhook URL 必须满足：格式合法、协议为 http/https、不允许携带 query 参数、不允许携带fragment
func IsValidURL(rawURL string) bool {
	if rawURL == "" || len(rawURL) > 2048 {
		return false
	}

	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	// 只允许 HTTP/HTTPS 协议
	if u.Scheme != "http" && u.Scheme != "https" {
		return false
	}

	// 禁止携带 query 参数，防止通过 ?key=value 拼接恶意参数攻击内部端点
	if u.RawQuery != "" {
		return false
	}

	// 禁止携带 fragment，防止通过 #suffix 绕过校验
	if u.RawFragment != "" {
		return false
	}

	return true
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
