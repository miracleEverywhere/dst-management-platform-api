package mod

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/datatypes"
)

// Service 模组模块业务实现，不依赖 gin
type Service struct {
	roomDao    *dao.RoomDAO
	worldDao   *dao.WorldDAO
	userDao    *dao.UserDAO
	modInfoDao *dao.ModInfoDAO
}

// New 创建模组模块 service
func New(roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, userDao *dao.UserDAO, modInfoDao *dao.ModInfoDAO) *Service {
	return &Service{
		roomDao:    roomDao,
		worldDao:   worldDao,
		userDao:    userDao,
		modInfoDao: modInfoDao,
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
	Timeout: constant.HttpTimeout * time.Second,
}

// ModSearch 按 id 或文本搜索模组
func (s *Service) ModSearch(actor service.Actor, searchType, searchText string, page, pageSize int) service.Result {
	langStr := normalizeModLanguage(actor.Lang)

	if searchType == "id" {
		id, err := strconv.Atoi(searchText)
		if err != nil || id <= 0 {
			logger.Logger.Infof("请求参数错误: %v, api: %s", err, actor.Path)
			return service.Fail(constant.CodeModBadRequest)
		}

		var cachedModInfo *models.ModInfo
		if s.modInfoDao != nil {
			cachedModInfo, err = s.modInfoDao.GetModInfoByID(id, langStr)
		}
		if err == nil && cachedModInfo != nil && checkCacheValidity(cachedModInfo.CacheTimestamp) {
			data := Data{
				Total:    1,
				Page:     1,
				PageSize: 1,
				Rows: []models.ModInfo{
					*cachedModInfo,
				},
			}
			return service.Success(data)
		}

		data, err := searchModById(id, langStr)
		if err != nil {
			logger.Logger.Errorf("获取mod信息失败, err: %v", err)
			return service.Fail(constant.CodeModSearchFail)
		}
		if len(data.Rows) > 0 && s.modInfoDao != nil {
			data.Rows[0].CacheTimestamp = utils.GetTimestamp()
			if cacheErr := s.modInfoDao.UpdateModInfo(&data.Rows[0]); cacheErr != nil {
				logger.Logger.Errorf("更新模组缓存信息失败, err: %v", cacheErr)
			}
		}

		return service.Success(data)
	}

	data, err := searchMod(page, pageSize, searchText, langStr)
	if err != nil {
		logger.Logger.Errorf("获取mod信息失败, err: %v", err)
		return service.Fail(constant.CodeModSearchFail)
	}

	return service.Success(data)
}

// ModDownload 下载模组，sync 为 true 时同步下载，否则后台下载
func (s *Service) ModDownload(actor service.Actor, roomID, id int, fileURL, size, name string, sync bool) service.Result {
	// url https://cdn.steamusercontent.com/ugc/1466437966115152320/8A3E11F0B32FCBFBF308DEB0B5C98A702215374B/
	if ok := checkNotUgcUrl(fileURL); !ok {
		logger.Logger.Warnf("异常模组下载请求已拦截: %s, api: %s", fileURL, actor.Path)
		logger.Logger.Warn("疑似攻击请求，请立即修改用户名密码及重置jwt-secret")
		return service.Fail(constant.CodeModBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	reqSize, err := strconv.Atoi(size)
	if err != nil || reqSize <= 0 || roomID <= 0 || id <= 0 {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, actor.Path)
		return service.Fail(constant.CodeModBadRequest)
	}

	reqSize64 := int64(reqSize)

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	if cache.ModDownloadStatus == nil {
		logger.Logger.Error("模组下载状态缓存未初始化")
		return service.Fail(constant.CodeModServerError)
	}

	if item, err := cache.ModDownloadStatus.Get(roomID, id); err == nil && item.CurrentSize < item.Size {
		return service.Result{
			Code: 200,
			MsgBuilder: func(tr func(msgKey string, args ...any) string) string {
				return name + " " + tr("downloading")
			},
		}
	}

	if err := cache.ModDownloadStatus.Set(roomID, &cache.ModItem{
		ID:          id,
		Size:        reqSize,
		CurrentSize: 0,
	}); err != nil {
		logger.Logger.Errorf("写入模组下载状态失败, err: %v", err)
		return service.Fail(constant.CodeModServerError)
	}

	dl := func() {
		err, modSize := game.DownloadMod(id, fileURL)
		if err != nil || modSize != reqSize64 {
			logger.Logger.Debugf("模组大小与预期不符, %d, %d, err: %v", modSize, reqSize64, err)
			if deleteErr := cache.ModDownloadStatus.Delete(roomID, id); deleteErr != nil {
				logger.Logger.Debugf("删除模组下载状态失败, err: %v", deleteErr)
			}
			return
		}

		if setErr := cache.ModDownloadStatus.Set(roomID, &cache.ModItem{
			ID:          id,
			Size:        reqSize,
			CurrentSize: reqSize,
		}); setErr != nil {
			logger.Logger.Errorf("更新模组下载状态失败, err: %v", setErr)
		}
	}

	if sync {
		dl()
		return service.MsgF("downloaded", []any{name}, nil)
	}

	go dl()
	return service.MsgF("downloading", []any{name}, nil)
}

// ModDownloadStatus 查询模组下载进度
func (s *Service) ModDownloadStatus(actor service.Actor, roomID, id int) service.Result {
	progress := 0
	if cache.ModDownloadStatus != nil {
		if item, err := cache.ModDownloadStatus.Get(roomID, id); err == nil {
			if item.Size > 0 && item.CurrentSize >= item.Size {
				progress = 100
			} else if item.Size > 0 && item.CurrentSize > 0 {
				progress = int(float64(item.CurrentSize) / float64(item.Size) * 100)
			} else if item.CurrentSize > 0 {
				progress = 100
			}
			if progress < 0 {
				progress = 0
			}
			if progress > 100 {
				progress = 100
			}
		}
	}

	return service.Success(progress)
}

// DownloadedMods 已下载模组列表
func (s *Service) DownloadedMods(actor service.Actor, roomID int) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	downloadedMods := game.GetDownloadedMods()

	if err = s.addDownloadedModInfo(downloadedMods, actor.Lang); err != nil {
		logger.Logger.Error("添加模组额外信息失败")
	}

	return service.Success(downloadedMods)
}

// DownloadedModIDs 已下载模组 ID 列表
func (s *Service) DownloadedModIDs(actor service.Actor, roomID int) service.Result {
	if roomID == 0 {
		return service.Result{Code: constant.CodeModBadRequest, Data: []int{}}
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Result{Code: constant.CodeModNoPermission, Data: []int{}}
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Result{Code: constant.CodeModDBError, Data: []int{}}
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	downloadedMods := game.GetDownloadedMods()
	modIDs := make([]int, 0, len(*downloadedMods))
	seen := make(map[int]struct{}, len(*downloadedMods))
	for _, downloadedMod := range *downloadedMods {
		if downloadedMod.ID <= 0 {
			continue
		}
		if _, exists := seen[downloadedMod.ID]; exists {
			continue
		}

		seen[downloadedMod.ID] = struct{}{}
		modIDs = append(modIDs, downloadedMod.ID)
	}
	sort.Ints(modIDs)

	return service.Success(modIDs)
}

// SettingModConfigStruct 模组配置结构
func (s *Service) SettingModConfigStruct(actor service.Actor, roomID, worldID, id int, fileURL string) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	options, err := game.GetModConfigureOptions(worldID, id, fileURL == "")
	if err != nil {
		logger.Logger.Error("获取模组设置失败")
		return service.Result{Code: constant.CodeModConfigOptionsError, Data: options}
	}

	return service.Success(options)
}

// SettingModConfigValue 模组配置值
func (s *Service) SettingModConfigValue(actor service.Actor, roomID, worldID, id int, fileURL string) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	options, err := game.GetModConfigureOptionsValues(worldID, id, fileURL == "")
	if err != nil {
		logger.Logger.Error("获取模组设置失败")
		return service.Result{Code: constant.CodeModConfigValuesError, Data: options}
	}

	return service.Success(options)
}

// SettingModConfigValuePut 修改模组配置值
func (s *Service) SettingModConfigValuePut(actor service.Actor, roomID, worldID, id int, modORConfig dst.ModORConfig) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	err = game.ModConfigureOptionsValuesChange(worldID, id, &modORConfig)
	if err != nil {
		logger.Logger.Error("修改模组设置失败")
		return service.Fail(constant.CodeModModifyConfigError)
	}

	if err = s.roomDao.UpdateRoom(room); err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	if err = s.worldDao.UpdateWorlds(worlds); err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	return service.Msg("modify mod configuration values success", nil)
}

// AddEnable 启用模组
func (s *Service) AddEnable(actor service.Actor, roomID, worldID, id int, fileURL string) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	err = game.ModEnable(worldID, id, fileURL == "")
	if err != nil {
		logger.Logger.Errorf("模组启用失败, err: %v", err)
		return service.Fail(constant.CodeModEnableFail)
	}

	if err = s.roomDao.UpdateRoom(room); err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	if err = s.worldDao.UpdateWorlds(worlds); err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	return service.Msg("mod enable success", nil)
}

// AddDisable 禁用模组
func (s *Service) AddDisable(actor service.Actor, roomID, worldID, id int) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	err = game.ModDisable(id)
	if err != nil {
		logger.Logger.Errorf("模组禁用失败, err: %v", err)
		return service.Fail(constant.CodeModDisableFail)
	}

	if err = s.roomDao.UpdateRoom(room); err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	if err = s.worldDao.UpdateWorlds(worlds); err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	return service.Msg("mod disable success", nil)
}

// EnabledMods 已启用模组列表
func (s *Service) EnabledMods(actor service.Actor, roomID, worldID int) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	modsID, err := game.GetEnabledMods(worldID)
	if err != nil {
		logger.Logger.Error("获取模组设置失败")
		return service.Result{Code: constant.CodeModGetEnabledFail, Data: modsID}
	}

	if err = s.addDownloadedModInfo(&modsID, actor.Lang); err != nil {
		logger.Logger.Error("添加模组额外信息失败")
	}

	return service.Success(modsID)
}

// ModInfo 批量查询模组信息
func (s *Service) ModInfo(actor service.Actor, inputIDs []int) service.Result {
	modIDs := make([]int, 0, len(inputIDs))
	seen := make(map[int]struct{}, len(inputIDs))
	for _, modID := range inputIDs {
		if modID <= 0 {
			continue
		}
		if _, ok := seen[modID]; ok {
			continue
		}

		seen[modID] = struct{}{}
		modIDs = append(modIDs, modID)
	}

	mods := make([]dst.DownloadedMod, 0, len(modIDs))
	for _, modID := range modIDs {
		mods = append(mods, dst.DownloadedMod{ID: modID})
	}

	if len(mods) > 0 {
		err := s.addDownloadedModInfo(&mods, actor.Lang)
		if err != nil {
			logger.Logger.Errorf("获取模组信息失败, err: %v", err)
			return service.Fail(constant.CodeModSearchFail)
		}
	}

	return service.Success(mods)
}

// Delete 删除单个模组
func (s *Service) Delete(actor service.Actor, roomID, id int, fileURL string) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	err = game.ModDelete(id, fileURL)
	if err != nil {
		logger.Logger.Errorf("删除模组失败, err: %v", err)
		return service.Fail(constant.CodeModDeleteFail)
	}

	err = cache.ModDownloadStatus.Delete(roomID, id)
	if err != nil {
		logger.Logger.Errorf("删除模组下载状态缓存失败: %v", err)
	}

	return service.Msg("delete success", nil)
}

// DeleteMulti 批量删除模组
func (s *Service) DeleteMulti(actor service.Actor, roomID int, mods []dst.DeleteModInfo) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	err = game.ModDeleteMulti(mods)
	if err != nil {
		logger.Logger.Errorf("删除模组失败, err: %v", err)
		return service.Fail(constant.CodeModDeleteFail)
	}

	for _, mod := range mods {
		err = cache.ModDownloadStatus.Delete(roomID, mod.ID)
		if err != nil {
			logger.Logger.Errorf("删除模组下载状态缓存失败: %v", err)
		}
	}

	return service.Msg("delete success", nil)
}

// DeleteAcf 删除 acf 文件
func (s *Service) DeleteAcf(actor service.Actor, roomID int) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	err = game.DeleteAcf()
	if err != nil {
		logger.Logger.Errorf("删除acf文件失败, err: %v", err)
		return service.Fail(constant.CodeModDeleteFail)
	}

	err = cache.ModDownloadStatus.DeleteByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("删除模组下载状态缓存失败: %v", err)
	}

	return service.Msg("delete success", nil)
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

func (s *Service) addDownloadedModInfo(mods *[]dst.DownloadedMod, lang string) error {
	if len(*mods) == 0 {
		return nil
	}

	var ids []int
	for _, mod := range *mods {
		ids = append(ids, mod.ID)
	}

	lang = normalizeModLanguage(lang)
	cacheValid := s.modInfoDao != nil
	modInfos := []models.ModInfo(nil)
	if cacheValid {
		var err error
		modInfos, err = s.modInfoDao.GetModInfoByIDs(ids, lang)
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

	req, err := http.NewRequest("GET", constant.SteamApiModDetail, nil)
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
		if s.modInfoDao != nil {
			modInfo := modInfoFromPublishedFile(i, lang)
			modInfo.CacheTimestamp = utils.GetTimestamp()
			if err = s.modInfoDao.UpdateModInfo(&modInfo); err != nil {
				logger.Logger.Errorf("更新模组缓存信息失败, err: %v", err)
			}
		}

	}

	return nil
}

func (s *Service) hasPermission(actor service.Actor, roomID string) bool {
	// 管理员直接返回true
	if actor.IsAdmin() {
		return true
	}

	dbUser, err := s.userDao.GetUserByUsername(actor.Username)
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
