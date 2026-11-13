package cloud

import (
	"bytes"
	"context"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"encoding/json"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	baseURL      string
	httpClient   *http.Client
	syncClient   *http.Client
	uploadClient *http.Client
	streamClient *http.Client
}

func NewClient() *Client {
	return &Client{
		baseURL:      strings.TrimRight(constant.CloudAPIBaseURL, "/"),
		httpClient:   &http.Client{Timeout: constant.CloudHTTPTimeout},
		syncClient:   &http.Client{Timeout: constant.CloudSyncTimeout},
		uploadClient: &http.Client{Timeout: constant.CloudArchiveTimeout},
		streamClient: &http.Client{Timeout: constant.CloudArchiveTimeout},
	}
}

func (c *Client) newRequest(ctx context.Context, method, path, accessToken string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "dmp/"+constant.Version)
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	return req, nil
}

func (c *Client) requestJSON(ctx context.Context, client *http.Client, method, path, accessToken string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}

	req, err := c.newRequest(ctx, method, path, accessToken, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	resp, err := client.Do(req)
	if err != nil {
		return &APIError{StatusCode: 0, Message: "云服务暂时不可用", Unavailable: true, cause: err}
	}
	defer resp.Body.Close()

	return decodeResponse(resp, out)
}

func decodeResponse(resp *http.Response, out any) error {
	data, err := io.ReadAll(io.LimitReader(resp.Body, constant.CloudMaxResponseBytes+1))
	if err != nil {
		return &APIError{StatusCode: resp.StatusCode, Message: "读取云服务响应失败", Unavailable: resp.StatusCode >= 500, cause: err}
	}
	if len(data) > constant.CloudMaxResponseBytes {
		return &APIError{StatusCode: resp.StatusCode, Message: "云服务响应过大"}
	}

	var envelope apiEnvelope
	unmarshalErr := json.Unmarshal(data, &envelope)
	if resp.StatusCode >= 500 {
		message := "云服务暂时不可用"
		if unmarshalErr == nil && envelope.Message != "" {
			message = envelope.Message
		}
		return &APIError{StatusCode: resp.StatusCode, Code: envelope.Code, Message: message, Unavailable: true}
	}
	if unmarshalErr != nil {
		return &APIError{StatusCode: resp.StatusCode, Message: "云服务响应格式不正确", cause: unmarshalErr}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || (envelope.Code != 200 && envelope.Code != 201) {
		message := envelope.Message
		if message == "" {
			message = "云服务请求失败"
		}
		return &APIError{StatusCode: resp.StatusCode, Code: envelope.Code, Message: message}
	}
	if out != nil && len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		if err = json.Unmarshal(envelope.Data, out); err != nil {
			return &APIError{StatusCode: resp.StatusCode, Message: "解析云服务响应失败", cause: err}
		}
	}

	return nil
}

// doAuthorized 发送需要鉴权的请求。401 时自动刷新一次 token 并重试。
func (c *Client) doAuthorized(ctx context.Context, client *http.Client, link *models.CloudLink, build func(accessToken string) (*http.Request, error)) (*http.Response, error) {
	token, err := c.accessToken(ctx, link)
	if err != nil {
		return nil, err
	}

	req, err := build(token)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, &APIError{Message: "云服务暂时不可用", Unavailable: true, cause: err}
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	_ = resp.Body.Close()

	token, err = c.refreshLink(ctx, link)
	if err != nil {
		return nil, err
	}
	req, err = build(token)
	if err != nil {
		return nil, err
	}
	resp, err = client.Do(req)
	if err != nil {
		return nil, &APIError{Message: "云服务暂时不可用", Unavailable: true, cause: err}
	}

	return resp, nil
}

func (c *Client) authorizedJSON(ctx context.Context, client *http.Client, link *models.CloudLink, method, path string, body any, out any) error {
	var bodyData []byte
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyData = data
	}

	resp, err := c.doAuthorized(ctx, client, link, func(accessToken string) (*http.Request, error) {
		var reader io.Reader
		if bodyData != nil {
			reader = bytes.NewReader(bodyData)
		}
		req, err := c.newRequest(ctx, method, path, accessToken, reader)
		if err != nil {
			return nil, err
		}
		if bodyData != nil {
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
		}
		return req, nil
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return decodeResponse(resp, out)
}
