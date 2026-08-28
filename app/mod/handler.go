package mod

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (h *Handler) modSearchGet(c *gin.Context) {
	langStr := normalizeModLanguage(c.Request.Header.Get("X-I18n-Lang"))

	type SearchForm struct {
		SearchType string `form:"searchType" json:"searchType"`
		SearchText string `form:"searchText" json:"searchText"`
		Page       int    `form:"page" json:"page"`
		PageSize   int    `form:"pageSize" json:"pageSize"`
	}
	var searchForm SearchForm
	if err := c.ShouldBindQuery(&searchForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	logger.Logger.Debug(utils.StructToFlatString(searchForm))

	if searchForm.SearchType == "id" {
		id, err := strconv.Atoi(searchForm.SearchText)
		if err != nil || id <= 0 {
			logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
			return
		}

		var cachedModInfo *models.ModInfo
		if h.modInfoDao != nil {
			cachedModInfo, err = h.modInfoDao.GetModInfoByID(id, langStr)
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
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
			return
		}

		data, err := SearchModById(id, langStr)
		if err != nil {
			logger.Logger.Errorf("获取mod信息失败, err: %v", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "search fail"), "data": nil})
			return
		}
		if len(data.Rows) > 0 && h.modInfoDao != nil {
			data.Rows[0].CacheTimestamp = utils.GetTimestamp()
			if cacheErr := h.modInfoDao.UpdateModInfo(&data.Rows[0]); cacheErr != nil {
				logger.Logger.Errorf("更新模组缓存信息失败, err: %v", cacheErr)
			}
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
		return
	}
	if searchForm.SearchType == "text" {
		data, err := SearchMod(searchForm.Page, searchForm.PageSize, searchForm.SearchText, langStr)
		if err != nil {
			logger.Logger.Errorf("获取mod信息失败, err: %v", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "search fail"), "data": nil})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
}

func (h *Handler) downloadPost(c *gin.Context) {
	type ReqForm struct {
		RoomID  int    `json:"roomID"`
		ID      int    `json:"id"`
		FileURL string `json:"file_url"`
		Update  bool   `json:"update"`
		Size    string `json:"size"`
		Name    string `json:"name"`
		Sync    bool   `json:"sync"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	// url https://cdn.steamusercontent.com/ugc/1466437966115152320/8A3E11F0B32FCBFBF308DEB0B5C98A702215374B/
	if ok := checkNotUgcUrl(reqForm.FileURL); !ok {
		logger.Logger.Warnf("异常模组下载请求已拦截: %s, api: %s", reqForm.FileURL, c.Request.URL.Path)
		logger.Logger.Warn("疑似攻击请求，请立即修改用户名密码及重置jwt-secret")
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	reqSize, err := strconv.Atoi(reqForm.Size)
	if err != nil || reqSize <= 0 || reqForm.RoomID <= 0 || reqForm.ID <= 0 {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	reqSize64 := int64(reqSize)

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	if cache.ModDownloadStatus == nil {
		logger.Logger.Error("模组下载状态缓存未初始化")
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "server error"), "data": nil})
		return
	}

	if item, err := cache.ModDownloadStatus.Get(reqForm.RoomID, reqForm.ID); err == nil && item.CurrentSize < item.Size {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": reqForm.Name + " " + message.Get(c, "downloading"), "data": nil})
		return
	}

	if err := cache.ModDownloadStatus.Set(reqForm.RoomID, &cache.ModItem{
		ID:          reqForm.ID,
		Size:        reqSize,
		CurrentSize: 0,
	}); err != nil {
		logger.Logger.Errorf("写入模组下载状态失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "server error"), "data": nil})
		return
	}

	dl := func() {
		err, modSize := game.DownloadMod(reqForm.ID, reqForm.FileURL)
		if err != nil || modSize != reqSize64 {
			logger.Logger.Debugf("模组大小与预期不符, %d, %d, err: %v", modSize, reqSize64, err)
			if deleteErr := cache.ModDownloadStatus.Delete(reqForm.RoomID, reqForm.ID); deleteErr != nil {
				logger.Logger.Debugf("删除模组下载状态失败, err: %v", deleteErr)
			}
			return
		}

		if setErr := cache.ModDownloadStatus.Set(reqForm.RoomID, &cache.ModItem{
			ID:          reqForm.ID,
			Size:        reqSize,
			CurrentSize: reqSize,
		}); setErr != nil {
			logger.Logger.Errorf("更新模组下载状态失败, err: %v", setErr)
		}
	}

	if reqForm.Sync {
		dl()
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.GetF(c, "downloaded", reqForm.Name), "data": nil})
	} else {
		go dl()
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.GetF(c, "downloading", reqForm.Name), "data": nil})
	}
}

func (h *Handler) downloadStatusGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `form:"roomID"`
		ID     int `form:"id"`
		ModID  int `form:"modID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil || reqForm.RoomID <= 0 || (reqForm.ID <= 0 && reqForm.ModID <= 0) {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": 0})
		return
	}
	if reqForm.ID <= 0 {
		reqForm.ID = reqForm.ModID
	}

	progress := 0
	if cache.ModDownloadStatus != nil {
		if item, err := cache.ModDownloadStatus.Get(reqForm.RoomID, reqForm.ID); err == nil {
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

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": progress})
}

func (h *Handler) downloadedModsGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	downloadedMods := game.GetDownloadedMods()

	err = h.addDownloadedModInfo(downloadedMods, c.Request.Header.Get("X-I18n-Lang"))
	if err != nil {
		logger.Logger.Error("添加模组额外信息失败")
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": downloadedMods})
}

func (h *Handler) downloadedModIDsGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": []int{}})
		return
	}

	if reqForm.RoomID == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": []int{}})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": []int{}})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": []int{}})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
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

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": modIDs})
}

func (h *Handler) settingModConfigStructGet(c *gin.Context) {
	type ReqForm struct {
		RoomID  int    `form:"roomID"`
		WorldID int    `form:"worldID"`
		ID      int    `form:"id"`
		FileURL string `form:"file_url"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	options, err := game.GetModConfigureOptions(reqForm.WorldID, reqForm.ID, reqForm.FileURL == "")
	if err != nil {
		logger.Logger.Error("获取模组设置失败")
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "mod configuration options error"), "data": options})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": options})
}

func (h *Handler) settingModConfigValueGet(c *gin.Context) {
	type ReqForm struct {
		RoomID  int    `form:"roomID"`
		WorldID int    `form:"worldID"`
		ID      int    `form:"id"`
		FileURL string `form:"file_url"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	options, err := game.GetModConfigureOptionsValues(reqForm.WorldID, reqForm.ID, reqForm.FileURL == "")
	if err != nil {
		logger.Logger.Error("获取模组设置失败")
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "mod configuration values error"), "data": options})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": options})
}

func (h *Handler) settingModConfigValuePut(c *gin.Context) {
	type ReqForm struct {
		RoomID      int             `json:"roomID"`
		WorldID     int             `json:"worldID"`
		ID          int             `json:"id"`
		ModORConfig dst.ModORConfig `json:"modORConfig"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	err = game.ModConfigureOptionsValuesChange(reqForm.WorldID, reqForm.ID, &reqForm.ModORConfig)
	if err != nil {
		logger.Logger.Error("修改模组设置失败")
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "modify mod configuration values error"), "data": nil})
		return
	}

	err = h.roomDao.UpdateRoom(room)
	if err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	err = h.worldDao.UpdateWorlds(worlds)
	if err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "modify mod configuration values success"), "data": nil})
}

func (h *Handler) addEnablePost(c *gin.Context) {
	type ReqForm struct {
		RoomID  int    `json:"roomID"`
		WorldID int    `json:"worldID"`
		ID      int    `json:"id"`
		FileURL string `json:"file_url"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	err = game.ModEnable(reqForm.WorldID, reqForm.ID, reqForm.FileURL == "")
	if err != nil {
		logger.Logger.Errorf("模组启用失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "mod enable fail"), "data": nil})
		return
	}

	err = h.roomDao.UpdateRoom(room)
	if err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	err = h.worldDao.UpdateWorlds(worlds)
	if err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "mod enable success"), "data": nil})
}

func (h *Handler) addDisablePost(c *gin.Context) {
	type ReqForm struct {
		RoomID  int `json:"roomID"`
		WorldID int `json:"worldID"`
		ID      int `json:"id"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	err = game.ModDisable(reqForm.ID)
	if err != nil {
		logger.Logger.Errorf("模组禁用失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "mod disable fail"), "data": nil})
		return
	}

	err = h.roomDao.UpdateRoom(room)
	if err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	err = h.worldDao.UpdateWorlds(worlds)
	if err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "mod disable success"), "data": nil})
}

func (h *Handler) getEnabledModsGet(c *gin.Context) {
	type ReqForm struct {
		RoomID  int `form:"roomID"`
		WorldID int `form:"worldID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	modsID, err := game.GetEnabledMods(reqForm.WorldID)
	if err != nil {
		logger.Logger.Error("获取模组设置失败")
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "get enabled mod fail"), "data": modsID})
		return
	}

	err = h.addDownloadedModInfo(&modsID, c.Request.Header.Get("X-I18n-Lang"))
	if err != nil {
		logger.Logger.Error("添加模组额外信息失败")
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": modsID})
}

func (h *Handler) deletePost(c *gin.Context) {
	type ReqForm struct {
		RoomID  int    `json:"roomID"`
		ID      int    `json:"id"`
		FileURL string `json:"file_url"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	err = game.ModDelete(reqForm.ID, reqForm.FileURL)
	if err != nil {
		logger.Logger.Errorf("删除模组失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "delete fail"), "data": nil})
		return
	}

	err = cache.ModDownloadStatus.Delete(reqForm.RoomID, reqForm.ID)
	if err != nil {
		logger.Logger.Errorf("删除模组下载状态缓存失败: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "delete success"), "data": nil})
}

func (h *Handler) acfDelete(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	err = game.DeleteAcf()
	if err != nil {
		logger.Logger.Errorf("删除acf文件失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "delete fail"), "data": nil})
		return
	}

	err = cache.ModDownloadStatus.DeleteByRoomID(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("删除模组下载状态缓存失败: %v", err)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "delete success"), "data": nil})
}
