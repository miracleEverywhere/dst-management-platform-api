package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"dst-management-platform-api/constant"
)

// SafeUserDirName 把用户名转换为可安全拼接进路径的目录名。
// 安全用户名保持原样；含非法字符（如斜杠、中文、纯点号）的用户名
// 使用用户名哈希生成确定性的安全目录名，避免目录穿越。
func SafeUserDirName(username string) string {
	if IsSafeString(username) {
		return username
	}

	sum := sha256.Sum256([]byte(username))
	return "u_" + hex.EncodeToString(sum[:16])
}

// UserModsDir 返回指定用户的模组缓存目录。
func UserModsDir(username string) string {
	return fmt.Sprintf("%s/%s", constant.UserModsPath, SafeUserDirName(username))
}
