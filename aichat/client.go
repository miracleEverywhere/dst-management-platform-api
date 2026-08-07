package aichat

import (
	"bytes"
	"context"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature"`
	MaxTokens   int           `json:"max_tokens"`
	Stream      bool          `json:"stream"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type aiClient struct {
	httpClient *http.Client
}

func newClient() *aiClient {
	return &aiClient{
		httpClient: &http.Client{
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
	}
}

func (c *aiClient) complete(ctx context.Context, config models.AIModelConfig, messages []chatMessage) (string, error) {
	requestBody := chatCompletionRequest{
		Model:       config.ChatModel,
		Messages:    messages,
		Temperature: config.Temperature,
		MaxTokens:   config.MaxTokens,
		Stream:      false,
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("序列化大模型请求失败: %w", err)
	}

	endpoint, err := chatCompletionsEndpoint(config.ChatBaseURL)
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
	req.Header.Set("User-Agent", fmt.Sprintf("DMP-AI/%s", utils.Version))
	if config.ChatApiKey != "" {
		req.Header.Set("Authorization", "Bearer "+config.ChatApiKey)
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
