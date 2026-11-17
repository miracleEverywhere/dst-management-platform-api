package models

import (
	"strings"
	"time"
)

// DiagnosisRecord 保存一次 AI 诊断的最终结果，用于历史查询与审计。
type DiagnosisRecord struct {
	ID               string     `gorm:"primaryKey;column:id" json:"id"`
	Scope            string     `gorm:"column:scope;index:idx_diagnosis_scope" json:"scope"`
	RoomID           int        `gorm:"column:room_id;index:idx_diagnosis_room" json:"roomID"`
	RoomName         string     `gorm:"column:room_name" json:"roomName"`
	Username         string     `gorm:"column:username;index:idx_diagnosis_user" json:"username"`
	Status           string     `gorm:"column:status;index:idx_diagnosis_status" json:"status"`
	Severity         string     `gorm:"column:severity" json:"severity"`
	Summary          string     `gorm:"column:summary;type:text" json:"summary"`
	Report           string     `gorm:"column:report;type:text" json:"report"`
	Findings         string     `gorm:"column:findings;type:text" json:"findings"`
	References       string     `gorm:"column:references_json;type:text" json:"references"`
	Actions          string     `gorm:"column:actions;type:text" json:"actions"`
	Model            string     `gorm:"column:model" json:"model"`
	PromptTokens     int        `gorm:"column:prompt_tokens" json:"promptTokens"`
	CompletionTokens int        `gorm:"column:completion_tokens" json:"completionTokens"`
	Error            string     `gorm:"column:error;type:text" json:"error"`
	CreatedAt        time.Time  `gorm:"column:created_at;index:idx_diagnosis_created" json:"createdAt"`
	FinishedAt       *time.Time `gorm:"column:finished_at" json:"finishedAt"`
}

func (DiagnosisRecord) TableName() string {
	return "diagnosis_records"
}

// DiagnosisActionRecord 保存一次修复动作的授权与执行结果。
type DiagnosisActionRecord struct {
	ID          uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	DiagnosisID string     `gorm:"column:diagnosis_id;index:idx_diagnosis_action" json:"diagnosisID"`
	ActionID    string     `gorm:"column:action_id" json:"actionID"`
	Params      string     `gorm:"column:params;type:text" json:"params"`
	Risk        string     `gorm:"column:risk" json:"risk"`
	Status      string     `gorm:"column:status" json:"status"`
	Code        int        `gorm:"column:code" json:"code"`
	Message     string     `gorm:"column:message;type:text" json:"message"`
	Verified    bool       `gorm:"column:verified" json:"verified"`
	Operator    string     `gorm:"column:operator" json:"operator"`
	CreatedAt   time.Time  `gorm:"column:created_at;index:idx_diagnosis_action_created" json:"createdAt"`
	FinishedAt  *time.Time `gorm:"column:finished_at" json:"finishedAt"`
}

func (DiagnosisActionRecord) TableName() string {
	return "diagnosis_action_records"
}

// DiagnosisAISetting 是 AI 诊断专用的 LLM 配置。
// chatBaseURL/chatModel/chatApiKey 都未填写时回退到 AI 对话的全局基础配置。
type DiagnosisAISetting struct {
	ChatBaseURL string `json:"chatBaseURL"`
	ChatApiKey  string `json:"chatApiKey"`
	ChatModel   string `json:"chatModel"`
}

// Configured 表示管理员是否填写了诊断专用配置。
// 只填了一部分也算已配置，此时会按配置不完整报错，而不是静默回退。
func (s DiagnosisAISetting) Configured() bool {
	return strings.TrimSpace(s.ChatBaseURL) != "" ||
		strings.TrimSpace(s.ChatModel) != "" ||
		strings.TrimSpace(s.ChatApiKey) != ""
}
