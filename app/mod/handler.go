package mod

import (
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func modSearchGet(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	type SearchForm struct {
		SearchType string `form:"searchType" json:"searchType"`
		SearchText string `form:"searchText" json:"searchText"`
		Page       int    `form:"page" json:"page"`
		PageSize   int    `form:"pageSize" json:"pageSize"`
	}
	var searchForm SearchForm
	if err := c.ShouldBindQuery(&searchForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	logger.Logger.Debug(utils.StructToFlatString(searchForm))

	if searchForm.SearchType == "id" {
		id, err := strconv.Atoi(searchForm.SearchText)
		if err != nil {
			logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
			return
		}
		data, err := SearchModById(id, langStr)
		if err != nil {
			logger.Logger.Error("获取mod信息失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "search fail"), "data": nil})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
		return
	}
	if searchForm.SearchType == "text" {
		data, err := SearchMod(searchForm.Page, searchForm.PageSize, searchForm.SearchText, langStr)
		if err != nil {
			logger.Logger.Error("获取mod信息失败", "err", err)
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
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	reqSize, err := strconv.Atoi(reqForm.Size)
	if err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	reqSize64 := int64(reqSize)

	room, worlds, roomSetting, err := h.fetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	err, modSize := game.DownloadMod(reqForm.ID, reqForm.FileURL)
	if err != nil || modSize != reqSize64 {
		logger.Logger.DebugF("模组大小与预期不符, %d, %d", modSize, reqSize64)
		if reqForm.Update {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": reqForm.Name + " " + message.Get(c, "update fail"), "data": nil})
			return
		} else {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": reqForm.Name + " " + message.Get(c, "download fail"), "data": nil})
			return
		}
	}

	if reqForm.Update {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": reqForm.Name + " " + message.Get(c, "update success"), "data": nil})
	} else {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": reqForm.Name + " " + message.Get(c, "download success"), "data": nil})
	}
}

func (h *Handler) downloadedModsGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := h.fetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	downloadedMods := game.GetDownloadedMods()

	err = addDownloadedModInfo(downloadedMods, c.Request.Header.Get("X-I18n-Lang"))
	if err != nil {
		logger.Logger.Error("添加模组额外信息失败")
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": downloadedMods})
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
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := h.fetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
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
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := h.fetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
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
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := h.fetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
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
		logger.Logger.Error("更新房间失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	err = h.worldDao.UpdateWorlds(worlds)
	if err != nil {
		logger.Logger.Error("更新房间失败", "err", err)
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
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := h.fetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	err = game.ModEnable(reqForm.WorldID, reqForm.ID, reqForm.FileURL == "")
	if err != nil {
		logger.Logger.Error("模组启用失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "mod enable fail"), "data": nil})
		return
	}

	err = h.roomDao.UpdateRoom(room)
	if err != nil {
		logger.Logger.Error("更新房间失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	err = h.worldDao.UpdateWorlds(worlds)
	if err != nil {
		logger.Logger.Error("更新房间失败", "err", err)
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
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := h.fetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	err = game.ModDisable(reqForm.ID)
	if err != nil {
		logger.Logger.Error("模组禁用失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "mod disable fail"), "data": nil})
		return
	}

	err = h.roomDao.UpdateRoom(room)
	if err != nil {
		logger.Logger.Error("更新房间失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	err = h.worldDao.UpdateWorlds(worlds)
	if err != nil {
		logger.Logger.Error("更新房间失败", "err", err)
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
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := h.fetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
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

	err = addDownloadedModInfo(&modsID, c.Request.Header.Get("X-I18n-Lang"))
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
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := h.fetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	err = game.ModDelete(reqForm.ID, reqForm.FileURL)
	if err != nil {
		logger.Logger.Error("删除模组失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "delete fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "delete success"), "data": nil})
}
