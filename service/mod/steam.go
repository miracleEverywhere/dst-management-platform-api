package mod

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"gorm.io/datatypes"
)

type JSONResponse struct {
	Response Response `json:"response"`
}

type Response struct {
	Total                int                    `json:"total"`
	Publishedfiledetails []PublishedFileDetails `json:"publishedfiledetails"`
}

type PublishedFileDetails struct {
	ID              string          `json:"publishedfileid"`
	FileSize        string          `json:"file_size"`
	FileDescription string          `json:"file_description"`
	FileUrl         string          `json:"file_url"`
	Title           string          `json:"title"`
	Tags            []models.Tags   `json:"tags"`
	PreviewUrl      string          `json:"preview_url"`
	VoteData        models.VoteData `json:"vote_data"`
	TimeCreated     int             `json:"time_created"`
	TimeUpdated     int             `json:"time_updated"`
	Subscriptions   int             `json:"subscriptions"`
}

type Data struct {
	Total    int              `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"pageSize"`
	Rows     []models.ModInfo `json:"rows"`
}

func searchMod(page int, pageSize int, searchText string, lang string) (Data, error) {
	var language int

	lang = normalizeModLanguage(lang)
	if lang == "zh" {
		language = 6
	} else {
		language = 0
	}

	req, err := http.NewRequest("GET", constant.SteamApiModSearch, nil)
	if err != nil {
		return Data{}, err
	}

	q := req.URL.Query()
	q.Add("appid", "322330")
	q.Add("return_vote_data", "true")
	q.Add("return_children", "true")
	q.Add("requiredtags[0]", "server_only_mod")
	q.Add("requiredtags[1]", "all_clients_require_mod")
	q.Add("match_all_tags", "false")
	q.Add("language", strconv.Itoa(language))
	q.Add("key", utils.GetSteamApiKey())
	q.Add("page", strconv.Itoa(page))
	q.Add("numperpage", strconv.Itoa(pageSize))
	if searchText != "" {
		q.Add("search_text", searchText)
	}
	req.URL.RawQuery = q.Encode()

	httpResponse, err := client.Do(req)
	if err != nil {
		return Data{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Logger.Errorf("请求关闭失败, err: %v", err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体
	// 检查 HTTP 状态码
	if httpResponse.StatusCode != http.StatusOK {
		return Data{}, fmt.Errorf("steam api returned status %d", httpResponse.StatusCode)
	}
	var jsonResp JSONResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&jsonResp); err != nil {
		logger.Logger.Errorf("解析JSON失败, err: %v", err)
		return Data{}, err
	}

	var modInfoList []models.ModInfo
	for _, i := range jsonResp.Response.Publishedfiledetails {
		modInfoList = append(modInfoList, modInfoFromPublishedFile(i, lang))
	}

	data := Data{
		Total:    jsonResp.Response.Total,
		Page:     page,
		PageSize: pageSize,
		Rows:     modInfoList,
	}

	return data, nil
}

func searchModById(id int, lang string) (Data, error) {
	var language int
	lang = normalizeModLanguage(lang)
	if lang == "zh" {
		language = 6
	} else {
		language = 0
	}

	req, err := http.NewRequest("GET", constant.SteamApiModDetail, nil)
	if err != nil {
		return Data{}, err
	}

	q := req.URL.Query()
	q.Add("language", strconv.Itoa(language))
	q.Add("key", utils.GetSteamApiKey())
	q.Add("publishedfileids[0]", strconv.Itoa(id))
	req.URL.RawQuery = q.Encode()

	httpResponse, err := client.Do(req)
	if err != nil {
		return Data{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Logger.Errorf("请求关闭失败, err: %v", err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体
	// 检查 HTTP 状态码
	if httpResponse.StatusCode != http.StatusOK {
		return Data{}, fmt.Errorf("steam api returned status %d", httpResponse.StatusCode)
	}
	var jsonResp JSONResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&jsonResp); err != nil {
		logger.Logger.Errorf("解析JSON失败, err: %v", err)
		return Data{}, err
	}

	var modInfoList []models.ModInfo
	for _, i := range jsonResp.Response.Publishedfiledetails {
		modInfoList = append(modInfoList, modInfoFromPublishedFile(i, lang))
	}

	data := Data{
		Total:    1,
		Page:     1,
		PageSize: 1,
		Rows:     modInfoList,
	}

	return data, nil
}

func modInfoFromPublishedFile(i PublishedFileDetails, lang string) models.ModInfo {
	return models.ModInfo{
		ID:              func() int { id, _ := strconv.Atoi(i.ID); return id }(),
		Language:        normalizeModLanguage(lang),
		Name:            i.Title,
		Size:            i.FileSize,
		Tags:            datatypes.NewJSONType(i.Tags),
		PreviewUrl:      i.PreviewUrl,
		FileDescription: i.FileDescription,
		FileUrl:         i.FileUrl,
		VoteData:        i.VoteData,
		TimeCreated:     i.TimeCreated,
		TimeUpdated:     i.TimeUpdated,
		Subscriptions:   i.Subscriptions,
	}
}
