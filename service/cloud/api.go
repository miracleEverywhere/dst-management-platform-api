package cloud

import (
	"context"
	"dst-management-platform-api/database/models"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

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
