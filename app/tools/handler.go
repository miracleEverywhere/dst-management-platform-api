package tools

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/service"
	toolssvc "dst-management-platform-api/service/tools"

	"github.com/gin-gonic/gin"
)

func (h *Handler) backupGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeToolsBadRequest))
		return
	}

	message.Respond(c, h.svc.BackupList(middleware.Actor(c), reqForm.RoomID))
}

func (h *Handler) backupPost(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeToolsBadRequest))
		return
	}

	message.Respond(c, h.svc.BackupCreate(middleware.Actor(c), reqForm.RoomID))
}

func (h *Handler) backupDelete(c *gin.Context) {
	type ReqForm struct {
		RoomID    int      `json:"roomID"`
		Filenames []string `json:"filenames"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeToolsBadRequest))
		return
	}

	message.Respond(c, h.svc.BackupDelete(middleware.Actor(c), reqForm.RoomID, reqForm.Filenames))
}

func (h *Handler) backupRestorePost(c *gin.Context) {
	type ReqForm struct {
		RoomID   int    `json:"roomID"`
		Filename string `json:"filename"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeToolsBadRequest))
		return
	}

	message.Respond(c, h.svc.BackupRestore(middleware.Actor(c), reqForm.RoomID, reqForm.Filename))
}

func (h *Handler) backupDownloadGet(c *gin.Context) {
	// 1. 获取路径参数
	type ReqForm struct {
		RoomID   int    `json:"roomID" form:"roomID"`
		Filename string `json:"filename" form:"filename"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeToolsBadRequest))
		return
	}

	filePath, fileName, result := h.svc.BackupDownload(middleware.Actor(c), reqForm.RoomID, reqForm.Filename)
	if result.Code != 200 {
		message.Respond(c, result)
		return
	}

	// service 只负责落盘路径与下载文件名，Content-Disposition 仍由 gin 按原逻辑写
	c.FileAttachment(filePath, fileName)
}

func (h *Handler) announceGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeToolsBadRequest))
		return
	}

	message.Respond(c, h.svc.AnnounceGet(middleware.Actor(c), reqForm.RoomID))
}

func (h *Handler) announcePut(c *gin.Context) {
	type ReqForm struct {
		RoomID  int    `json:"roomID"`
		Setting string `json:"setting"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeToolsBadRequest))
		return
	}

	message.Respond(c, h.svc.AnnouncePut(middleware.Actor(c), reqForm.RoomID, reqForm.Setting))
}

func (h *Handler) tokenPost(c *gin.Context) {
	type ReqForm struct {
		Expiration int `json:"expiration"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeToolsBadRequest))
		return
	}

	message.Respond(c, h.svc.Token(middleware.Actor(c), reqForm.Expiration))
}

func (h *Handler) snapshotGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeToolsBadRequest))
		return
	}

	message.Respond(c, h.svc.SnapshotGet(middleware.Actor(c), reqForm.RoomID))
}

func (h *Handler) snapshotDelete(c *gin.Context) {
	type ReqForm struct {
		RoomID int    `json:"roomID"`
		Name   string `json:"name"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeToolsBadRequest))
		return
	}

	message.Respond(c, h.svc.SnapshotDelete(middleware.Actor(c), reqForm.RoomID, reqForm.Name))
}

func (h *Handler) categoryGet(c *gin.Context) {
	message.Respond(c, h.svc.TMICategoryList())
}

func (h *Handler) categoryItemsGet(c *gin.Context) {
	var reqForm struct {
		Category string `json:"category" form:"category"`
		Page     int    `json:"page" form:"page"`
		PageSize int    `json:"pageSize" form:"pageSize"`
	}
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeToolsBadRequest))
		return
	}

	message.Respond(c, h.svc.TMICategoryItems(reqForm.Category, reqForm.Page, reqForm.PageSize))
}

func (h *Handler) consolePost(c *gin.Context) {
	var reqForm struct {
		RoomID  int    `json:"roomID" binding:"required"`
		WorldID int    `json:"worldID" binding:"required"`
		Type    string `json:"type" binding:"required"`
		Uid     string `json:"uid"`
		Prefab  string `json:"prefab"`
		Num     int    `json:"num"`
	}
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeToolsBadRequest))
		return
	}

	message.Respond(c, h.svc.Console(middleware.Actor(c), reqForm.RoomID, reqForm.WorldID, reqForm.Type, reqForm.Uid, reqForm.Prefab, reqForm.Num))
}

func (h *Handler) aiSettingGet(c *gin.Context) {
	var reqForm struct {
		RoomID int `form:"roomID" binding:"required"`
	}
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeToolsBadRequest))
		return
	}

	message.Respond(c, h.svc.AISettingGet(middleware.Actor(c), reqForm.RoomID))
}

func (h *Handler) aiSettingPut(c *gin.Context) {
	var reqForm toolssvc.AISettingPutInput
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeToolsBadRequest))
		return
	}

	message.Respond(c, h.svc.AISettingPut(middleware.Actor(c), reqForm))
}

func (h *Handler) aiBaseSettingGet(c *gin.Context) {
	message.Respond(c, h.svc.AIBaseSettingGet())
}

func (h *Handler) aiBaseSettingPut(c *gin.Context) {
	var reqForm models.AIBaseSetting
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeToolsBadRequest))
		return
	}

	message.Respond(c, h.svc.AIBaseSettingPut(middleware.Actor(c), reqForm))
}

func (h *Handler) aiKeywordIndexReBuild(c *gin.Context) {
	message.Respond(c, h.svc.AIKeywordIndexRebuild())
}

func (h *Handler) aiEmbeddingIndexReBuild(c *gin.Context) {
	message.Respond(c, h.svc.AIEmbeddingIndexRebuild())
}

func (h *Handler) aiEmbeddingIndexCancel(c *gin.Context) {
	message.Respond(c, h.svc.AIEmbeddingIndexCancel())
}

func (h *Handler) mapGet(c *gin.Context) {
	type ReqForm struct {
		RoomID  int    `form:"roomID"`
		WorldID int    `form:"worldID"`
		Players string `form:"players"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeToolsBadRequest))
		return
	}

	result := h.svc.Map(middleware.Actor(c), c.Request.Context(), reqForm.RoomID, reqForm.WorldID, reqForm.Players)
	// 只有成功生成地图时才禁止缓存，保持与原实现的响应头一致
	if result.Code == 200 {
		c.Header("Cache-Control", "no-store")
	}
	message.Respond(c, result)
}

func (h *Handler) mapPlayersGet(c *gin.Context) {
	type ReqForm struct {
		RoomID  int `form:"roomID"`
		WorldID int `form:"worldID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeToolsBadRequest))
		return
	}

	result := h.svc.MapPlayers(middleware.Actor(c), c.Request.Context(), reqForm.RoomID, reqForm.WorldID)
	// 只有成功刷新玩家位置时才禁止缓存，保持与原实现的响应头一致
	if result.Code == 200 {
		c.Header("Cache-Control", "no-store")
	}
	message.Respond(c, result)
}
