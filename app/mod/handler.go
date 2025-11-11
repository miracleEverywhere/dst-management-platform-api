package mod

import (
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
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
	game.DownloadMod(reqForm.ID, reqForm.FileURL == "")

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "downloading"), "data": nil})
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

func (h *Handler) settingGet(c *gin.Context) {
	type ReqForm struct {
		RoomID  int    `form:"roomID"`
		WorldID int    `form:"worldID"`
		ID      int    `json:"id"`
		FileURL string `json:"file_url"`
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
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": "success", "data": options})
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": options})
}
