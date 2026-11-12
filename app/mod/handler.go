package mod

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) modSearchGet(c *gin.Context) {
	type SearchForm struct {
		SearchType string `form:"searchType" json:"searchType"`
		SearchText string `form:"searchText" json:"searchText"`
		Page       int    `form:"page" json:"page"`
		PageSize   int    `form:"pageSize" json:"pageSize"`
	}
	var searchForm SearchForm
	if err := c.ShouldBindQuery(&searchForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeModBadRequest))
		return
	}
	logger.Logger.Debug(utils.StructToFlatString(searchForm))

	// 非法的搜索类型保留原有响应形状，与业务返回结构不同，故留在 handler 处理
	if searchForm.SearchType != "id" && searchForm.SearchType != "text" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}

	message.Respond(c, h.svc.ModSearch(middleware.Actor(c), searchForm.SearchType, searchForm.SearchText, searchForm.Page, searchForm.PageSize))
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
		message.Respond(c, service.Fail(constant.CodeModBadRequest))
		return
	}

	message.Respond(c, h.svc.ModDownload(middleware.Actor(c), reqForm.RoomID, reqForm.ID, reqForm.FileURL, reqForm.Size, reqForm.Name, reqForm.Sync))
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
		message.Respond(c, service.Result{Code: constant.CodeModBadRequest, Data: 0})
		return
	}
	if reqForm.ID <= 0 {
		reqForm.ID = reqForm.ModID
	}

	message.Respond(c, h.svc.ModDownloadStatus(middleware.Actor(c), reqForm.RoomID, reqForm.ID))
}

func (h *Handler) downloadedModsGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeModBadRequest))
		return
	}

	message.Respond(c, h.svc.DownloadedMods(middleware.Actor(c), reqForm.RoomID))
}

func (h *Handler) downloadedModIDsGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Result{Code: constant.CodeModBadRequest, Data: []int{}})
		return
	}

	message.Respond(c, h.svc.DownloadedModIDs(middleware.Actor(c), reqForm.RoomID))
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
		message.Respond(c, service.Fail(constant.CodeModBadRequest))
		return
	}

	message.Respond(c, h.svc.SettingModConfigStruct(middleware.Actor(c), reqForm.RoomID, reqForm.WorldID, reqForm.ID, reqForm.FileURL))
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
		message.Respond(c, service.Fail(constant.CodeModBadRequest))
		return
	}

	message.Respond(c, h.svc.SettingModConfigValue(middleware.Actor(c), reqForm.RoomID, reqForm.WorldID, reqForm.ID, reqForm.FileURL))
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
		message.Respond(c, service.Fail(constant.CodeModBadRequest))
		return
	}

	message.Respond(c, h.svc.SettingModConfigValuePut(middleware.Actor(c), reqForm.RoomID, reqForm.WorldID, reqForm.ID, reqForm.ModORConfig))
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
		message.Respond(c, service.Fail(constant.CodeModBadRequest))
		return
	}

	message.Respond(c, h.svc.AddEnable(middleware.Actor(c), reqForm.RoomID, reqForm.WorldID, reqForm.ID, reqForm.FileURL))
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
		message.Respond(c, service.Fail(constant.CodeModBadRequest))
		return
	}

	message.Respond(c, h.svc.AddDisable(middleware.Actor(c), reqForm.RoomID, reqForm.WorldID, reqForm.ID))
}

func (h *Handler) getEnabledModsGet(c *gin.Context) {
	type ReqForm struct {
		RoomID  int `form:"roomID"`
		WorldID int `form:"worldID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeModBadRequest))
		return
	}

	message.Respond(c, h.svc.EnabledMods(middleware.Actor(c), reqForm.RoomID, reqForm.WorldID))
}

func (h *Handler) modInfoPost(c *gin.Context) {
	type ReqForm struct {
		ModIDs []int `json:"modIDs"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeModBadRequest))
		return
	}

	message.Respond(c, h.svc.ModInfo(middleware.Actor(c), reqForm.ModIDs))
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
		message.Respond(c, service.Fail(constant.CodeModBadRequest))
		return
	}

	message.Respond(c, h.svc.Delete(middleware.Actor(c), reqForm.RoomID, reqForm.ID, reqForm.FileURL))
}

func (h *Handler) deletePostMulti(c *gin.Context) {
	var reqForm struct {
		RoomID int                 `json:"roomID"`
		Mods   []dst.DeleteModInfo `json:"mods"`
	}
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeModBadRequest))
		return
	}

	message.Respond(c, h.svc.DeleteMulti(middleware.Actor(c), reqForm.RoomID, reqForm.Mods))
}

func (h *Handler) acfDelete(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeModBadRequest))
		return
	}

	message.Respond(c, h.svc.DeleteAcf(middleware.Actor(c), reqForm.RoomID))
}
