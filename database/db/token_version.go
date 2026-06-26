package db

// SetTokenVersion 将用户的 token 版本号写入缓存（登录时调用）
func SetTokenVersion(username string, version int) {
	TokenVersionCacheLock.Lock()
	defer TokenVersionCacheLock.Unlock()
	TokenVersionCache[username] = version
}

// ValidateTokenVersion 校验 token 中的版本号是否与缓存一致。
// 返回 true 表示 token 有效，false 表示已被撤销。
func ValidateTokenVersion(username string, tokenVersion int) bool {
	TokenVersionCacheLock.RLock()
	cachedVersion, exists := TokenVersionCache[username]
	TokenVersionCacheLock.RUnlock()

	if !exists {
		// 缓存中不存在：说明是当前进程签发后首次请求，缓存该版本号
		TokenVersionCacheLock.Lock()
		TokenVersionCache[username] = tokenVersion
		TokenVersionCacheLock.Unlock()
		return true
	}

	return cachedVersion == tokenVersion
}

// RevokeTokenVersion 递增用户的 token 版本号，使所有旧 token 失效。
// 返回新的版本号，调用方负责持久化到数据库。
func RevokeTokenVersion(username string, currentVersion int) int {
	newVersion := currentVersion + 1
	TokenVersionCacheLock.Lock()
	TokenVersionCache[username] = newVersion
	TokenVersionCacheLock.Unlock()
	return newVersion
}
