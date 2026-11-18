package aichat

import (
	"bytes"
	"context"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"dst-management-platform-api/utils"
)

// ChatConfig 是一次大模型对话所需的配置，AI 对话与 AI 诊断共用。
type ChatConfig struct {
	BaseURL               string
	APIKey                string
	Model                 string
	Temperature           float64
	MaxTokens             int
	RequestTimeoutSeconds int
}

// ChatConfigFromModel 从数据库配置构造 ChatConfig。
func ChatConfigFromModel(config models.AIModelConfig) ChatConfig {
	return ChatConfig{
		BaseURL:               config.ChatBaseURL,
		APIKey:                config.ChatApiKey,
		Model:                 config.ChatModel,
		Temperature:           config.Temperature,
		MaxTokens:             config.MaxTokens,
		RequestTimeoutSeconds: config.RequestTimeoutSeconds,
	}.Normalize()
}

// Normalize 去掉配置中的首尾空白。
func (c ChatConfig) Normalize() ChatConfig {
	c.BaseURL = strings.TrimSpace(c.BaseURL)
	c.APIKey = strings.TrimSpace(c.APIKey)
	c.Model = strings.TrimSpace(c.Model)
	return c
}

// ValidateEndpoint 只校验连接所需的字段。
// 采样参数与超时由调用方的 ChatOptions 指定时，不需要再要求 ChatConfig 携带这些值。
func (c ChatConfig) ValidateEndpoint() error {
	c = c.Normalize()
	if c.BaseURL == "" || !utils.IsValidURL(c.BaseURL) {
		return fmt.Errorf("大模型 Base URL 不合法")
	}
	if c.Model == "" {
		return fmt.Errorf("大模型名称不能为空")
	}
	if utf8.RuneCountInString(c.Model) > 256 {
		return fmt.Errorf("大模型名称过长")
	}
	if len(c.APIKey) > 16*1024 {
		return fmt.Errorf("API Key 过长")
	}
	return nil
}

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type chatCompletionRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Temperature float64   `json:"temperature"`
	MaxTokens   int       `json:"max_tokens"`
	Stream      bool      `json:"stream"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ChatClient 是 OpenAI Chat Completions 兼容客户端，配置由调用方传入。
// AI 对话与 AI 诊断各自解析配置后复用同一份请求逻辑。
type ChatClient struct {
	httpClient *http.Client
}

// NewChatClient 创建大模型客户端。
func NewChatClient() *ChatClient {
	return &ChatClient{
		httpClient: &http.Client{
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (c *ChatClient) complete(ctx context.Context, config ChatConfig, messages []Message) (string, error) {
	config = config.Normalize()
	requestBody := chatCompletionRequest{
		Model:       config.Model,
		Messages:    messages,
		Temperature: config.Temperature,
		MaxTokens:   config.MaxTokens,
		Stream:      false,
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("序列化大模型请求失败: %w", err)
	}

	endpoint, err := chatCompletionsEndpoint(config.BaseURL)
	if err != nil {
		return "", err
	}
	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(config.RequestTimeoutSeconds)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("创建大模型请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("DMP-AI/%s", constant.Version))
	if config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("请求大模型失败: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
	if err != nil {
		return "", fmt.Errorf("读取大模型响应失败: %w", err)
	}

	var result chatCompletionResponse
	if len(responseBody) > 0 {
		if err = json.Unmarshal(responseBody, &result); err != nil {
			if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
				return "", fmt.Errorf("大模型响应异常 HTTP %d", resp.StatusCode)
			}
			return "", fmt.Errorf("解析大模型响应失败: %w", err)
		}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if result.Error != nil && result.Error.Message != "" {
			return "", fmt.Errorf("大模型响应异常 HTTP %d: %s", resp.StatusCode, result.Error.Message)
		}
		return "", fmt.Errorf("大模型响应异常 HTTP %d", resp.StatusCode)
	}
	if result.Error != nil && result.Error.Message != "" {
		return "", fmt.Errorf("大模型响应异常: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("大模型响应中没有候选答案")
	}

	answer := strings.TrimSpace(result.Choices[0].Message.Content)
	if answer == "" {
		return "", fmt.Errorf("大模型返回了空答案")
	}
	return answer, nil
}

func chatCompletionsEndpoint(baseURL string) (string, error) {
	u, err := url.Parse(strings.TrimSpace(baseURL))
	if err != nil {
		return "", fmt.Errorf("解析大模型 Base URL 失败: %w", err)
	}

	path := strings.TrimRight(u.Path, "/")
	if !strings.HasSuffix(path, "/chat/completions") {
		if path == "" {
			path = "/chat/completions"
		} else {
			path += "/chat/completions"
		}
	}
	u.Path = path
	return u.String(), nil
}
