package opmgr

import (
	"errors"
	"fmt"
	"sync"
	"time"
)

// 包级操作管理器：同一房间同一时刻只允许一个启停类操作，
// 操作状态保存在内存中（页面刷新后可通过 Current/History 恢复展示），
// 完成的操作通过 SetPersist 注入的回调落库。

type Type string

const (
	TypeStartup        Type = "startup"
	TypeShutdown       Type = "shutdown"
	TypeRestart        Type = "restart"
	TypeUpdate         Type = "update"
	TypeReset          Type = "reset"
	TypeRestore        Type = "restore"
	TypeDeleteSnapshot Type = "deleteSnapshot"
	TypeActivate       Type = "activate"
	TypeDeactivate     Type = "deactivate"
)

func (t Type) Label() string {
	switch t {
	case TypeStartup:
		return "启动游戏"
	case TypeShutdown:
		return "关闭游戏"
	case TypeRestart:
		return "重启游戏"
	case TypeUpdate:
		return "更新游戏"
	case TypeReset:
		return "重置世界"
	case TypeRestore:
		return "恢复备份"
	case TypeDeleteSnapshot:
		return "删除存档快照"
	case TypeActivate:
		return "激活房间"
	case TypeDeactivate:
		return "停用房间"
	}
	return string(t)
}

type State string

const (
	StateRunning State = "running"
	StateSuccess State = "success"
	StateFailed  State = "failed"
)

type Operation struct {
	ID        string     `json:"id"`
	RoomID    int        `json:"roomID"`
	Type      Type       `json:"type"`
	State     State      `json:"state"`
	Stage     string     `json:"stage"`
	Error     string     `json:"error,omitempty"`
	StartedAt time.Time  `json:"startedAt"`
	EndedAt   *time.Time `json:"endedAt,omitempty"`
	By        string     `json:"by,omitempty"`
}

// ProgressFunc 操作进度回调：在后台任务执行的各阶段被调用，更新任务状态
type ProgressFunc func(stage string)

var (
	ErrConflict = errors.New("该房间已有正在执行的操作")

	mu          sync.Mutex
	current     = map[int]*Operation{}
	history     = map[int][]*Operation{}
	persistHook func(*Operation)
)

// Submit 注册房间操作并异步执行 fn。若该房间已有进行中的操作，返回冲突错误
// 与当前操作的快照（用于拼装提示）。
func Submit(roomID int, t Type, by string, fn func(progress ProgressFunc) error) (*Operation, error) {
	mu.Lock()
	if busy, ok := current[roomID]; ok {
		mu.Unlock()
		return clone(busy), fmt.Errorf("%w: %s（已进行 %d 秒），请等待其完成",
			ErrConflict, busy.Type.Label(), int(time.Since(busy.StartedAt).Seconds()))
	}
	op := &Operation{
		ID:        fmt.Sprintf("%d-%s-%d", roomID, t, time.Now().UnixMilli()),
		RoomID:    roomID,
		Type:      t,
		State:     StateRunning,
		Stage:     "已受理，正在执行",
		StartedAt: time.Now(),
		By:        by,
	}
	current[roomID] = op
	mu.Unlock()
	go execute(op, fn)
	return clone(op), nil
}

func execute(op *Operation, fn func(ProgressFunc) error) {
	defer func() {
		if r := recover(); r != nil {
			finish(op, StateFailed, fmt.Sprintf("内部错误: %v", r))
		}
	}()
	err := fn(func(stage string) {
		mu.Lock()
		op.Stage = stage
		mu.Unlock()
	})
	if err != nil {
		finish(op, StateFailed, err.Error())
		return
	}
	finish(op, StateSuccess, "")
}

func finish(op *Operation, state State, errMsg string) {
	now := time.Now()
	mu.Lock()
	op.State = state
	op.Error = errMsg
	op.EndedAt = &now
	delete(current, op.RoomID)
	h := append(history[op.RoomID], clone(op))
	if len(h) > 50 {
		h = h[len(h)-50:]
	}
	history[op.RoomID] = h
	hook := persistHook
	mu.Unlock()
	if hook != nil {
		go func(opCopy Operation) {
			defer func() { _ = recover() }()
			hook(&opCopy)
		}(*clone(op))
	}
}

func clone(op *Operation) *Operation {
	c := *op
	return &c
}

// Current 返回房间正在执行的操作快照，没有则返回 nil
func Current(roomID int) *Operation {
	mu.Lock()
	defer mu.Unlock()
	if op, ok := current[roomID]; ok {
		return clone(op)
	}
	return nil
}

// History 返回房间最近 limit 条已完成的操作（新的在前）
func History(roomID int, limit int) []*Operation {
	mu.Lock()
	defer mu.Unlock()
	h := history[roomID]
	if limit <= 0 || limit > len(h) {
		limit = len(h)
	}
	out := make([]*Operation, 0, limit)
	for i := len(h) - 1; i >= len(h)-limit; i-- {
		out = append(out, clone(h[i]))
	}
	return out
}

// SetPersist 注册操作完成后的持久化回调（如写入数据库）
func SetPersist(hook func(*Operation)) {
	persistHook = hook
}
