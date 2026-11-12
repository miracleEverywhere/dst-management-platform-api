package cloud

import (
	"bytes"
	"context"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ===== 云服务客户端 =====

// ErrRelink 表示云凭据已失效或无法解密，需要用户重新绑定云账号。
var ErrRelink = errors.New("cloud relink required")

// APIError 是云服务返回的错误。Unavailable 为 true 时表示网络错误、超时或 5xx，
// 只有这类错误允许在云登录时回退到本地账号。
type APIError struct {
	StatusCode  int
	Code        int
	Message     string
	Unavailable bool
	cause       error
}

func (e *APIError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("cloud api %d: %s: %v", e.StatusCode, e.Message, e.cause)
	}

	return fmt.Sprintf("cloud api %d: %s", e.StatusCode, e.Message)
}

func (e *APIError) Unwrap() error {
	return e.cause
}

func IsUnavailable(err error) bool {
	var target *APIError
	return errors.As(err, &target) && target.Unavailable
}

func IsUnauthorized(err error) bool {
	var target *APIError
	return errors.As(err, &target) && target.StatusCode == http.StatusUnauthorized
}

func IsForbidden(err error) bool {
	var target *APIError
	return errors.As(err, &target) && target.StatusCode == http.StatusForbidden
}

func MessageOf(err error) string {
	var target *APIError
	if errors.As(err, &target) && target.Message != "" {
		return target.Message
	}

	return ""
}

type apiEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type User struct {
	ID        int64           `json:"id"`
	Username  string          `json:"username"`
	Email     string          `json:"email"`
	Role      string          `json:"role"`
	IsActive  bool            `json:"isActive"`
	CreatedAt json.RawMessage `json:"createdAt"`
}

type Account struct {
	ID            int64           `json:"id"`
	Username      string          `json:"username"`
	Email         string          `json:"email"`
	Role          string          `json:"role"`
	Plan          string          `json:"plan"`
	Status        string          `json:"status"`
	FavoriteQuota int             `json:"favoriteQuota"`
	CreatedAt     json.RawMessage `json:"createdAt"`
	LastLoginAt   json.RawMessage `json:"lastLoginAt"`
}

type Plan struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	Price          int    `json:"price"`
	FavoriteQuota  int    `json:"favoriteQuota"`
	TokenQuota     int    `json:"tokenQuota"`
	GameAdminQuota int    `json:"gameAdminQuota"`
	GameSlotQuota  int    `json:"gameSlotQuota"`
	ArchiveQuota   int    `json:"archiveQuota"`
}

type TokenPair struct {
	AccessToken           string `json:"accessToken"`
	RefreshToken          string `json:"refreshToken"`
	AccessTokenExpiresAt  int64  `json:"accessTokenExpiresAt"`
	RefreshTokenExpiresAt int64  `json:"refreshTokenExpiresAt"`
}

type AuthResult struct {
	User           User      `json:"user"`
	Account        Account   `json:"account"`
	Plan           Plan      `json:"plan"`
	Tokens         TokenPair `json:"tokens"`
	CloudActivated bool      `json:"cloudActivated"`
	FavoriteQuota  int       `json:"favoriteQuota"`
	FavoriteCount  int       `json:"favoriteCount"`
	TokenCount     int       `json:"tokenCount"`
}

type MeResult struct {
	Account       Account `json:"account"`
	Plan          Plan    `json:"plan"`
	FavoriteQuota int     `json:"favoriteQuota"`
	FavoriteCount int     `json:"favoriteCount"`
	TokenCount    int     `json:"tokenCount"`
}

type KleiToken struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Token     string `json:"token"`
	Remark    string `json:"remark"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type TokenList struct {
	Items      []KleiToken `json:"items"`
	TokenQuota int         `json:"tokenQuota"`
	Used       int         `json:"used"`
}

type GameEntry struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	UID       string `json:"uid"`
	Nickname  string `json:"nickname"`
	Remark    string `json:"remark"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type GameEntryList struct {
	Items []GameEntry `json:"items"`
	Kind  string      `json:"kind"`
	Quota int         `json:"quota"`
	Used  int         `json:"used"`
}

type Archive struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	FileName         string `json:"fileName"`
	Remark           string `json:"remark"`
	SizeBytes        int64  `json:"sizeBytes"`
	Checksum         string `json:"checksum"`
	CreatedAt        int64  `json:"createdAt"`
	UpdatedAt        int64  `json:"updatedAt"`
	LastDownloadedAt *int64 `json:"lastDownloadedAt"`
	DownloadCount    int    `json:"downloadCount"`
}

type ArchiveList struct {
	Items    []Archive `json:"items"`
	Quota    int       `json:"quota"`
	Used     int       `json:"used"`
	MaxBytes int64     `json:"maxBytes"`
}

type FavoriteMeta struct {
	RoomName   string `json:"roomName"`
	GameMode   string `json:"gameMode"`
	WorldCount int    `json:"worldCount"`
	ModCount   int    `json:"modCount"`
}

type Favorite struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Remark          string          `json:"remark"`
	Snapshot        json.RawMessage `json:"snapshot"`
	LocalFavoriteID string          `json:"localFavoriteId"`
	Source          string          `json:"source"`
	CreatedAt       int64           `json:"createdAt"`
	UpdatedAt       int64           `json:"updatedAt"`
	Deleted         bool            `json:"deleted"`
	Meta            FavoriteMeta    `json:"meta"`
}

type FavoritePage struct {
	Items      []Favorite `json:"items"`
	Page       int        `json:"page"`
	PageSize   int        `json:"pageSize"`
	Total      int64      `json:"total"`
	TotalPages int        `json:"totalPages"`
}

type SyncItem struct {
	CloudID   string          `json:"cloudId,omitempty"`
	LocalID   string          `json:"localId,omitempty"`
	Name      string          `json:"name,omitempty"`
	Remark    string          `json:"remark,omitempty"`
	Snapshot  json.RawMessage `json:"snapshot,omitempty"`
	CreatedAt int64           `json:"createdAt,omitempty"`
	UpdatedAt int64           `json:"updatedAt,omitempty"`
	Deleted   bool            `json:"deleted,omitempty"`
}

type SyncRequest struct {
	DeviceID  string     `json:"deviceId"`
	Since     int64      `json:"since"`
	Favorites []SyncItem `json:"favorites"`
}

type SyncSkip struct {
	LocalID string `json:"localId"`
	Name    string `json:"name"`
	Reason  string `json:"reason"`
}

type DeletedRef struct {
	ID        string `json:"id"`
	LocalID   string `json:"localId"`
	Name      string `json:"name"`
	UpdatedAt int64  `json:"updatedAt"`
}

type SyncSummary struct {
	Uploaded      int `json:"uploaded"`
	Updated       int `json:"updated"`
	Deleted       int `json:"deleted"`
	Skipped       int `json:"skipped"`
	QuotaExceeded int `json:"quotaExceeded"`
	Conflicts     int `json:"conflicts"`
	Downloaded    int `json:"downloaded"`
}

type FavoriteQuotaInfo struct {
	Limit int `json:"limit"`
	Used  int `json:"used"`
}

type SyncResult struct {
	ServerTime    int64             `json:"serverTime"`
	Favorites     []Favorite        `json:"favorites"`
	Deleted       []DeletedRef      `json:"deleted"`
	Skipped       []SyncSkip        `json:"skipped"`
	Summary       SyncSummary       `json:"summary"`
	FavoriteQuota FavoriteQuotaInfo `json:"favoriteQuota"`
}

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

func (c *Client) accessToken(ctx context.Context, link *models.CloudLink) (string, error) {
	if link.AccessToken == "" || link.RefreshToken == "" {
		return "", ErrRelink
	}
	if link.AccessExpiresAt > time.Now().Unix()+60 {
		return c.decryptToken(link, link.AccessToken)
	}

	return c.refreshLink(ctx, link)
}

func (c *Client) decryptToken(link *models.CloudLink, encrypted string) (string, error) {
	key, err := loadOrCreateKey()
	if err != nil {
		return "", err
	}
	token, err := decryptSecret(key, link.Username, encrypted)
	if err != nil {
		return "", ErrRelink
	}
	if token == "" {
		return "", ErrRelink
	}

	return token, nil
}

func (c *Client) refreshLink(ctx context.Context, link *models.CloudLink) (string, error) {
	if link.RefreshToken == "" {
		return "", ErrRelink
	}
	if link.RefreshExpiresAt > 0 && link.RefreshExpiresAt <= time.Now().Unix() {
		return "", ErrRelink
	}

	refreshToken, err := c.decryptToken(link, link.RefreshToken)
	if err != nil {
		return "", err
	}

	pair, err := c.Refresh(ctx, refreshToken)
	if err != nil {
		if IsUnauthorized(err) || IsForbidden(err) {
			return "", ErrRelink
		}
		return "", err
	}

	if err = applyTokenPair(link, *pair); err != nil {
		return "", err
	}

	return c.decryptToken(link, link.AccessToken)
}

func applyTokenPair(link *models.CloudLink, pair TokenPair) error {
	key, err := loadOrCreateKey()
	if err != nil {
		return err
	}
	access, err := encryptSecret(key, link.Username, pair.AccessToken)
	if err != nil {
		return err
	}
	refresh, err := encryptSecret(key, link.Username, pair.RefreshToken)
	if err != nil {
		return err
	}
	link.AccessToken = access
	link.RefreshToken = refresh
	link.AccessExpiresAt = pair.AccessTokenExpiresAt
	link.RefreshExpiresAt = pair.RefreshTokenExpiresAt

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

func (c *Client) Login(ctx context.Context, identity, password string) (*AuthResult, error) {
	var result AuthResult
	err := c.requestJSON(ctx, c.httpClient, http.MethodPost, "/auth/login", "", map[string]string{
		"identity": identity,
		"password": password,
	}, &result)

	return &result, err
}

func (c *Client) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	var result struct {
		Tokens TokenPair `json:"tokens"`
	}
	err := c.requestJSON(ctx, c.httpClient, http.MethodPost, "/auth/refresh", "", map[string]string{
		"refreshToken": refreshToken,
	}, &result)

	return &result.Tokens, err
}

func (c *Client) Me(ctx context.Context, link *models.CloudLink) (*MeResult, error) {
	var result MeResult
	err := c.authorizedJSON(ctx, c.httpClient, link, http.MethodGet, "/auth/me", nil, &result)

	return &result, err
}

func (c *Client) ListFavorites(ctx context.Context, link *models.CloudLink, page, pageSize int, keyword string) (*FavoritePage, error) {
	path := fmt.Sprintf("/favorites?page=%d&pageSize=%d&keyword=%s", page, pageSize, url.QueryEscape(keyword))
	var result FavoritePage
	err := c.authorizedJSON(ctx, c.httpClient, link, http.MethodGet, path, nil, &result)

	return &result, err
}

func (c *Client) Sync(ctx context.Context, link *models.CloudLink, in SyncRequest) (*SyncResult, error) {
	var result SyncResult
	err := c.authorizedJSON(ctx, c.syncClient, link, http.MethodPost, "/favorites/sync", in, &result)

	return &result, err
}

func (c *Client) ListTokens(ctx context.Context, link *models.CloudLink) (*TokenList, error) {
	var result TokenList
	err := c.authorizedJSON(ctx, c.httpClient, link, http.MethodGet, "/tokens", nil, &result)

	return &result, err
}

func (c *Client) ListGameEntries(ctx context.Context, link *models.CloudLink, kind string) (*GameEntryList, error) {
	path := "/game-admins"
	if kind == "slot" {
		path = "/game-reserved-slots"
	}
	var result GameEntryList
	err := c.authorizedJSON(ctx, c.httpClient, link, http.MethodGet, path, nil, &result)

	return &result, err
}

func (c *Client) ListArchives(ctx context.Context, link *models.CloudLink) (*ArchiveList, error) {
	var result ArchiveList
	err := c.authorizedJSON(ctx, c.httpClient, link, http.MethodGet, "/archives", nil, &result)

	return &result, err
}

func (c *Client) UploadArchive(ctx context.Context, link *models.CloudLink, filePath, name, remark, metadata string) (*Archive, error) {
	build := func(accessToken string) (*http.Request, error) {
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}

		pipeReader, pipeWriter := io.Pipe()
		writer := multipart.NewWriter(pipeWriter)
		go func() {
			defer file.Close()
			part, err := writer.CreateFormFile("file", filepath.Base(filePath))
			if err != nil {
				_ = pipeWriter.CloseWithError(err)
				return
			}
			if _, err = io.Copy(part, file); err != nil {
				_ = pipeWriter.CloseWithError(err)
				return
			}
			if err = writer.WriteField("name", name); err != nil {
				_ = pipeWriter.CloseWithError(err)
				return
			}
			if err = writer.WriteField("remark", remark); err != nil {
				_ = pipeWriter.CloseWithError(err)
				return
			}
			if err = writer.WriteField("metadata", metadata); err != nil {
				_ = pipeWriter.CloseWithError(err)
				return
			}
			if err = writer.Close(); err != nil {
				_ = pipeWriter.CloseWithError(err)
				return
			}
			_ = pipeWriter.Close()
		}()

		req, err := c.newRequest(ctx, http.MethodPost, "/archives", accessToken, pipeReader)
		if err != nil {
			_ = pipeReader.Close()
			return nil, err
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())

		return req, nil
	}

	resp, err := c.doAuthorized(ctx, c.uploadClient, link, build)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Archive
	if err = decodeResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) DownloadArchive(ctx context.Context, link *models.CloudLink, id string) (io.ReadCloser, string, int64, error) {
	resp, err := c.doAuthorized(ctx, c.streamClient, link, func(accessToken string) (*http.Request, error) {
		return c.newRequest(ctx, http.MethodGet, "/archives/"+url.PathEscape(id)+"/download", accessToken, nil)
	})
	if err != nil {
		return nil, "", 0, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		if err = decodeResponse(resp, nil); err != nil {
			return nil, "", 0, err
		}
		return nil, "", 0, &APIError{StatusCode: resp.StatusCode, Message: "下载云存档失败"}
	}

	return resp.Body, downloadFileName(resp.Header.Get("Content-Disposition"), id), resp.ContentLength, nil
}

func downloadFileName(header, fallback string) string {
	if header == "" {
		return "archive-" + fallback + ".zip"
	}
	if index := strings.Index(header, "filename*=UTF-8''"); index >= 0 {
		encoded := header[index+len("filename*=UTF-8''"):]
		if end := strings.Index(encoded, ";"); end >= 0 {
			encoded = encoded[:end]
		}
		if decoded, err := url.PathUnescape(strings.TrimSpace(encoded)); err == nil && decoded != "" {
			return decoded
		}
	}
	_, params, err := mime.ParseMediaType(header)
	if err == nil {
		if name := strings.TrimSpace(params["filename"]); name != "" {
			return name
		}
	}

	return "archive-" + fallback + ".zip"
}
