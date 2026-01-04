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
	"time"
)

type Handler struct {
	roomDao        *dao.RoomDAO
	worldDao       *dao.WorldDAO
	roomSettingDao *dao.RoomSettingDAO
}

func NewHandler(roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO) *Handler {
	return &Handler{
		roomDao:        roomDao,
		worldDao:       worldDao,
		roomSettingDao: roomSettingDao,
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
	ID              string   `json:"publishedfileid"`
	FileSize        string   `json:"file_size"`
	FileDescription string   `json:"file_description"`
	FileUrl         string   `json:"file_url"`
	Title           string   `json:"title"`
	Tags            []Tags   `json:"tags"`
	PreviewUrl      string   `json:"preview_url"`
	VoteData        VoteData `json:"vote_data"`
	TimeCreated     int      `json:"time_created"`
	TimeUpdated     int      `json:"time_updated"`
	Subscriptions   int      `json:"subscriptions"`
}
type Data struct {
	Total    int       `json:"total"`
	Page     int       `json:"page"`
	PageSize int       `json:"pageSize"`
	Rows     []ModInfo `json:"rows"`
}
type ModInfo struct {
	Name            string   `json:"name"`
	ID              int      `json:"id"`
	Size            string   `json:"size"`
	Tags            []Tags   `json:"tags"`
	PreviewUrl      string   `json:"preview_url"`
	FileDescription string   `json:"file_description"`
	FileUrl         string   `json:"file_url"`
	VoteData        VoteData `json:"vote_data"`
	DownloadedReady bool     `json:"downloadedReady"`
	TimeCreated     int      `json:"time_created"`
	TimeUpdated     int      `json:"time_updated"`
	Subscriptions   int      `json:"subscriptions"`
}
type Tags struct {
	Tag         string `json:"tag"`
	DisplayName string `json:"display_name"`
}
type VoteData struct {
	Score     float64 `json:"score"`
	VotesUp   int     `json:"votes_up"`
	VotesDown int     `json:"votes_down"`
}

func SearchMod(page int, pageSize int, searchText string, lang string) (Data, error) {
	var (
		language int
		url      string
	)
	if lang == "zh" {
		language = 6
	} else {
		language = 0
	}
	url = fmt.Sprintf("%s?appid=322330&return_vote_data=true&return_children=true&", utils.SteamApiModSearch)
	url = url + "requiredtags[0]=server_only_mod&requiredtags[1]=all_clients_require_mod&match_all_tags=false&"
	if searchText == "" {
		url = url + fmt.Sprintf("language=%d&key=%s&page=%d&numperpage=%d",
			language,
			utils.GetSteamApiKey(),
			page,
			pageSize,
		)
	} else {
		url = url + fmt.Sprintf("language=%d&key=%s&page=%d&numperpage=%d&search_text=%s",
			language,
			utils.GetSteamApiKey(),
			page,
			pageSize,
			searchText,
		)
	}

	client := &http.Client{
		Timeout: utils.HttpTimeout * time.Second,
	}
	httpResponse, err := client.Get(url)
	if err != nil {
		return Data{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Logger.Error("请求关闭失败", "err", err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体
	// 检查 HTTP 状态码
	if httpResponse.StatusCode != http.StatusOK {
		return Data{}, err
	}
	var jsonResp JSONResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&jsonResp); err != nil {
		logger.Logger.Error("解析JSON失败", "err", err)
		return Data{}, err
	}

	var modInfoList []ModInfo
	for _, i := range jsonResp.Response.Publishedfiledetails {
		modInfo := ModInfo{
			ID:              func() int { id, _ := strconv.Atoi(i.ID); return id }(),
			Name:            i.Title,
			Size:            i.FileSize,
			Tags:            i.Tags,
			PreviewUrl:      i.PreviewUrl,
			FileDescription: i.FileDescription,
			FileUrl:         i.FileUrl,
			VoteData:        i.VoteData,
			TimeCreated:     i.TimeCreated,
			TimeUpdated:     i.TimeUpdated,
			Subscriptions:   i.Subscriptions,
		}
		modInfoList = append(modInfoList, modInfo)
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
	var (
		language int
		url      string
	)
	if lang == "zh" {
		language = 6
	} else {
		language = 0
	}

	url = fmt.Sprintf("%s?language=%d&key=%s", utils.SteamApiModDetail, language, utils.GetSteamApiKey())
	url = url + fmt.Sprintf("&publishedfileids[0]=%d", id)

	client := &http.Client{
		Timeout: utils.HttpTimeout * time.Second,
	}
	httpResponse, err := client.Get(url)
	if err != nil {
		return Data{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Logger.Error("请求关闭失败", "err", err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体
	// 检查 HTTP 状态码
	if httpResponse.StatusCode != http.StatusOK {
		return Data{}, err
	}
	var jsonResp JSONResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&jsonResp); err != nil {
		logger.Logger.Error("解析JSON失败", "err", err)
		return Data{}, err
	}

	var modInfoList []ModInfo
	for _, i := range jsonResp.Response.Publishedfiledetails {
		modInfo := ModInfo{
			ID:              func() int { id, _ := strconv.Atoi(i.ID); return id }(),
			Name:            i.Title,
			Size:            i.FileSize,
			Tags:            i.Tags,
			PreviewUrl:      i.PreviewUrl,
			FileDescription: i.FileDescription,
			FileUrl:         i.FileUrl,
			VoteData:        i.VoteData,
		}
		modInfoList = append(modInfoList, modInfo)
	}

	data := Data{
		Total:    1,
		Page:     1,
		PageSize: 1,
		Rows:     modInfoList,
	}

	return data, nil
}

func addDownloadedModInfo(mods *[]dst.DownloadedMod, lang string) error {
	if len(*mods) == 0 {
		return nil
	}

	var language int
	if lang == "zh" {
		language = 6
	} else {
		language = 0
	}

	url := fmt.Sprintf("%s?language=%d&key=%s", utils.SteamApiModDetail, language, utils.GetSteamApiKey())
	for index, mod := range *mods {
		logger.Logger.Debug(fmt.Sprintf("mod id %d", mod.ID))
		url = url + fmt.Sprintf("&publishedfileids[%d]=%d", index, mod.ID)
	}

	client := &http.Client{
		Timeout: utils.HttpTimeout * time.Second,
	}
	httpResponse, err := client.Get(url)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Logger.Error("请求关闭失败", "err", err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体
	// 检查 HTTP 状态码
	if httpResponse.StatusCode != http.StatusOK {
		return err
	}
	var jsonResp JSONResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&jsonResp); err != nil {
		logger.Logger.Error("解析JSON失败", "err", err)
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
	}

	return nil
}

func (h *Handler) fetchGameInfo(roomID int) (*models.Room, *[]models.World, *models.RoomSetting, error) {
	room, err := h.roomDao.GetRoomByID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}
	worlds, err := h.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}
	roomSetting, err := h.roomSettingDao.GetRoomSettingsByRoomID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}

	return room, worlds, roomSetting, nil
}
