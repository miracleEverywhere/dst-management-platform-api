package cache

import (
	"sync"
	"time"
)

// 游戏安装/更新任务的动作标识。install 与 update 共用同一把互斥锁：
// 定时自动更新、控制台「更新游戏」、诊断修复动作与网页安装/更新之间，
// 同一时刻只允许一个在跑，避免多个 SteamCMD 同时操作同一个游戏目录。
const (
	GameTaskActionInstall = "install"
	GameTaskActionUpdate  = "update"

	// gameTaskBufferMax 保留的纯文本输出上限（字节）。超出后丢弃最早的部分，
	// 前端按绝对偏移续传；已被丢弃的区间不会再补发。
	gameTaskBufferMax = 512 << 10
	// gameTaskSubBuffer 单个订阅者的待发送队列长度。浏览器消费不过来时丢弃最旧的事件，
	// 宁可少显示几行，也不能让安装进程被慢客户端阻塞。
	gameTaskSubBuffer = 512
)

// 任务事件类型，与 SSE 的 event 名一致
const (
	GameTaskEventChunk = "chunk"
	GameTaskEventDone  = "done"
)

// GameTaskEvent 是一条投递给 SSE 订阅者的事件
type GameTaskEvent struct {
	Type string
	// ID 是产生该事件之后的绝对输出偏移，前端可原样作为 Last-Event-ID 回传续传
	ID   int64
	Data any
}

// GameTaskChunk 是一段已经清洗过的纯文本输出（不含 ANSI 控制序列）
type GameTaskChunk struct {
	C string `json:"c"`
}

// GameTaskDone 是任务结束事件
type GameTaskDone struct {
	Action   string `json:"action"`
	Success  bool   `json:"success"`
	ExitCode int    `json:"exitCode"`
}

// GameTaskState 是任务状态快照，供前端刷新页面后恢复界面
type GameTaskState struct {
	Running bool `json:"running"`
	// Action 取 install / update
	Action string `json:"action"`
	// Streamed 为 false 表示该任务是定时任务、控制台或诊断触发的，
	// 输出没有经过本任务表，前端不应据 Success 弹「安装/更新完成」
	Streamed   bool  `json:"streamed"`
	StartedAt  int64 `json:"startedAt"`
	FinishedAt int64 `json:"finishedAt"`
	Success    bool  `json:"success"`
	ExitCode   int   `json:"exitCode"`
	// Offset 是已产生的输出总字节数，可作为重新订阅时的起点
	Offset int64 `json:"offset"`
}

// gameTaskSubscriber 是单个 SSE 连接对应的订阅者。
// 自带锁与关闭标记，避免「解除订阅关通道」与「任务线程推送」并发时向已关闭通道发送。
type gameTaskSubscriber struct {
	mu     sync.Mutex
	ch     chan GameTaskEvent
	closed bool
}

func (s *gameTaskSubscriber) push(event GameTaskEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}

	select {
	case s.ch <- event:
		return
	default:
	}

	// 队列已满：丢掉最旧的一条再塞进去，保持订阅者始终能拿到最新输出
	select {
	case <-s.ch:
	default:
	}
	select {
	case s.ch <- event:
	default:
	}
}

func (s *gameTaskSubscriber) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	close(s.ch)
}

// gameTaskManager 是进程内唯一的游戏安装/更新任务表
type gameTaskManager struct {
	mu sync.Mutex

	running    bool
	action     string
	streamed   bool
	startedAt  time.Time
	finishedAt time.Time
	success    bool
	exitCode   int

	// buf 保存最近一段输出，baseOffset 是 buf[0] 对应的绝对偏移，offset 是已产生总量
	buf        []byte
	baseOffset int64
	offset     int64

	subs    map[int64]*gameTaskSubscriber
	nextSub int64
}

// GameTask 是游戏安装/更新任务表。它同时承担互斥职责：
// 需要操作游戏目录的调用方先 Acquire，结束后 Release 或 Complete。
var GameTask = &gameTaskManager{subs: make(map[int64]*gameTaskSubscriber)}

// Acquire 尝试独占游戏安装/更新。
// streamed 为 true 表示调用方会通过 Append/Complete 推送输出与结果（网页安装/更新）；
// 为 false 表示调用方自己执行命令、只共享互斥（定时任务、控制台、诊断修复）。
// 返回 false 表示已有任务在跑，调用方应直接放弃本次操作而不是排队。
func (m *gameTaskManager) Acquire(action string, streamed bool) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return false
	}

	m.running = true
	m.action = action
	m.streamed = streamed
	m.startedAt = time.Now()
	m.finishedAt = time.Time{}
	m.success = false
	m.exitCode = 0
	m.buf = nil
	m.baseOffset = 0
	m.offset = 0
	return true
}

// Append 追加一段输出并广播给订阅者。非流式任务不会调用它。
func (m *gameTaskManager) Append(p []byte) {
	if len(p) == 0 {
		return
	}

	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}

	m.buf = append(m.buf, p...)
	if over := len(m.buf) - gameTaskBufferMax; over > 0 {
		m.buf = m.buf[over:]
		m.baseOffset += int64(over)
	}
	m.offset += int64(len(p))
	event := GameTaskEvent{
		Type: GameTaskEventChunk,
		ID:   m.offset,
		Data: GameTaskChunk{C: string(p)},
	}
	targets := m.subscriberSnapshotLocked()
	m.mu.Unlock()

	for _, sub := range targets {
		sub.push(event)
	}
}

// Complete 结束流式任务并投递 done 事件
func (m *gameTaskManager) Complete(success bool, exitCode int) {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}

	m.running = false
	m.success = success
	m.exitCode = exitCode
	m.finishedAt = time.Now()
	event := GameTaskEvent{
		Type: GameTaskEventDone,
		ID:   m.offset,
		Data: GameTaskDone{Action: m.action, Success: success, ExitCode: exitCode},
	}
	targets := m.subscriberSnapshotLocked()
	m.mu.Unlock()

	for _, sub := range targets {
		sub.push(event)
	}
}

// Release 结束非流式任务：只释放互斥，不投递 done（调用方没有把输出交给任务表）
func (m *gameTaskManager) Release() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return
	}

	m.running = false
	m.streamed = false
	m.success = false
	m.exitCode = 0
	m.finishedAt = time.Now()
}

// Busy 是否已有游戏安装/更新任务在跑
func (m *gameTaskManager) Busy() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.running
}

// State 返回任务状态快照
func (m *gameTaskManager) State() GameTaskState {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := GameTaskState{
		Running:  m.running,
		Action:   m.action,
		Streamed: m.streamed,
		Success:  m.success,
		ExitCode: m.exitCode,
		Offset:   m.offset,
	}
	if !m.startedAt.IsZero() {
		state.StartedAt = m.startedAt.Unix()
	}
	if !m.finishedAt.IsZero() {
		state.FinishedAt = m.finishedAt.Unix()
	}

	return state
}

// Subscribe 订阅任务输出并补发历史。
// after 是客户端已经收到的绝对偏移，首次订阅传 0 即可：缓冲区被截断时会从仍保留的
// 最早偏移开始补发。没有正在运行、且进程启动后从未执行过任务时返回 nil，
// 调用方应提示「当前没有任务」而不是挂起一条空流。
func (m *gameTaskManager) Subscribe(after int64) (int64, <-chan GameTaskEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running && m.finishedAt.IsZero() {
		return 0, nil
	}

	ch := make(chan GameTaskEvent, gameTaskSubBuffer)
	if after < m.baseOffset {
		after = m.baseOffset
	}
	if from := after - m.baseOffset; from < int64(len(m.buf)) {
		ch <- GameTaskEvent{
			Type: GameTaskEventChunk,
			ID:   m.baseOffset + int64(len(m.buf)),
			Data: GameTaskChunk{C: string(m.buf[from:])},
		}
	}
	if !m.running {
		// 任务已结束：补发历史后立刻给出结果，刷新页面也能看到最终状态
		ch <- GameTaskEvent{
			Type: GameTaskEventDone,
			ID:   m.offset,
			Data: GameTaskDone{Action: m.action, Success: m.success, ExitCode: m.exitCode},
		}
	}

	m.nextSub++
	id := m.nextSub
	m.subs[id] = &gameTaskSubscriber{ch: ch}

	return id, ch
}

// Unsubscribe 解除订阅。关闭后该订阅者的通道会被关闭，接收方读到零值即退出。
func (m *gameTaskManager) Unsubscribe(id int64) {
	m.mu.Lock()
	sub, ok := m.subs[id]
	if ok {
		delete(m.subs, id)
	}
	m.mu.Unlock()

	if ok {
		sub.close()
	}
}

func (m *gameTaskManager) subscriberSnapshotLocked() []*gameTaskSubscriber {
	if len(m.subs) == 0 {
		return nil
	}

	targets := make([]*gameTaskSubscriber, 0, len(m.subs))
	for _, sub := range m.subs {
		targets = append(targets, sub)
	}

	return targets
}
