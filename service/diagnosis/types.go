// Package diagnosis 提供 AI 诊断、修复动作授权与执行能力。
//
// 本包不依赖 gin，也不直接写 HTTP 响应；SSE 由 app/diagnosis 负责传输。
package diagnosis

import (
	"dst-management-platform-api/aichat"
	"time"
)

const (
	ScopeRoom     = "room"
	ScopePlatform = "platform"

	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
	StatusCanceled  = "canceled"

	ActionStatusProposed  = "proposed"
	ActionStatusRunning   = "running"
	ActionStatusSucceeded = "succeeded"
	ActionStatusFailed    = "failed"
	ActionStatusSkipped   = "skipped"

	// ApplyStatusPartial 表示修复动作都执行了，但有动作复检未通过。
	ApplyStatusPartial = "partial"

	RiskLow    = "low"
	RiskMedium = "medium"
	RiskHigh   = "high"

	ConfirmNone   = "none"
	ConfirmNormal = "normal"
	ConfirmTyped  = "typed"

	EventSession     = "session"
	EventStep        = "step"
	EventNotice      = "notice"
	EventReportMeta  = "report_meta"
	EventReportStart = "report_start"
	EventReportDelta = "report_delta"
	EventReportEnd   = "report_end"
	EventDone        = "done"
	EventError       = "error"
	EventApplyStart  = "apply_start"
	EventActionStart = "action_start"
	EventActionDone  = "action_done"
	EventApplyEnd    = "apply_end"
)

// StartInput 是创建诊断的请求体。
// Description 是用户对异常现象的可选描述，作为排查线索传给模型。
type StartInput struct {
	Scope       string `json:"scope"`
	RoomID      int    `json:"roomID"`
	Description string `json:"description"`
}

// SSEEvent 是推送给前端的一条 SSE 事件。
type SSEEvent struct {
	ID   int64  `json:"-"`
	Type string `json:"-"`
	Data any    `json:"data"`
}

// Step 是一次工具调用的进度。
type Step struct {
	Index      int       `json:"index"`
	Tool       string    `json:"tool"`
	Status     string    `json:"status"`
	StartedAt  time.Time `json:"startedAt"`
	DurationMs int64     `json:"durationMs"`
	Error      string    `json:"error,omitempty"`
}

// Evidence 是一次工具调用的可追溯证据。
type Evidence struct {
	EvidenceID string    `json:"evidenceID"`
	Tool       string    `json:"tool"`
	Source     string    `json:"source"`
	Summary    string    `json:"summary"`
	Detail     string    `json:"detail"`
	Truncated  bool      `json:"truncated"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Finding 是诊断报告中的一条发现。
type Finding struct {
	FindingID   string      `json:"findingID"`
	Title       string      `json:"title"`
	Severity    string      `json:"severity"`
	Confidence  float64     `json:"confidence"`
	KBID        string      `json:"kbID,omitempty"`
	Cause       string      `json:"cause,omitempty"`
	Solutions   []string    `json:"solutions,omitempty"`
	EvidenceIDs []string    `json:"evidenceIDs"`
	Evidence    []*Evidence `json:"evidence,omitempty"`
}

// Action 是诊断报告中的一个可执行修复动作。
type Action struct {
	ActionID          string         `json:"actionID"`
	TitleKey          string         `json:"titleKey"`
	Risk              string         `json:"risk"`
	Reversible        bool           `json:"reversible"`
	AdminOnly         bool           `json:"adminOnly"`
	ConfirmationLevel string         `json:"confirmationLevel"`
	ConfirmationHint  string         `json:"confirmationHint,omitempty"`
	Params            map[string]any `json:"params"`
	Preview           string         `json:"preview"`
	Reason            string         `json:"reason"`
	EvidenceIDs       []string       `json:"evidenceIDs"`
	Order             int            `json:"order"`
	Status            string         `json:"status"`
	ApplyResult       *ActionResult  `json:"applyResult,omitempty"`
}

// Reference 是报告中的文档链接。
type Reference struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

// Report 是一次诊断的完整结果。
type Report struct {
	Summary    string           `json:"summary"`
	Severity   string           `json:"severity"`
	Markdown   string           `json:"markdown"`
	Findings   []*Finding       `json:"findings"`
	Actions    []*Action        `json:"actions"`
	References []Reference      `json:"references"`
	Model      string           `json:"model"`
	Usage      aichat.ChatUsage `json:"usage"`
}

// ActionResult 是单个修复动作的执行结果。
type ActionResult struct {
	ActionID      string `json:"actionID"`
	Status        string `json:"status"`
	Code          int    `json:"code,omitempty"`
	Message       string `json:"message,omitempty"`
	Verified      bool   `json:"verified"`
	VerifyMessage string `json:"verifyMessage,omitempty"`
	DurationMs    int64  `json:"durationMs"`
}

// ApplyState 是一次修复任务的整体状态。
type ApplyState struct {
	ApplyID    string         `json:"applyID"`
	Status     string         `json:"status"`
	StartedAt  time.Time      `json:"startedAt"`
	FinishedAt *time.Time     `json:"finishedAt"`
	Results    []ActionResult `json:"results"`
}

// ApplyInput 是执行修复的请求体。
type ApplyInput struct {
	ActionIDs        []string                  `json:"actionIDs"`
	Confirm          bool                      `json:"confirm"`
	ConfirmationText string                    `json:"confirmationText"`
	ParamOverrides   map[string]map[string]any `json:"paramOverrides"`
}

// SessionSnapshot 是 GET /diagnosis/:id 返回的数据。
type SessionSnapshot struct {
	DiagnosisID string      `json:"diagnosisID"`
	Scope       string      `json:"scope"`
	RoomID      int         `json:"roomID"`
	RoomName    string      `json:"roomName"`
	Description string      `json:"description"`
	Status      string      `json:"status"`
	CreatedAt   time.Time   `json:"createdAt"`
	FinishedAt  *time.Time  `json:"finishedAt"`
	Steps       []Step      `json:"steps"`
	Report      *Report     `json:"report,omitempty"`
	CanApply    bool        `json:"canApply"`
	Apply       *ApplyState `json:"apply,omitempty"`
	Error       string      `json:"error,omitempty"`
}

// StartData 是创建诊断的响应 data。
type StartData struct {
	DiagnosisID string `json:"diagnosisID"`
	Scope       string `json:"scope"`
	Status      string `json:"status"`
	EventsURL   string `json:"eventsURL"`
}

// ApplyData 是开始修复的响应 data。
type ApplyData struct {
	ApplyID   string `json:"applyID"`
	Status    string `json:"status"`
	EventsURL string `json:"eventsURL"`
}

// HistoryRow 是历史列表中的一行。
type HistoryRow struct {
	DiagnosisID  string     `json:"diagnosisID"`
	Scope        string     `json:"scope"`
	RoomID       int        `json:"roomID"`
	RoomName     string     `json:"roomName"`
	Description  string     `json:"description"`
	Username     string     `json:"username"`
	Status       string     `json:"status"`
	Severity     string     `json:"severity"`
	Summary      string     `json:"summary"`
	Model        string     `json:"model"`
	ActionCount  int        `json:"actionCount"`
	AppliedCount int        `json:"appliedCount"`
	CreatedAt    time.Time  `json:"createdAt"`
	FinishedAt   *time.Time `json:"finishedAt"`
}

// EventError 是 SSE error 事件的数据。
type ErrorEventData struct {
	Code      int    `json:"code"`
	Message   string `json:"message,omitempty"`
	Retryable bool   `json:"retryable"`
}

// EventNotice 是 SSE notice 事件的数据。
type NoticeEventData struct {
	Level   string `json:"level"`
	Message string `json:"message"`
}
