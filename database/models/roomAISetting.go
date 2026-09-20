package models

const (
	// DefaultAIWikiMaxResults 是房间未配置最大返回文档数时使用的默认值。
	DefaultAIWikiMaxResults = 10
	// MinAIReplyLength 是 AI 回复字数允许的最小值。
	MinAIReplyLength = 100
	// MaxAIReplyLength 是 AI 回复字数允许的最大值。
	MaxAIReplyLength = 300
	// DefaultAIReplyMaxLength 是房间未配置 AI 回复字数时使用的默认值。
	DefaultAIReplyMaxLength = 200
)

type AIModelConfig struct {
	ChatBaseURL      string `json:"chatBaseURL"`
	ChatApiKey       string `json:"chatApiKey"`
	ChatModel        string `json:"chatModel"`
	EmbeddingBaseURL string `json:"embeddingBaseURL"`
	EmbeddingApiKey  string `json:"embeddingApiKey"`
	EmbeddingModel   string `json:"embeddingModel"`

	// EmbeddingDimensions 是向量维度。0 表示不向 embedding API 发送 dimensions 参数，
	// 由模型自行决定原生维度（BGE-M3 等模型不支持该参数，必须保持 0）。
	EmbeddingDimensions int `json:"embeddingDimensions"`

	SystemPrompt          string  `json:"systemPrompt"`
	Temperature           float64 `json:"temperature"`
	MaxTokens             int     `json:"maxTokens"`
	RequestTimeoutSeconds int     `json:"requestTimeoutSeconds"`
}

// AIBaseSetting 是全局 AI 基础配置，只允许管理员修改。
type AIBaseSetting struct {
	AIModelConfig
	ContextMaxMessages int `json:"contextMaxMessages"`
	ContextTTLMinutes  int `json:"contextTTLMinutes"`
}

// DefaultAIBaseSetting 返回尚未配置模型时使用的表单默认值。
func DefaultAIBaseSetting() AIBaseSetting {
	return AIBaseSetting{
		AIModelConfig: AIModelConfig{
			Temperature:           0.7,
			MaxTokens:             512,
			RequestTimeoutSeconds: 60,
		},
		ContextMaxMessages: 10,
		ContextTTLMinutes:  30,
	}
}

// RoomAISetting 是用户可修改的房间级配置。
type RoomAISetting struct {
	RoomID         int    `gorm:"primaryKey;not null;column:room_id" json:"roomID"`
	Enabled        bool   `gorm:"column:enabled" json:"enabled"`
	AllowChat      bool   `gorm:"column:allow_chat" json:"allowChat"`
	Prefix         string `gorm:"column:prefix" json:"prefix"`
	MaxResults     int    `gorm:"column:max_results;default:10" json:"maxResults"`
	MaxReplyLength int    `gorm:"column:max_reply_length;default:200" json:"maxReplyLength"`
}

func (RoomAISetting) TableName() string {
	return "room_ai_settings"
}

// AIChatSetting 是 Manager 使用的完整运行时配置，不直接接收 API 请求。
type AIChatSetting struct {
	RoomAISetting
	AIBaseSetting
}
