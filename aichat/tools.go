package aichat

import (
	"bufio"
	"bytes"
	"context"
	"dst-management-platform-api/constant"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ToolFunction 是 OpenAI Chat Completions 风格的工具定义。
type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

// Tool 是一个可被模型调用的函数工具。
type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

// ToolCall 是模型返回的一次工具调用。
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

// ChatOptions 覆盖单次 AI 请求的采样与超时参数。
type ChatOptions struct {
	Temperature float64
	MaxTokens   int
	Timeout     time.Duration
	ToolChoice  any
}

// ChatUsage 是一次 AI 请求的 token 用量。
type ChatUsage struct {
	PromptTokens     int `json:"promptTokens"`
	CompletionTokens int `json:"completionTokens"`
	TotalTokens      int `json:"totalTokens"`
}

// ChatResult 是一次带工具能力的非流式响应。
type ChatResult struct {
	Message Message
	Usage   ChatUsage
	Model   string
}

type apiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type toolCompletionResponse struct {
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
	Usage *apiUsage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

type streamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type streamCompletionRequest struct {
	Model         string         `json:"model"`
	Messages      []Message      `json:"messages"`
	Temperature   float64        `json:"temperature"`
	MaxTokens     int            `json:"max_tokens"`
	Stream        bool           `json:"stream"`
	Tools         []Tool         `json:"tools,omitempty"`
	ToolChoice    any            `json:"tool_choice,omitempty"`
	StreamOptions *streamOptions `json:"stream_options,omitempty"`
}

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage *apiUsage `json:"usage"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ChatWithTools 使用给定配置执行一次带工具定义的非流式对话。
func (c *ChatClient) ChatWithTools(ctx context.Context, config ChatConfig, messages []Message, tools []Tool, opts ChatOptions) (*ChatResult, error) {
	config = config.Normalize()
	temperature := config.Temperature
	if opts.Temperature > 0 {
		temperature = opts.Temperature
	}
	maxTokens := config.MaxTokens
	if opts.MaxTokens > 0 {
		maxTokens = opts.MaxTokens
	}
	timeout := time.Duration(config.RequestTimeoutSeconds) * time.Second
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	requestBody := streamCompletionRequest{
		Model:       config.Model,
		Messages:    messages,
		Temperature: temperature,
		MaxTokens:   maxTokens,
		Stream:      false,
		Tools:       tools,
		ToolChoice:  opts.ToolChoice,
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("序列化大模型请求失败: %w", err)
	}

	endpoint, err := chatCompletionsEndpoint(config.BaseURL)
	if err != nil {
		return nil, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("创建大模型请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("DMP-AI/%s", constant.Version))
	if config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求大模型失败: %w", err)
	}
	defer resp.Body.Close()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, 4*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("读取大模型响应失败: %w", err)
	}

	var result toolCompletionResponse
	if len(responseBody) > 0 {
		if err = json.Unmarshal(responseBody, &result); err != nil {
			if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
				return nil, fmt.Errorf("大模型响应异常 HTTP %d", resp.StatusCode)
			}
			return nil, fmt.Errorf("解析大模型响应失败: %w", err)
		}
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		if result.Error != nil && result.Error.Message != "" {
			return nil, fmt.Errorf("大模型响应异常 HTTP %d: %s", resp.StatusCode, result.Error.Message)
		}
		return nil, fmt.Errorf("大模型响应异常 HTTP %d", resp.StatusCode)
	}
	if result.Error != nil && result.Error.Message != "" {
		return nil, fmt.Errorf("大模型响应异常: %s", result.Error.Message)
	}
	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("大模型响应中没有候选答案")
	}

	chatResult := &ChatResult{
		Message: result.Choices[0].Message,
		Model:   config.Model,
	}
	if result.Usage != nil {
		chatResult.Usage = ChatUsage{
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
		}
	}
	return chatResult, nil
}

// ChatStream 使用给定配置执行一次流式对话，通过 onDelta 回调推送正文增量。
func (c *ChatClient) ChatStream(ctx context.Context, config ChatConfig, messages []Message, opts ChatOptions, onDelta func(string) error) (ChatUsage, error) {
	var usage ChatUsage
	config = config.Normalize()

	temperature := config.Temperature
	if opts.Temperature > 0 {
		temperature = opts.Temperature
	}
	maxTokens := config.MaxTokens
	if opts.MaxTokens > 0 {
		maxTokens = opts.MaxTokens
	}
	timeout := time.Duration(config.RequestTimeoutSeconds) * time.Second
	if opts.Timeout > 0 {
		timeout = opts.Timeout
	}

	requestBody := streamCompletionRequest{
		Model:         config.Model,
		Messages:      messages,
		Temperature:   temperature,
		MaxTokens:     maxTokens,
		Stream:        true,
		Tools:         nil,
		ToolChoice:    "none",
		StreamOptions: &streamOptions{IncludeUsage: true},
	}
	body, err := json.Marshal(requestBody)
	if err != nil {
		return usage, fmt.Errorf("序列化大模型请求失败: %w", err)
	}

	endpoint, err := chatCompletionsEndpoint(config.BaseURL)
	if err != nil {
		return usage, err
	}
	requestCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return usage, fmt.Errorf("创建大模型请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("User-Agent", fmt.Sprintf("DMP-AI/%s", constant.Version))
	if config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+config.APIKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return usage, fmt.Errorf("请求大模型失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
		var apiErr struct {
			Error *struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(responseBody, &apiErr) == nil && apiErr.Error != nil && apiErr.Error.Message != "" {
			return usage, fmt.Errorf("大模型响应异常 HTTP %d: %s", resp.StatusCode, apiErr.Error.Message)
		}
		return usage, fmt.Errorf("大模型响应异常 HTTP %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "[DONE]" {
			break
		}
		var chunk streamChunk
		if err = json.Unmarshal([]byte(payload), &chunk); err != nil {
			continue
		}
		if chunk.Error != nil && chunk.Error.Message != "" {
			return usage, fmt.Errorf("大模型响应异常: %s", chunk.Error.Message)
		}
		if chunk.Usage != nil {
			usage = ChatUsage{
				PromptTokens:     chunk.Usage.PromptTokens,
				CompletionTokens: chunk.Usage.CompletionTokens,
				TotalTokens:      chunk.Usage.TotalTokens,
			}
		}
		for _, choice := range chunk.Choices {
			if choice.Delta.Content == "" {
				continue
			}
			if err = onDelta(choice.Delta.Content); err != nil {
				return usage, err
			}
		}
	}
	if err = scanner.Err(); err != nil {
		return usage, fmt.Errorf("读取大模型流式响应失败: %w", err)
	}
	return usage, nil
}
