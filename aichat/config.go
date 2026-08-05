package aichat

import (
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/utils"
	"fmt"
	"strings"
	"unicode/utf8"
)

func validateModelConfig(config *models.AIModelConfig) error {
	config.ChatBaseURL = strings.TrimSpace(config.ChatBaseURL)
	config.ChatModel = strings.TrimSpace(config.ChatModel)
	config.EmbeddingModel = strings.TrimSpace(config.EmbeddingModel)

	if config.ChatBaseURL == "" || !utils.IsValidURL(config.ChatBaseURL) {
		return fmt.Errorf("大模型 Base URL 不合法")
	}
	if config.ChatModel == "" {
		return fmt.Errorf("大模型名称不能为空")
	}
	if utf8.RuneCountInString(config.ChatModel) > 256 {
		return fmt.Errorf("大模型名称过长")
	}
	if len(config.ChatApiKey) > 16*1024 {
		return fmt.Errorf("API Key 过长")
	}
	if config.EmbeddingModel != "" {
		if utf8.RuneCountInString(config.EmbeddingModel) > 256 {
			return fmt.Errorf("嵌入模型名称过长")
		}
		if len(config.EmbeddingApiKey) > 16*1024 {
			return fmt.Errorf("嵌入 API Key 过长")
		}
	}
	if utf8.RuneCountInString(config.EmbeddingBaseURL) > 8000 {
		return fmt.Errorf("系统提示词过长")
	}
	if config.Temperature < 0 || config.Temperature > 2 {
		return fmt.Errorf("temperature 必须在 0 到 2 之间")
	}
	if config.MaxTokens <= 0 || config.MaxTokens > 32768 {
		return fmt.Errorf("maxTokens 必须在 1 到 32768 之间")
	}
	if config.RequestTimeoutSeconds <= 0 || config.RequestTimeoutSeconds > 300 {
		return fmt.Errorf("请求超时时间必须在 1 到 300 秒之间")
	}

	return nil
}

func validateRoomSetting(setting *models.RoomAISetting) error {
	if setting.RoomID <= 0 {
		return fmt.Errorf("房间 ID 不合法")
	}
	if strings.ContainsAny(setting.Prefix, "\r\n") || utf8.RuneCountInString(setting.Prefix) > 64 {
		return fmt.Errorf("AI 对话前缀不合法")
	}
	if setting.ContextMaxMessages < 2 || setting.ContextMaxMessages > 100 {
		return fmt.Errorf("上下文消息数量必须在 2 到 100 之间")
	}
	if setting.ContextTTLMinutes <= 0 || setting.ContextTTLMinutes > 10080 {
		return fmt.Errorf("上下文有效期必须在 1 到 10080 分钟之间")
	}

	return ValidateModelConfig(&setting.AIModelConfig)
}
