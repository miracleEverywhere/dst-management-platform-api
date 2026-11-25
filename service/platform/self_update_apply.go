package platform

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/logger"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

// 更新记录的结果取值
const (
	resultPending    = "pending"
	resultSuccess    = "success"
	resultFailed     = "failed"
	resultRolledBack = "rolled_back"
)

// updateRecord 是一次平台更新的记录，落到 dmp_files/dmp_update/records 下。
// 每次更新一个文件，避免"读改写"带来的并发覆盖：更新线程、新进程与回滚守护
// 都可能改写同一条记录，各写各的文件最省事。
type updateRecord struct {
	RunID      string `json:"runId"`
	From       string `json:"from"`
	To         string `json:"to"`
	User       string `json:"user"`
	Proxy      string `json:"proxy,omitempty"`
	StartedAt  int64  `json:"startedAt"`
	FinishedAt int64  `json:"finishedAt,omitempty"`
	// Code 失败时的业务错误码，便于对照 docs/error-codes.md 排查
	Code    int    `json:"code,omitempty"`
	Result  string `json:"result"`
	Message string `json:"message,omitempty"`
}

// pendingUpdate 是替换二进制前落盘的回滚状态（dmp_files/dmp_update/pending.json），
// 回滚守护与新进程都读它：守护据此判断新版本是否启动成功，新进程据此上报健康并落成功记录。
type pendingUpdate struct {
	RunID     string   `json:"runId"`
	From      string   `json:"from"`
	To        string   `json:"to"`
	User      string   `json:"user"`
	Binary    string   `json:"binary"`
	Backup    string   `json:"backup"`
	Failed    string   `json:"failed"`
	Argv      []string `json:"argv"`
	Cwd       string   `json:"cwd"`
	Pid       int      `json:"pid"`
	GuardWait int      `json:"guardWait"`
	StartedAt int64    `json:"startedAt"`

	// guardCmd 回滚守护进程句柄，用于 exec 失败时取消守护；不落盘
	guardCmd *exec.Cmd
}

// healthState 是新进程上报的健康标记（dmp_files/dmp_update/health.json）
type healthState struct {
	Version string `json:"version"`
	Pid     int    `json:"pid"`
	At      int64  `json:"at"`
}

// updateDir 自更新工作目录
func updateDir() string {
	return filepath.Join(cache.CurrentDir, constant.UpdateDir)
}

// downloadDir 下载与解包目录，每次更新前整体清空
func downloadDir() string {
	return filepath.Join(updateDir(), "download")
}

// downloadPath 安装包落盘路径
func downloadPath() string {
	return filepath.Join(downloadDir(), constant.UpdateBinaryName+".tgz")
}

// pendingPath 回滚状态文件
func pendingPath() string {
	return filepath.Join(updateDir(), "pending.json")
}

// healthPath 健康标记文件
func healthPath() string {
	return filepath.Join(updateDir(), "health.json")
}

// guardLogPath 回滚守护日志
func guardLogPath() string {
	return filepath.Join(updateDir(), "guard.log")
}

// recordsDir 更新记录目录
func recordsDir() string {
	return filepath.Join(updateDir(), "records")
}

// backupDirName 更新备份根目录：数据库与旧版本二进制都放在这里
func backupDirName() string {
	return filepath.Join(updateDir(), "backup")
}

// updateBackupDir 本次更新的备份目录。
// 目录名以 runId 开头（形如 20261125-104200-v3.2.3-to-v3.2.4），排出来就是时间顺序。
func updateBackupDir(runID, from, to string) string {
	return filepath.Join(backupDirName(), fmt.Sprintf("%s-%s-to-%s", runID, from, to))
}

// prepareUpdateDir 准备自更新目录：records/backup 常驻，download 每次重建
func prepareUpdateDir() error {
	for _, dir := range []string{updateDir(), recordsDir(), backupDirName()} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("创建更新目录失败: %w", err)
		}
	}
	if err := os.RemoveAll(downloadDir()); err != nil {
		return fmt.Errorf("清理下载目录失败: %w", err)
	}
	if err := os.MkdirAll(downloadDir(), 0o755); err != nil {
		return fmt.Errorf("创建下载目录失败: %w", err)
	}
	// 上一次更新留下的健康标记必须清掉，否则本次回滚守护会误判"新版本已启动"
	if err := os.Remove(healthPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("清理健康标记失败: %w", err)
	}

	return nil
}

// backupDatabase 更新前备份数据库。
// SQLite 处于 WAL 模式，dmp.db 里可能不含最近提交的数据，因此同时复制 -wal / -shm；
// 运行中复制只能做到"尽力一致"，用途是出问题时能退回到更新前的状态，不是严格的备份方案。
func backupDatabase(dir string) error {
	dbDir := cache.DBPath
	if dbDir == "" {
		dbDir = "data"
	}
	if _, err := os.Stat(filepath.Join(dbDir, "dmp.db")); err != nil {
		return fmt.Errorf("数据库文件不可用: %w", err)
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("创建备份目录失败: %w", err)
	}

	for _, name := range []string{"dmp.db", "dmp.db-wal", "dmp.db-shm"} {
		src := filepath.Join(dbDir, name)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		if err := copyFile(src, filepath.Join(dir, name)); err != nil {
			return fmt.Errorf("备份 %s 失败: %w", name, err)
		}
	}

	pruneBackups()
	cache.UpdateTask.Log("数据库已备份到 " + dir)

	return nil
}

// copyFile 复制数据库文件并落盘（权限 0600）
func copyFile(src, dst string) error {
	return copyFileMode(src, dst, 0o600)
}

// copyFileMode 复制文件并落盘，mode 为目标权限
func copyFileMode(src, dst string, mode os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		_ = in.Close()
	}()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		_ = out.Close()
		return err
	}
	if err := out.Sync(); err != nil {
		_ = out.Close()
		return err
	}

	return out.Close()
}

// moveFile 把 src 移到 dst。同一文件系统上用 rename（原子），
// 跨文件系统（EXDEV）时退化为复制 + 删除——工作目录与 dmp_files 未必在同一个挂载点。
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	} else if !errors.Is(err, syscall.EXDEV) {
		return err
	}

	if err := copyFileMode(src, dst, 0o755); err != nil {
		return err
	}

	return os.Remove(src)
}

// pruneBackups 只保留最近若干份备份（数据库与旧版本二进制按目录整体淘汰）
func pruneBackups() {
	entries, err := os.ReadDir(backupDirName())
	if err != nil {
		return
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}
	if len(dirs) <= constant.UpdateKeepBackups {
		return
	}

	sort.Strings(dirs)
	for _, name := range dirs[:len(dirs)-constant.UpdateKeepBackups] {
		_ = os.RemoveAll(filepath.Join(backupDirName(), name))
	}
}

// applyUpdate 替换二进制并启动回滚守护。
// 顺序很关键：先落盘回滚状态 → 启动守护 → 把旧文件挪进备份目录 → 重命名新文件，
// 这样任何一步失败都还能把旧文件放回去，且守护手里始终有一份可回滚的备份。
// 旧版本备份放在 dmp_files/dmp_update/backup/<runId>-<旧版本>-to-<新版本>/dmp，
// 不落在可执行文件旁边；新文件从同目录的临时文件 rename 过来，保证替换这一步是原子的。
func applyUpdate(record updateRecord, newBinary, backupDir string) (pendingUpdate, error) {
	exe, err := currentBinary()
	if err != nil {
		return pendingUpdate{}, err
	}

	runID := record.RunID
	if runID == "" {
		runID = time.Now().Format("20060102-150405")
	}
	state := pendingUpdate{
		RunID:     runID,
		From:      record.From,
		To:        record.To,
		User:      record.User,
		Binary:    exe,
		Backup:    filepath.Join(backupDir, constant.UpdateBackupBinaryName),
		Failed:    filepath.Join(backupDir, constant.UpdateFailedBinaryName),
		Argv:      os.Args,
		Cwd:       cache.CurrentDir,
		Pid:       os.Getpid(),
		GuardWait: int(constant.UpdateHealthTimeout / time.Second),
		StartedAt: record.StartedAt,
	}

	// 备份目录由 backupDatabase 创建，这里再兜一次：旧版本二进制要挪进去
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return pendingUpdate{}, fmt.Errorf("创建备份目录失败: %w", err)
	}

	if err := writeJSONFile(pendingPath(), state); err != nil {
		return pendingUpdate{}, fmt.Errorf("写入回滚状态失败: %w", err)
	}

	guard, err := startGuard(exe)
	if err != nil {
		_ = os.Remove(pendingPath())
		return pendingUpdate{}, fmt.Errorf("启动回滚守护失败: %w", err)
	}
	state.guardCmd = guard

	// 旧版本挪进备份目录，留给人工回滚
	if err := moveFile(state.Binary, state.Backup); err != nil {
		cancelPendingUpdate(state, err)
		return pendingUpdate{}, fmt.Errorf("备份当前程序失败: %w", err)
	}

	if err := os.Rename(newBinary, state.Binary); err != nil {
		cancelPendingUpdate(state, err)
		return pendingUpdate{}, fmt.Errorf("替换程序文件失败: %w", err)
	}

	return state, nil
}

// execSelf 用新镜像原地替换当前进程：PID 不变，启动参数、工作目录与环境变量全部沿用。
// 监听 socket 带 CLOEXEC，exec 时关闭，新进程启动后重新监听。
// 成功时不会返回。
func execSelf(state pendingUpdate) error {
	return syscall.Exec(state.Binary, state.Argv, os.Environ())
}

// cancelPendingUpdate 取消更新的收尾：杀掉守护、放回旧文件、清理状态文件。
// 只在 exec 失败（程序还能继续跑）或替换失败时调用。
func cancelPendingUpdate(state pendingUpdate, cause error) {
	if state.guardCmd != nil && state.guardCmd.Process != nil {
		_ = state.guardCmd.Process.Kill()
	}
	_ = os.Remove(pendingPath())

	if _, err := os.Stat(state.Binary); err == nil {
		_ = moveFile(state.Binary, state.Failed)
	}
	if _, err := os.Stat(state.Backup); err == nil {
		_ = moveFile(state.Backup, state.Binary)
	}

	cache.UpdateTask.Log("更新已取消: " + cause.Error())
	_ = os.Remove(healthPath())
}

// startGuard 启动回滚守护。
// 守护用"当前正在运行的旧二进制"启动（-update_guard），因此不依赖新版本是否可用：
// 新进程若启动失败，守护仍能把备份目录里的旧版本放回去并重新拉起。
func startGuard(exe string) (*exec.Cmd, error) {
	logFile, err := os.OpenFile(guardLogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(exe, "-update_guard", pendingPath())
	cmd.Dir = cache.CurrentDir
	cmd.Env = os.Environ()
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	// 独立会话：父进程 exec 之后守护仍然是独立的进程，不受终端/进程组影响
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		_ = logFile.Close()
		return nil, err
	}
	// 子进程已持有自己的句柄，父进程这份可以关掉，避免带进新镜像
	_ = logFile.Close()

	return cmd, nil
}

// RunUpdateGuard 是回滚守护的入口，由入口文件在解析到 -update_guard 时调用。
// 它等待新进程上报健康标记；超时或发现新进程已退出，就把备份目录里的旧版本放回去并重新拉起。
func RunUpdateGuard(statePath string) {
	// 守护不通往 InitCache，工作目录要自己补上：更新目录、日志与重启命令都依赖它
	if cache.CurrentDir == "" {
		if wd, err := os.Getwd(); err == nil {
			cache.CurrentDir = wd
		}
	}

	logFile, err := os.OpenFile(guardLogPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer func() {
		_ = logFile.Close()
	}()

	logf := func(format string, args ...any) {
		_, _ = fmt.Fprintf(logFile, "[%s] %s\n", time.Now().Format("2006-01-02 15:04:05"), fmt.Sprintf(format, args...))
	}

	state, err := readPending(statePath)
	if err != nil {
		logf("读取回滚状态失败, err: %v", err)
		return
	}
	if state.GuardWait <= 0 {
		state.GuardWait = int(constant.UpdateHealthTimeout / time.Second)
	}

	logf("回滚守护启动, 等待 %s 上报健康状态, 最长 %d 秒, pid: %d", state.To, state.GuardWait, state.Pid)

	deadline := time.Now().Add(time.Duration(state.GuardWait) * time.Second)
	var deadSince time.Time
	for {
		// 健康标记必须同时匹配版本与时间：上一次更新残留的标记不能算数
		if health, ok := readHealth(); ok && health.Version == state.To && health.At >= state.StartedAt {
			logf("新版本 %s 已上报健康状态, 守护正常退出", state.To)
			return
		}
		if _, err := os.Stat(statePath); errors.Is(err, os.ErrNotExist) {
			logf("回滚状态已被清理, 守护退出")
			return
		}

		alive := processAlive(state.Pid)
		if !alive {
			if deadSince.IsZero() {
				deadSince = time.Now()
				logf("进程 %d 已退出, 等待 %s 后回滚", state.Pid, constant.UpdateDeadGrace)
			}
			if time.Since(deadSince) > constant.UpdateDeadGrace {
				break
			}
		} else {
			deadSince = time.Time{}
		}

		if time.Now().After(deadline) {
			if alive {
				// 进程还活着却迟迟没有上报：可能是初始化很慢。此时回滚会启动第二份实例抢端口，
				// 比"服务能用但记录没更新"更糟，因此放弃回滚。
				logf("等待超时但进程 %d 仍在运行, 放弃回滚以避免启动重复实例", state.Pid)
				return
			}
			break
		}

		time.Sleep(time.Second)
	}

	rollbackUpdate(state, logf)
}

// rollbackUpdate 把坏掉的新版本改名留存，恢复旧版本并重新拉起
func rollbackUpdate(state pendingUpdate, logf func(format string, args ...any)) {
	logf("开始回滚: %s -> %s", state.To, state.From)

	if _, err := os.Stat(state.Binary); err == nil {
		if err := moveFile(state.Binary, state.Failed); err != nil {
			logf("留存故障程序失败, err: %v", err)
		}
	}
	if err := moveFile(state.Backup, state.Binary); err != nil {
		logf("恢复旧版本失败, err: %v", err)
		writeRecord(updateRecord{
			RunID: state.RunID, From: state.From, To: state.To, User: state.User,
			StartedAt: state.StartedAt, FinishedAt: time.Now().Unix(),
			Result: resultRolledBack, Message: "恢复旧版本失败: " + err.Error(),
		})
		return
	}
	logf("已恢复 %s, 重新启动", state.Binary)

	if err := startRestartedBinary(state, logf); err != nil {
		logf("重启旧版本失败, err: %v", err)
	}

	writeRecord(updateRecord{
		RunID: state.RunID, From: state.From, To: state.To, User: state.User,
		StartedAt: state.StartedAt, FinishedAt: time.Now().Unix(),
		Result: resultRolledBack, Message: "新版本启动失败，已自动回滚",
	})
	_ = os.Remove(pendingPath())
}

// startRestartedBinary 用原参数重新拉起旧版本，输出写入 rollback.log
func startRestartedBinary(state pendingUpdate, logf func(format string, args ...any)) error {
	logPath := filepath.Join(updateDir(), "rollback.log")
	out, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	args := state.Argv
	if len(args) == 0 {
		args = []string{state.Binary}
	}
	cmd := exec.Command(state.Binary, args[1:]...)
	cmd.Dir = state.Cwd
	cmd.Env = os.Environ()
	cmd.Stdout = out
	cmd.Stderr = out
	// 独立会话，守护退出后旧版本继续运行
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	if err := cmd.Start(); err != nil {
		return err
	}
	pid := cmd.Process.Pid
	_ = cmd.Process.Release()
	logf("旧版本已启动, pid: %d", pid)

	return nil
}

// NotifyServerStarted 由入口在 HTTP 监听成功后调用。
// 存在待确认的更新时写入健康标记（回滚守护据此判定启动成功），落成功记录并清理状态。
func NotifyServerStarted() {
	state, err := readPending(pendingPath())
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			logger.Logger.Warnf("读取回滚状态失败, err: %v", err)
		}
		return
	}
	if state.To != constant.Version {
		// 目标版本与当前运行版本不一致（例如人工回滚后启动），不上报健康，交给守护处理
		logger.Logger.Warnf("待确认的更新版本与当前版本不一致, pending: %s, current: %s", state.To, constant.Version)
		return
	}

	if err := writeJSONFile(healthPath(), healthState{Version: state.To, Pid: os.Getpid(), At: time.Now().Unix()}); err != nil {
		logger.Logger.Errorf("写入健康标记失败, err: %v", err)
		return
	}

	writeRecord(updateRecord{
		RunID: state.RunID, From: state.From, To: state.To, User: state.User,
		StartedAt: state.StartedAt, FinishedAt: time.Now().Unix(), Result: resultSuccess,
	})
	_ = os.Remove(pendingPath())
	_ = os.RemoveAll(downloadDir())
	logger.Logger.Infof("平台已更新到 %s, 旧版本备份: %s", state.To, state.Backup)
}

// writeRecord 落一条更新记录。同名记录（同一 RunID）会被覆盖，因此状态推进不需要读改写。
func writeRecord(record updateRecord) {
	if record.RunID == "" {
		record.RunID = time.Now().Format("20060102-150405")
	}
	if err := os.MkdirAll(recordsDir(), 0o755); err != nil {
		logger.Logger.Errorf("创建更新记录目录失败, err: %v", err)
		return
	}

	name := fmt.Sprintf("%s-%s-to-%s.json", record.RunID, record.From, record.To)
	if err := writeJSONFile(filepath.Join(recordsDir(), name), record); err != nil {
		logger.Logger.Errorf("写入更新记录失败, err: %v", err)
		return
	}

	pruneRecords()
}

// pruneRecords 只保留最近若干条更新记录
func pruneRecords() {
	entries, err := os.ReadDir(recordsDir())
	if err != nil {
		return
	}

	var names []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".json") {
			names = append(names, entry.Name())
		}
	}
	if len(names) <= constant.UpdateKeepRecords {
		return
	}

	sort.Strings(names)
	for _, name := range names[:len(names)-constant.UpdateKeepRecords] {
		_ = os.Remove(filepath.Join(recordsDir(), name))
	}
}

// writeJSONFile 以"临时文件 + rename"的方式原子写 JSON
func writeJSONFile(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	return os.Rename(tmpName, path)
}

// readPending 读取回滚状态
func readPending(path string) (pendingUpdate, error) {
	var state pendingUpdate
	if err := readJSONFile(path, &state); err != nil {
		return pendingUpdate{}, err
	}

	return state, nil
}

// readHealth 读取健康标记
func readHealth() (healthState, bool) {
	var health healthState
	if err := readJSONFile(healthPath(), &health); err != nil {
		return healthState{}, false
	}

	return health, true
}

// readJSONFile 读取 JSON 文件
func readJSONFile(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, value)
}

// processAlive 进程是否存在
func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}

	return syscall.Kill(pid, 0) == nil
}
