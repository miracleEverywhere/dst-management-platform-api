package mod

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

type Handler struct {
	roomDao        *dao.RoomDAO
	worldDao       *dao.WorldDAO
	roomSettingDao *dao.RoomSettingDAO
	userDao        *dao.UserDAO
	modInfoDao     *dao.ModInfoDAO
}

func NewHandler(roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO, userDao *dao.UserDAO, modInfoDaos ...*dao.ModInfoDAO) *Handler {
	var modInfoDao *dao.ModInfoDAO
	if len(modInfoDaos) > 0 {
		modInfoDao = modInfoDaos[0]
	}

	return &Handler{
		roomDao:        roomDao,
		worldDao:       worldDao,
		roomSettingDao: roomSettingDao,
		userDao:        userDao,
		modInfoDao:     modInfoDao,
	}
}

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

// 模组信息缓存时长 10分钟
const cacheModInfoMaxAgeMs = 10 * 60 * 1000

var client *http.Client = &http.Client{
	Timeout: utils.HttpTimeout * time.Second,
}

func SearchMod(page int, pageSize int, searchText string, lang string) (Data, error) {
	var language int

	lang = normalizeModLanguage(lang)
	if lang == "zh" {
		language = 6
	} else {
		language = 0
	}

	req, err := http.NewRequest("GET", utils.SteamApiModSearch, nil)
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

func SearchModById(id int, lang string) (Data, error) {
	var language int
	lang = normalizeModLanguage(lang)
	if lang == "zh" {
		language = 6
	} else {
		language = 0
	}

	req, err := http.NewRequest("GET", utils.SteamApiModDetail, nil)
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

func (h *Handler) addDownloadedModInfo(mods *[]dst.DownloadedMod, lang string) error {
	if len(*mods) == 0 {
		return nil
	}

	var ids []int
	for _, mod := range *mods {
		ids = append(ids, mod.ID)
	}

	lang = normalizeModLanguage(lang)
	cacheValid := h.modInfoDao != nil
	modInfos := []models.ModInfo(nil)
	if cacheValid {
		var err error
		modInfos, err = h.modInfoDao.GetModInfoByIDs(ids, lang)
		cacheValid = err == nil
	}

	if cacheValid {
		cachedByID := make(map[int]models.ModInfo, len(modInfos))
		for _, modInfo := range modInfos {
			cachedByID[modInfo.ID] = modInfo
		}
		for idx := range *mods {
			modInfo, ok := cachedByID[(*mods)[idx].ID]
			if !ok || !checkCacheValidity(modInfo.CacheTimestamp) {
				cacheValid = false
				continue
			}
			(*mods)[idx].Name = modInfo.Name
			(*mods)[idx].FileURL = modInfo.FileUrl
			(*mods)[idx].PreviewURL = modInfo.PreviewUrl
			(*mods)[idx].ServerSize = modInfo.Size
		}
	}

	if cacheValid {
		logger.Logger.Debugf("返回模组缓存数据")
		return nil
	}

	var language int
	if lang == "zh" {
		language = 6
	} else {
		language = 0
	}

	req, err := http.NewRequest("GET", utils.SteamApiModDetail, nil)
	if err != nil {
		return err
	}

	q := req.URL.Query()
	q.Add("language", strconv.Itoa(language))
	q.Add("key", utils.GetSteamApiKey())
	for index, mod := range *mods {
		q.Add(fmt.Sprintf("publishedfileids[%d]", index), strconv.Itoa(mod.ID))
	}
	req.URL.RawQuery = q.Encode()

	httpResponse, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Logger.Errorf("请求关闭失败, err: %v", err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体
	// 检查 HTTP 状态码
	if httpResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("steam api returned status %d", httpResponse.StatusCode)
	}
	var jsonResp JSONResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&jsonResp); err != nil {
		logger.Logger.Errorf("解析JSON失败, err: %v", err)
		return err
	}

	for _, i := range jsonResp.Response.Publishedfiledetails {
		id := func() int { id, _ := strconv.Atoi(i.ID); return id }()
		for idx := range *mods {
			if (*mods)[idx].ID == id {
				(*mods)[idx].Name = i.Title
				(*mods)[idx].FileURL = i.FileUrl
				(*mods)[idx].PreviewURL = i.PreviewUrl
				(*mods)[idx].ServerSize = i.FileSize
			}
		}
		if h.modInfoDao != nil {
			modInfo := modInfoFromPublishedFile(i, lang)
			modInfo.CacheTimestamp = utils.GetTimestamp()
			if err = h.modInfoDao.UpdateModInfo(&modInfo); err != nil {
				logger.Logger.Errorf("更新模组缓存信息失败, err: %v", err)
			}
		}

	}

	return nil
}

func (h *Handler) hasPermission(c *gin.Context, roomID string) bool {
	role, _ := c.Get("role")
	username, _ := c.Get("username")

	// 管理员直接返回true
	if role.(string) == "admin" {
		return true
	} else {
		dbUser, err := h.userDao.GetUserByUsername(username.(string))
		if err != nil {
			logger.Logger.Error("查询数据库失败")
			return false
		}
		roomIDs := strings.Split(dbUser.Rooms, ",")
		for _, id := range roomIDs {
			if id == roomID {
				return true
			}
		}
	}

	return false
}

func checkNotUgcUrl(url string) bool {
	if url == "" {
		return true
	}

	urlParts := strings.Split(url, "/")
	if len(urlParts) != 7 {
		return false
	}

	if urlParts[2] != "cdn.steamusercontent.com" {
		return false
	}

	return true
}

func checkCacheValidity(ts int64) bool {
	if ts <= 0 {
		return false
	}

	tsDiff := time.Now().UnixMilli() - ts
	return tsDiff >= 0 && tsDiff < cacheModInfoMaxAgeMs
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

func normalizeModLanguage(lang string) string {
	if lang == "en" {
		return "en"
	}
	return "zh"
}
