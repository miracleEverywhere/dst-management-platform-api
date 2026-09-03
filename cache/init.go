package cache

// InitCache 初始化缓存
func InitCache() {
	initCurrentDir()
	initOsType()
	initModDownloadStatus()
}
