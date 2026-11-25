package platform

import (
	"context"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// 自更新被禁用的原因，随版本检查结果一起返回给前端，由前端决定提示文案
const (
	// SelfUpdateReasonReady 可以自更新
	SelfUpdateReasonReady = ""
	// SelfUpdateReasonContainer 容器部署，应更新镜像
	SelfUpdateReasonContainer = "container"
	// SelfUpdateReasonUnsupported 运行环境不支持（系统、架构、启动方式）
	SelfUpdateReasonUnsupported = "unsupported"
	// SelfUpdateReasonNotWritable 程序所在目录不可写
	SelfUpdateReasonNotWritable = "not_writable"
	// SelfUpdateReasonLocalNewer 当前是开发版，比线上最新版本更新
	SelfUpdateReasonLocalNewer = "local_newer"
)

// 更新任务的事件类型，与 SSE 的 event 名一致
const (
	UpdateEventProgress = cache.UpdateEventProgress
	UpdateEventChunk    = cache.UpdateEventChunk
	UpdateEventDone     = cache.UpdateEventDone
)

// SelfUpdateEvent 推送给 SSE 的一条更新任务事件，Type 取 progress/chunk/done
type SelfUpdateEvent struct {
	Type string
	// ID 事件序号，可作为 Last-Event-ID 续传
	ID int64
	// Data 对应 cache.UpdateProgress / cache.UpdateChunk / cache.UpdateDone
	Data any
}

// VersionInfo 是平台版本检查结果
type VersionInfo struct {
	// Current 当前运行版本
	Current string `json:"current"`
	// Latest GitHub 最新 Release 的 tag
	Latest string `json:"latest"`
	// Notes Release 说明原文
	Notes string `json:"notes"`
	// CheckedAt 版本信息的时间戳（缓存命中时是上次成功查询的时间）
	CheckedAt int64 `json:"checkedAt"`
	// Updatable 线上是否有比当前更新的版本
	Updatable bool `json:"updatable"`
	// CanSelfUpdate 当前环境能否在网页上升级
	CanSelfUpdate bool `json:"canSelfUpdate"`
	// Reason 不能自更新的原因，取 SelfUpdateReason* 常量
	Reason string `json:"reason"`
	// Asset 当前系统对应的发布资产名
	Asset string `json:"asset"`
	// OS / Arch 运行环境，前端用于提示不支持的原因
	OS   string `json:"os"`
	Arch string `json:"arch"`
	// InContainer 是否运行在容器里
	InContainer bool `json:"inContainer"`
	// CheckFailed 查询 GitHub 失败，此时 Latest 为空，前端只能展示当前版本
	CheckFailed bool `json:"checkFailed"`
}

// Version 检查平台版本。
// 查询 GitHub 失败时返回错误码并带上已确定的本地信息，前端仍能展示当前版本。
func (s *Service) Version(refresh bool) service.Result {
	info := s.versionInfo(refresh)

	if info.CheckFailed {
		return service.FailData(constant.CodePlatformUpdateCheckFail, info)
	}
	if info.Updatable {
		return service.Msg("update available", info)
	}
	if info.Reason == SelfUpdateReasonLocalNewer {
		return service.Msg("update local newer", info)
	}

	return service.Msg("update up to date", info)
}

// SelfUpdateState 当前更新任务状态，供前端刷新页面后恢复界面
func (s *Service) SelfUpdateState() service.Result {
	state := cache.UpdateTask.State()
	state.From = firstNonEmpty(state.From, constant.Version)

	return service.Success(state)
}

// SelfUpdateStream 订阅更新进度：先补发仍保留的历史事件，再持续推送新增内容。
// 连接断开只解绑订阅，不影响更新任务本身。
func (s *Service) SelfUpdateStream(ctx context.Context, actor service.Actor, after int64) (<-chan SelfUpdateEvent, service.Result) {
	if after < 0 {
		after = 0
	}

	id, events := cache.UpdateTask.Subscribe(after)
	if events == nil {
		logger.Logger.Infof("没有可订阅的平台更新任务, api: %s", actor.Path)
		return nil, service.Fail(constant.CodePlatformUpdateNotRunning)
	}

	out := make(chan SelfUpdateEvent, 16)
	go func() {
		defer close(out)
		defer cache.UpdateTask.Unsubscribe(id)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}

				item := SelfUpdateEvent{Type: event.Type, ID: event.ID, Data: event.Data}
				select {
				case out <- item:
				case <-ctx.Done():
					return
				}
				// done 是最后一条事件，转发后立即结束
				if event.Type == cache.UpdateEventDone {
					return
				}
			}
		}
	}()

	return out, service.Success(nil)
}

// StartSelfUpdate 发起平台自更新：下载最新 Release、校验、备份数据库、替换二进制并原地重启。
//   - 只接受管理员调用（路由层限制）
//   - proxy 是加速节点前缀，为空表示直连；只影响资产下载，元数据始终直连 GitHub
//   - 更新在后台 goroutine 中执行，与请求生命周期解耦，前端刷新或断线都不会中断
func (s *Service) StartSelfUpdate(actor service.Actor, proxy string) service.Result {
	if proxy != "" && !utils.IsValidURL(proxy) {
		logger.Logger.Warnf("非法加速节点已拦截, api: %s, proxy: %s", actor.Path, proxy)
		return service.Fail(constant.CodePlatformInvalidURL)
	}

	if code, reason := selfUpdateBlock(); code != 0 {
		logger.Logger.Infof("当前环境不支持平台自更新, reason: %s, api: %s", reason, actor.Path)
		return service.Fail(code)
	}

	// 更新会重启进程，正在跑的游戏安装/更新脚本会被中断，因此直接拒绝并发
	if cache.GameTask.Busy() {
		logger.Logger.Infof("游戏安装或更新任务进行中, 拒绝平台更新, api: %s", actor.Path)
		return service.Fail(constant.CodePlatformGameTaskBusy)
	}

	release, err := fetchRelease(true)
	if err != nil {
		logger.Logger.Errorf("查询平台最新版本失败, err: %v, api: %s", err, actor.Path)
		return service.Fail(constant.CodePlatformUpdateCheckFail)
	}
	if compareVersion(release.TagName, constant.Version) <= 0 {
		logger.Logger.Infof("当前版本已是最新, current: %s, latest: %s", constant.Version, release.TagName)
		return service.Fail(constant.CodePlatformUpdateNotNeeded)
	}

	asset, ok := release.asset(runtime.GOOS)
	if !ok || asset.sha256() == "" || asset.URL == "" {
		logger.Logger.Errorf("发布资产不可用, tag: %s, os: %s", release.TagName, runtime.GOOS)
		return service.Fail(constant.CodePlatformUpdateAssetInvalid)
	}

	if !cache.UpdateTask.Begin(constant.Version, release.TagName) {
		return service.Fail(constant.CodePlatformUpdateBusy)
	}

	logger.Logger.Infof("开始更新平台, from: %s, to: %s, user: %s, proxy: %s", constant.Version, release.TagName, actor.Username, proxy)
	go runSelfUpdate(actor, release, asset, proxy)

	return service.Msg("update started", cache.UpdateTask.State())
}

// versionInfo 组装版本检查结果：环境判定 + Release 查询
func (s *Service) versionInfo(refresh bool) VersionInfo {
	info := VersionInfo{
		Current:     constant.Version,
		OS:          runtime.GOOS,
		Arch:        runtime.GOARCH,
		InContainer: inContainer(),
		Asset:       assetName(runtime.GOOS),
	}

	code, reason := selfUpdateBlock()
	info.CanSelfUpdate = code == 0
	info.Reason = reason

	release, err := fetchRelease(refresh)
	if err != nil {
		logger.Logger.Warnf("查询平台最新版本失败, err: %v", err)
		info.CheckFailed = true
		return info
	}
	if release.TagName == "" {
		logger.Logger.Warnf("查询平台最新版本失败, tag 为空")
		info.CheckFailed = true
		return info
	}

	info.Latest = release.TagName
	info.Notes = release.Body
	info.CheckedAt = release.fetchedAt

	if cmp := compareVersion(release.TagName, constant.Version); cmp > 0 {
		info.Updatable = true
	} else if cmp < 0 && info.CanSelfUpdate {
		// 开发版的版本号比线上大，不提示也不允许回退
		info.Reason = SelfUpdateReasonLocalNewer
	}

	return info
}

// runSelfUpdate 执行更新流程。任何一步失败都要落一条记录并结束任务，
// 除"替换二进制"之后的步骤外都不会改动现有文件。
func runSelfUpdate(actor service.Actor, release *releaseInfo, asset releaseAsset, proxy string) {
	from := constant.Version
	to := release.TagName
	startedAt := time.Now()
	record := updateRecord{
		RunID:     startedAt.Format("20060102-150405"),
		From:      from,
		To:        to,
		User:      actor.Username,
		Proxy:     proxy,
		StartedAt: startedAt.Unix(),
		Result:    resultPending,
	}

	fail := func(code int, err error) {
		logger.Logger.Errorf("平台更新失败, from: %s, to: %s, code: %d, err: %v", from, to, code, err)
		record.Result = resultFailed
		record.Code = code
		record.FinishedAt = time.Now().Unix()
		record.Message = err.Error()
		writeRecord(record)
		cache.UpdateTask.Complete(false, false, err.Error())
	}

	if err := prepareUpdateDir(); err != nil {
		fail(constant.CodePlatformUpdateDownloadFail, err)
		return
	}
	writeRecord(record)

	// 新二进制解到可执行文件同目录：最终 rename 必须同目录，跨设备 rename 会失败
	exe, err := currentBinary()
	if err != nil {
		fail(constant.CodePlatformSelfUpdateUnsupported, err)
		return
	}
	newBinary := filepath.Join(filepath.Dir(exe), fmt.Sprintf(".%s-%s.new", constant.UpdateBinaryName, to))
	// rename 成功后该路径已不存在，删除是空操作；失败路径则需要清理残留
	defer func() {
		_ = os.Remove(newBinary)
	}()

	cache.UpdateTask.Progress(cache.UpdatePhaseDownloading, 5, "正在下载安装包")
	sum, err := downloadAsset(asset, proxy)
	if err != nil {
		fail(constant.CodePlatformUpdateDownloadFail, err)
		return
	}

	cache.UpdateTask.Progress(cache.UpdatePhaseVerifying, 60, "正在校验安装包")
	if want := asset.sha256(); sum != want {
		fail(constant.CodePlatformUpdateDigestMismatch, fmt.Errorf("sha256 不一致, want: %s, got: %s", want, sum))
		return
	}

	cache.UpdateTask.Progress(cache.UpdatePhaseExtracting, 70, "正在解压安装包")
	if err := extractBinary(downloadPath(), newBinary); err != nil {
		fail(constant.CodePlatformUpdateExtractFail, err)
		return
	}

	cache.UpdateTask.Progress(cache.UpdatePhasePreflight, 78, "正在预检新版本")
	if err := verifyBinary(newBinary, to); err != nil {
		fail(constant.CodePlatformUpdatePreflightFail, err)
		return
	}

	// 数据库与旧版本二进制放在同一个备份目录，按目录整体淘汰
	backupDir := updateBackupDir(record.RunID, from, to)
	cache.UpdateTask.Progress(cache.UpdatePhaseBackup, 84, "正在备份数据库")
	if err := backupDatabase(backupDir); err != nil {
		fail(constant.CodePlatformUpdateBackupFail, err)
		return
	}

	cache.UpdateTask.Progress(cache.UpdatePhaseReplacing, 92, "正在替换程序文件")
	state, err := applyUpdate(record, newBinary, backupDir)
	if err != nil {
		fail(constant.CodePlatformUpdateReplaceFail, err)
		return
	}

	// 进程马上会被新镜像替换，先把"即将重启"推给前端
	cache.UpdateTask.Progress(cache.UpdatePhaseRestarting, 100, "正在重启，页面将自动重连")
	time.Sleep(constant.UpdateRestartFlushDelay)

	logger.Logger.Infof("平台更新即将重启, from: %s, to: %s, run: %s", from, to, state.RunID)
	if err := execSelf(state); err != nil {
		// exec 只在失败时返回：杀掉守护、回滚文件，保证进程还能继续提供服务
		cancelPendingUpdate(state, err)
		fail(constant.CodePlatformUpdateReplaceFail, err)
	}
}

// firstNonEmpty 返回第一个非空字符串
func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}

	return ""
}
