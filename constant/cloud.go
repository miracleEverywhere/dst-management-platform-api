package constant

import "time"

// DMP Cloud 对接配置。
// 云服务地址写死在编译期常量中，不在运行时接受用户输入，避免 SSRF 与配置错误。
const (
	// CloudAPIBaseURL 云服务 API 根地址
	CloudAPIBaseURL = "https://dmp-cloud.miraclesses.top/api/v1"
	// CloudHTTPTimeout 普通云 API 请求超时时间
	CloudHTTPTimeout = 30 * time.Second
	// CloudSyncTimeout 收藏同步请求超时时间
	CloudSyncTimeout = 90 * time.Second
	// CloudArchiveTimeout 云存档上传 / 下载超时时间
	CloudArchiveTimeout = 5 * time.Minute
	// CloudMaxSyncItems 单次同步最多提交的收藏数量
	CloudMaxSyncItems = 1000
	// CloudMaxResponseBytes 云 API JSON 响应读取上限
	CloudMaxResponseBytes = 64 << 20
	// CloudArchiveMaxBytes 单个云存档 zip 大小上限，与云服务保持一致
	CloudArchiveMaxBytes = 50 << 20
	// CloudKeepSnapshots 上传云存档时每个会话保留的最近快照数量
	CloudKeepSnapshots = 2
)

// 云功能本地文件路径。均位于 dmp_files 下，随平台数据目录一起备份。
const (
	// CloudKeyPath 云凭据 AES-GCM 密钥文件
	CloudKeyPath = DmpFiles + "/cloud.key"
	// CloudTempPath 云存档打包临时目录
	CloudTempPath = DmpFiles + "/cloud_tmp"
)
