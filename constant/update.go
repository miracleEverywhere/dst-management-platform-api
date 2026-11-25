package constant

import "time"

// 平台自更新（网页端更新 DMP 自身）相关常量。
//
// 设计要点：
//   - 只信任 GitHub Release API 返回的 tag 与 digest，加速镜像仅用于下载资产；
//     镜像若也负责元数据，就等于把"下载什么"的决定权交给第三方，校验失去意义。
//   - 更新产物、数据库备份、更新记录、回滚状态统一放在工作目录的 DmpFiles/dmp_update 下，
//     不污染工作目录根部（根部只留可执行文件，备份与临时文件都在 dmp_files 下）。
const (
	// GitHubReleaseAPI 平台最新 Release 元数据接口
	GitHubReleaseAPI = "https://api.github.com/repos/miracleEverywhere/dst-management-platform-api/releases/latest"
	// GitHubReleaseAPI = "http://127.0.0.1:8080/latest"

	// UpdateDir 自更新工作目录（相对工作目录）
	UpdateDir = DmpFiles + "/dmp_update"

	// UpdateAssetLinux / UpdateAssetDarwin 发布流水线产出的资产名，
	// 见 .github/workflows/go.yml 的 build-linux / build-darwin
	UpdateAssetLinux  = "dmp.tgz"
	UpdateAssetDarwin = "dmp_darwin.tgz"

	// UpdateBinaryName 可执行文件名，同时也是发布包内的成员名
	UpdateBinaryName = "dmp"

	// UpdateBackupBinaryName 旧版本二进制在备份目录里的文件名。
	// 备份目录是 DmpFiles/dmp_update/backup/<runId>-<旧版本>-to-<新版本>/，
	// 与数据库备份同目录，按目录整体淘汰，避免出现"数据库还在、二进制被删"的错配。
	UpdateBackupBinaryName = "dmp"

	// UpdateFailedBinaryName 回滚时坏掉的新版本在同一个备份目录里的留存名
	UpdateFailedBinaryName = "dmp.failed"
)

const (
	// UpdateCheckTTL 版本检查结果缓存时长。GitHub 未认证请求限流为每小时 60 次/IP，
	// 前端每次进页面都会查询，必须缓存。
	UpdateCheckTTL = 10 * time.Minute

	// UpdateAPITimeout 查询 Release 元数据的超时时间
	UpdateAPITimeout = 30 * time.Second

	// UpdateDownloadTimeout 下载安装包的总超时
	UpdateDownloadTimeout = 20 * time.Minute

	// UpdateMaxAssetBytes 安装包体积上限，防止异常资产把磁盘写满
	UpdateMaxAssetBytes = 200 << 20

	// UpdatePreflightTimeout 新版本 `-v` 预检的超时时间
	UpdatePreflightTimeout = 15 * time.Second

	// UpdateHealthTimeout 回滚守护等待新进程上报健康信息的最长时间
	UpdateHealthTimeout = 120 * time.Second

	// UpdateDeadGrace 新进程已退出后再等多久才回滚，给 exec 与瞬时崩溃留缓冲
	UpdateDeadGrace = 3 * time.Second

	// UpdateKeepBackups 保留的数据库备份份数
	UpdateKeepBackups = 3

	// UpdateKeepRecords 保留的更新记录条数
	UpdateKeepRecords = 50

	// UpdateRestartFlushDelay 推送「即将重启」后等待前端收到事件的时间
	UpdateRestartFlushDelay = 600 * time.Millisecond
)
