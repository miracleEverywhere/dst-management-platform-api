package models

type AIModelConfig struct {
	ChatBaseURL           string  `json:"chatBaseURL"`
	ChatApiKey            string  `json:"chatApiKey"`
	ChatModel             string  `json:"chatModel"`
	EmbeddingApiKey       string  `json:"embeddingApiKey"`
	EmbeddingModel        string  `json:"embeddingModel"`
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
	RoomID  int    `gorm:"primaryKey;not null;column:room_id" json:"roomID"`
	Enabled bool   `gorm:"column:enabled" json:"enabled"`
	Prefix  string `gorm:"column:prefix" json:"prefix"`
}

func (RoomAISetting) TableName() string {
	return "room_ai_settings"
}

// AIChatSetting 是 Manager 使用的完整运行时配置，不直接接收 API 请求。
type AIChatSetting struct {
	RoomAISetting
	AIBaseSetting
}
