package platform

import (
	"context"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"

	"github.com/creack/pty"
)

const (
	// gameTaskInstallScript / gameTaskUpdateScript 由 embedFS 在启动时释放到工作目录
	gameTaskInstallScript = "manual_install.sh"
	gameTaskUpdateScript  = "manual_update.sh"

	// gameTaskTimeout 兜底超时：安装含系统依赖与 SteamCMD 下载，正常几分钟内完成，
	// 超过两小时视为卡死，直接杀掉，避免互斥锁被永久占用。
	gameTaskTimeout = 2 * time.Hour

	// gameTaskPtyRows / gameTaskPtyCols 给足宽度，减少进度条在 PTY 内折行
	gameTaskPtyRows = 40
	gameTaskPtyCols = 200

	// gameTaskReadBufSize PTY 单次读取大小
	gameTaskReadBufSize = 4096

	// gameTaskFlushInterval 未换行片段的刷出间隔。进度条只有 \r 没有 \n，
	// 靠定时刷出保证长下载期间前端仍有实时反馈。
	gameTaskFlushInterval = 500 * time.Millisecond

	// gameTaskPendingMax 单行长度上限，超过就强制刷出，避免无换行输出把内存撑大
	gameTaskPendingMax = 8 << 10
)

// 安装/更新任务的事件类型，与 SSE 的 event 名一致
const (
	GameTaskEventChunk = cache.GameTaskEventChunk
	GameTaskEventDone  = cache.GameTaskEventDone
)

// GameTaskEvent 是推送给 SSE 的一条任务事件
type GameTaskEvent struct {
	// Type 取 chunk（一段输出）或 done（任务结束）
	Type string
	// ID 是产生该事件之后的绝对输出偏移，可作为 Last-Event-ID 续传
	ID int64
	// Data 对应 cache.GameTaskChunk 或 cache.GameTaskDone
	Data any
}

// GameTaskStart 启动游戏安装/更新任务。
//   - action 只接受 install / update，脚本名来自常量表，不接受调用方传入任意路径
//   - 任务在后台 goroutine 中执行，与请求生命周期解耦：前端刷新或断线不会中断安装
//   - 已有任务在跑（含定时任务、控制台更新、诊断修复）时返回忙，不排队
func (s *Service) GameTaskStart(actor service.Actor, action string) service.Result {
	script, ok := gameTaskScripts()[action]
	if !ok {
		logger.Logger.Infof("请求参数错误: action=%s, api: %s", action, actor.Path)
		return service.Fail(constant.CodePlatformBadRequest)
	}

	path := filepath.Join(cache.CurrentDir, script)
	if info, err := os.Stat(path); err != nil || info.IsDir() {
		logger.Logger.Errorf("安装/更新脚本不可用, path: %s, err: %v", path, err)
		return service.Fail(constant.CodePlatformGameTaskStartFail)
	}

	if !cache.GameTask.Acquire(action, true) {
		logger.Logger.Infof("已有游戏安装或更新任务在跑, 拒绝新的 %s 任务, api: %s", action, actor.Path)
		return service.Fail(constant.CodePlatformGameTaskBusy)
	}

	logger.Logger.Infof("开始执行游戏安装/更新任务, action: %s, script: %s, user: %s", action, path, actor.Username)
	go runGameTaskScript(action, script)

	return service.Success(cache.GameTask.State())
}

// GameTaskStream 订阅任务输出：先补发仍保留的历史，再持续推送新增内容。
// 连接断开只解绑订阅，不影响任务本身；刷新页面后传 after=0 即可重新接上。
func (s *Service) GameTaskStream(ctx context.Context, actor service.Actor, after int64) (<-chan GameTaskEvent, service.Result) {
	if after < 0 {
		after = 0
	}

	id, events := cache.GameTask.Subscribe(after)
	if events == nil {
		logger.Logger.Infof("没有可订阅的安装/更新任务, api: %s", actor.Path)
		return nil, service.Fail(constant.CodePlatformGameTaskNotRunning)
	}

	out := make(chan GameTaskEvent, 16)
	go func() {
		defer close(out)
		defer cache.GameTask.Unsubscribe(id)

		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-events:
				if !ok {
					return
				}

				item := GameTaskEvent{Type: event.Type, ID: event.ID, Data: event.Data}
				select {
				case out <- item:
				case <-ctx.Done():
					return
				}
				// done 是最后一条事件，转发后立即结束，让 SSE 响应干净收尾
				if event.Type == GameTaskEventDone {
					return
				}
			}
		}
	}()

	return out, service.Success(nil)
}

// GameTaskState 当前任务状态，供前端刷新页面后恢复界面
func (s *Service) GameTaskState() service.Result {
	return service.Success(cache.GameTask.State())
}

// gameTaskScripts 动作到脚本文件名的白名单
func gameTaskScripts() map[string]string {
	return map[string]string{
		cache.GameTaskActionInstall: gameTaskInstallScript,
		cache.GameTaskActionUpdate:  gameTaskUpdateScript,
	}
}

// runGameTaskScript 在 PTY 中执行安装/更新脚本，把清洗后的输出写进任务表。
// 用 PTY 是为了保持与人工执行一致的输出行为：apt-get / wget / SteamCMD 在管道下会
// 退化成块缓冲，只能等到缓冲区满才吐一次，长任务看起来像卡死。
func runGameTaskScript(action, script string) {
	ctx, cancel := context.WithTimeout(context.Background(), gameTaskTimeout)
	defer cancel()

	// 脚本名来自白名单常量；cwd 固定为工作目录，脚本内部的 pwd 依赖这一点
	cmd := exec.CommandContext(ctx, "bash", "-lc", "./"+script)
	cmd.Dir = cache.CurrentDir
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"LANG=en_US.UTF-8",
		"LC_ALL=en_US.UTF-8",
	)

	f, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: gameTaskPtyRows, Cols: gameTaskPtyCols})
	if err != nil {
		logger.Logger.Errorf("启动安装/更新脚本失败, action: %s, err: %v", action, err)
		cache.GameTask.Complete(false, -1)
		return
	}
	defer func() {
		_ = f.Close()
	}()

	writer := newGameTaskLineWriter(func(chunk []byte) {
		cache.GameTask.Append(chunk)
	})

	stopFlush := make(chan struct{})
	var flushWG sync.WaitGroup
	flushWG.Add(1)
	go func() {
		defer flushWG.Done()

		ticker := time.NewTicker(gameTaskFlushInterval)
		defer ticker.Stop()
		for {
			select {
			case <-stopFlush:
				return
			case <-ticker.C:
				writer.Flush()
			}
		}
	}()

	buf := make([]byte, gameTaskReadBufSize)
	for {
		n, readErr := f.Read(buf)
		if n > 0 {
			writer.Write(buf[:n])
		}
		if readErr != nil {
			// 子进程退出后 PTY 读取返回 EIO，属于正常结束
			break
		}
	}

	close(stopFlush)
	flushWG.Wait()
	writer.Flush()

	exitCode := 0
	if waitErr := cmd.Wait(); waitErr != nil {
		exitCode = -1
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			exitCode = exitErr.ExitCode()
		}
		logger.Logger.Warnf("安装/更新脚本退出异常, action: %s, exitCode: %d, err: %v", action, exitCode, waitErr)
	}

	logger.Logger.Infof("游戏安装/更新任务结束, action: %s, exitCode: %d", action, exitCode)
	cache.GameTask.Complete(exitCode == 0, exitCode)
}

// gameTaskStripState 是 ANSI 转义序列的状态机状态，跨读取块保持
type gameTaskStripState int

const (
	gameTaskStripNormal gameTaskStripState = iota
	gameTaskStripEsc
	gameTaskStripCSI
	gameTaskStripOSC
	gameTaskStripOSCEsc
)

// filter 去掉 ANSI 控制序列（颜色、光标移动、OSC 标题等），只留可显示文本。
// 前端不再使用终端组件渲染，这些序列留着只会变成可见乱码。
func (st *gameTaskStripState) filter(p []byte) []byte {
	out := make([]byte, 0, len(p))
	for _, b := range p {
		switch *st {
		case gameTaskStripNormal:
			switch b {
			case 0x1b:
				*st = gameTaskStripEsc
			case '\a':
				// 单独的 BEL，丢弃
			default:
				out = append(out, b)
			}
		case gameTaskStripEsc:
			switch b {
			case '[':
				*st = gameTaskStripCSI
			case ']', 'P', '^', '_':
				*st = gameTaskStripOSC
			default:
				// 其它两字节转义序列（含 ESC ESC），整体丢弃
				*st = gameTaskStripNormal
			}
		case gameTaskStripCSI:
			// CSI 以 0x40-0x7E 的终止字节结束
			if b >= 0x40 && b <= 0x7e {
				*st = gameTaskStripNormal
			}
		case gameTaskStripOSC:
			switch b {
			case '\a':
				*st = gameTaskStripNormal
			case 0x1b:
				*st = gameTaskStripOSCEsc
			}
		case gameTaskStripOSCEsc:
			if b == '\\' {
				*st = gameTaskStripNormal
			} else {
				*st = gameTaskStripOSC
			}
		}
	}

	return out
}

// gameTaskLineWriter 把 PTY 输出整理成纯文本行：
//   - 去掉 ANSI 控制序列
//   - 单独出现的 \r 表示「覆盖当前行」（进度条原地刷新），丢弃尚未刷出的半行，
//     避免日志里堆几百行半截进度
//   - 只有 \r 没有 \n 的输出靠定时 Flush 刷出，保证长任务仍有实时反馈
//
// 刷出规则保证每段输出都以换行结尾，前端可以直接按行追加到列表里。
type gameTaskLineWriter struct {
	mu        sync.Mutex
	pending   []byte
	crPending bool
	emit      func([]byte)
	strip     gameTaskStripState
}

func newGameTaskLineWriter(emit func([]byte)) *gameTaskLineWriter {
	return &gameTaskLineWriter{emit: emit}
}

// Write 追加一段原始 PTY 输出
func (w *gameTaskLineWriter) Write(p []byte) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, b := range w.strip.filter(p) {
		if w.crPending {
			w.crPending = false
			// 单独 \r：覆盖当前行，丢弃还没刷出的半行
			if b != '\n' {
				w.pending = w.pending[:0]
			}
		}

		switch b {
		case '\n':
			w.pending = append(w.pending, '\n')
			w.flushLocked()
		case '\r':
			w.crPending = true
		default:
			w.pending = append(w.pending, b)
		}
	}

	if len(w.pending) >= gameTaskPendingMax {
		w.flushLocked()
	}
}

// Flush 刷出尚未换行的残留内容，由定时器调用
func (w *gameTaskLineWriter) Flush() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.flushLocked()
}

func (w *gameTaskLineWriter) flushLocked() {
	if len(w.pending) == 0 {
		return
	}

	if w.pending[len(w.pending)-1] != '\n' {
		w.pending = append(w.pending, '\n')
	}

	out := make([]byte, len(w.pending))
	copy(out, w.pending)
	w.pending = w.pending[:0]

	w.emit(out)
}
