package cache

import (
	"sync"
	"time"
)

// 平台自更新任务的事件类型，与 SSE 的 event 名一致
const (
	UpdateEventProgress = "progress"
	UpdateEventChunk    = "chunk"
	UpdateEventDone     = "done"
)

// 平台自更新的阶段标识，前端据此显示分阶段文案
const (
	UpdatePhaseChecking    = "checking"
	UpdatePhaseDownloading = "downloading"
	UpdatePhaseVerifying   = "verifying"
	UpdatePhaseExtracting  = "extracting"
	UpdatePhasePreflight   = "preflight"
	UpdatePhaseBackup      = "backup"
	UpdatePhaseReplacing   = "replacing"
	UpdatePhaseRestarting  = "restarting"
	UpdatePhaseFailed      = "failed"
)

const (
	// updateEventMax 保留的最近事件条数。更新任务的事件很少（几十条进度 + 少量日志），
	// 超过上限后丢弃最早的部分，订阅者按事件 ID 续传时只会漏掉很旧的进度。
	updateEventMax = 512
	// updateSubBuffer 单个订阅者的待发送队列长度，慢客户端不能阻塞更新流程
	updateSubBuffer = 128
)

// UpdateEvent 是投递给 SSE 订阅者的一条更新任务事件
type UpdateEvent struct {
	Type string
	// ID 事件序号，单调递增，前端可原样作为 Last-Event-ID 回传续传
	ID   int64
	Data any
}

// UpdateProgress 阶段进度
type UpdateProgress struct {
	Phase   string `json:"phase"`
	Percent int    `json:"percent"`
	Message string `json:"message"`
}

// UpdateChunk 一段纯文本输出（不含 ANSI 控制序列）
type UpdateChunk struct {
	C string `json:"c"`
}

// UpdateDone 任务结束事件
type UpdateDone struct {
	Success  bool   `json:"success"`
	Rollback bool   `json:"rollback"`
	From     string `json:"from"`
	To       string `json:"to"`
	Message  string `json:"message"`
}

// UpdateState 是更新任务状态快照，供前端刷新页面后恢复界面
type UpdateState struct {
	Running    bool   `json:"running"`
	Phase      string `json:"phase"`
	Percent    int    `json:"percent"`
	From       string `json:"from"`
	To         string `json:"to"`
	StartedAt  int64  `json:"startedAt"`
	FinishedAt int64  `json:"finishedAt"`
	Success    bool   `json:"success"`
	Rollback   bool   `json:"rollback"`
	Message    string `json:"message"`
	// Offset 已产生的事件数，可作为重新订阅时的起点
	Offset int64 `json:"offset"`
}

// updateSubscriber 单个 SSE 连接对应的订阅者。
// 自带锁与关闭标记，避免「解除订阅关通道」与「任务线程推送」并发时向已关闭通道发送。
type updateSubscriber struct {
	mu     sync.Mutex
	ch     chan UpdateEvent
	closed bool
}

func (s *updateSubscriber) push(event UpdateEvent) {
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

	select {
	case <-s.ch:
	default:
	}
	select {
	case s.ch <- event:
	default:
	}
}

func (s *updateSubscriber) close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	close(s.ch)
}

// updateTaskManager 是进程内唯一的平台自更新任务表。
// 与 gameTaskManager 的区别：更新任务的事件以「阶段 + 百分比」为主，日志是补充，
// 因此统一按事件序号缓冲，而不是按字节偏移缓冲输出。
type updateTaskManager struct {
	mu sync.Mutex

	running    bool
	phase      string
	percent    int
	from       string
	to         string
	message    string
	success    bool
	rollback   bool
	startedAt  time.Time
	finishedAt time.Time

	events []UpdateEvent
	nextID int64

	subs    map[int64]*updateSubscriber
	nextSub int64
}

// UpdateTask 平台自更新任务表，同时承担"同一时刻只允许一个更新"的互斥职责。
var UpdateTask = &updateTaskManager{subs: make(map[int64]*updateSubscriber)}

// Begin 尝试开始一次更新任务，返回 false 表示已有任务在跑
func (m *updateTaskManager) Begin(from, to string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return false
	}

	m.running = true
	m.phase = UpdatePhaseChecking
	m.percent = 0
	m.from = from
	m.to = to
	m.message = ""
	m.success = false
	m.rollback = false
	m.startedAt = time.Now()
	m.finishedAt = time.Time{}
	m.events = nil
	m.nextID = 0
	return true
}

// Progress 更新阶段与百分比，并广播给订阅者
func (m *updateTaskManager) Progress(phase string, percent int, message string) {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}

	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	m.phase = phase
	m.percent = percent
	m.message = message
	event := m.appendLocked(UpdateEventProgress, UpdateProgress{Phase: phase, Percent: percent, Message: message})
	targets := m.subscriberSnapshotLocked()
	m.mu.Unlock()

	m.broadcast(targets, event)
}

// Log 追加一行输出（可选），作为进度之外的细节展示
func (m *updateTaskManager) Log(line string) {
	if line == "" {
		return
	}

	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}

	event := m.appendLocked(UpdateEventChunk, UpdateChunk{C: line})
	targets := m.subscriberSnapshotLocked()
	m.mu.Unlock()

	m.broadcast(targets, event)
}

// Complete 结束任务并投递 done 事件
func (m *updateTaskManager) Complete(success, rollback bool, message string) {
	m.mu.Lock()
	if !m.running {
		m.mu.Unlock()
		return
	}

	m.running = false
	m.success = success
	m.rollback = rollback
	m.message = message
	m.finishedAt = time.Now()
	if !success {
		m.phase = UpdatePhaseFailed
	}

	event := UpdateEvent{
		Type: UpdateEventDone,
		ID:   m.nextID,
		Data: UpdateDone{Success: success, Rollback: rollback, From: m.from, To: m.to, Message: message},
	}
	targets := m.subscriberSnapshotLocked()
	m.mu.Unlock()

	for _, sub := range targets {
		sub.push(event)
	}
}

// Busy 是否已有更新任务在跑
func (m *updateTaskManager) Busy() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.running
}

// State 返回任务状态快照
func (m *updateTaskManager) State() UpdateState {
	m.mu.Lock()
	defer m.mu.Unlock()

	state := UpdateState{
		Running:  m.running,
		Phase:    m.phase,
		Percent:  m.percent,
		From:     m.from,
		To:       m.to,
		Success:  m.success,
		Rollback: m.rollback,
		Message:  m.message,
		Offset:   m.nextID,
	}
	if !m.startedAt.IsZero() {
		state.StartedAt = m.startedAt.Unix()
	}
	if !m.finishedAt.IsZero() {
		state.FinishedAt = m.finishedAt.Unix()
	}

	return state
}

// Subscribe 订阅更新进度并补发历史事件。
// after 是客户端已收到的最后一条事件 ID，首次订阅传 0；任务已结束时补发历史后立刻
// 投递 done，刷新页面也能看到最终结果。进程启动后从未执行过任务时返回 nil，
// 调用方应提示「当前没有更新任务」而不是挂起一条空流。
func (m *updateTaskManager) Subscribe(after int64) (int64, <-chan UpdateEvent) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running && m.nextID == 0 {
		return 0, nil
	}

	ch := make(chan UpdateEvent, updateSubBuffer)
	for _, event := range m.events {
		if event.ID > after {
			select {
			case ch <- event:
			default:
			}
		}
	}

	if !m.running {
		ch <- UpdateEvent{
			Type: UpdateEventDone,
			ID:   m.nextID,
			Data: UpdateDone{Success: m.success, Rollback: m.rollback, From: m.from, To: m.to, Message: m.message},
		}
	}

	m.nextSub++
	id := m.nextSub
	m.subs[id] = &updateSubscriber{ch: ch}

	return id, ch
}

// Unsubscribe 解除订阅，关闭后该订阅者的通道会被关闭
func (m *updateTaskManager) Unsubscribe(id int64) {
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

// appendLocked 追加一条事件到缓冲区并返回它，调用方必须持锁
func (m *updateTaskManager) appendLocked(eventType string, data any) UpdateEvent {
	m.nextID++
	event := UpdateEvent{Type: eventType, ID: m.nextID, Data: data}
	m.events = append(m.events, event)
	if over := len(m.events) - updateEventMax; over > 0 {
		n := copy(m.events, m.events[over:])
		m.events = m.events[:n]
	}

	return event
}

// broadcast 把事件投递给订阅者，调用方不能持锁（push 可能阻塞在队列上）
func (m *updateTaskManager) broadcast(targets []*updateSubscriber, event UpdateEvent) {
	for _, sub := range targets {
		sub.push(event)
	}
}

func (m *updateTaskManager) subscriberSnapshotLocked() []*updateSubscriber {
	if len(m.subs) == 0 {
		return nil
	}

	targets := make([]*updateSubscriber, 0, len(m.subs))
	for _, sub := range m.subs {
		targets = append(targets, sub)
	}

	return targets
}
