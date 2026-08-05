package models

type AIModelConfig struct {
	ChatBaseURL           string  `gorm:"column:chat_base_url" json:"chatBaseURL"`
	ChatApiKey            string  `gorm:"column:chat_api_key" json:"chatApiKey"`
	ChatModel             string  `gorm:"column:chat_model" json:"chat_model"`
	EmbeddingBaseURL      string  `gorm:"column:embedding_base_url" json:"embeddingBaseURL"`
	EmbeddingApiKey       string  `gorm:"column:embedding_api_key" json:"embeddingApiKEY"`
	EmbeddingModel        string  `gorm:"column:embedding_model" json:"embeddingModel"`
	Temperature           float64 `gorm:"column:temperature" json:"temperature"`
	MaxTokens             int     `gorm:"column:max_tokens" json:"maxTokens"`
	RequestTimeoutSeconds int     `gorm:"column:request_timeout_seconds" json:"requestTimeoutSeconds"`
}

type RoomAISetting struct {
	RoomID             int `gorm:"primaryKey;not null;column:room_id" json:"roomID"`
	AIModelConfig      `gorm:"embedded"`
	Enabled            bool   `gorm:"column:enabled" json:"enabled"`
	Prefix             string `gorm:"column:prefix" json:"prefix"`
	ContextMaxMessages int    `gorm:"column:context_max_messages" json:"contextMaxMessages"`
	ContextTTLMinutes  int    `gorm:"column:context_ttl_minutes" json:"contextTTLMinutes"`
}

func (RoomAISetting) TableName() string {
	return "room_ai_settings"
}
