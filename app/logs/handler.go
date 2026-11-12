package logs

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/service"
	nethttp "net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) contentGet(c *gin.Context) {
	type ReqForm struct {
		RoomID  int    `json:"roomID" form:"roomID"`
		WorldID int    `json:"worldID" form:"worldID"`
		LogType string `json:"logType" form:"logType"`
		Lines   int    `json:"lines" form:"lines"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeLogsBadRequest))
		return
	}

	message.Respond(c, h.svc.Content(middleware.Actor(c), reqForm.RoomID, reqForm.WorldID, reqForm.LogType, reqForm.Lines))
}

func (h *Handler) historyListGet(c *gin.Context) {
	type ReqForm struct {
		RoomID  int    `json:"roomID" form:"roomID"`
		WorldID int    `json:"worldID" form:"worldID"`
		LogType string `json:"logType" form:"logType"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeLogsBadRequest))
		return
	}

	message.Respond(c, h.svc.HistoryList(middleware.Actor(c), reqForm.RoomID, reqForm.WorldID, reqForm.LogType))
}

func (h *Handler) historyContentGet(c *gin.Context) {
	type ReqForm struct {
		RoomID  int    `json:"roomID" form:"roomID"`
		WorldID int    `json:"worldID" form:"worldID"`
		LogType string `json:"logType" form:"logType"`
		LogFile string `json:"logFile" form:"logFile"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeLogsBadRequest))
		return
	}

	message.Respond(c, h.svc.HistoryContent(middleware.Actor(c), reqForm.RoomID, reqForm.WorldID, reqForm.LogType, reqForm.LogFile))
}

func (h *Handler) cleanInfoGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeLogsBadRequest))
		return
	}

	message.Respond(c, h.svc.CleanInfo(middleware.Actor(c), reqForm.RoomID))
}

func (h *Handler) cleanDelete(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
		dst.CleanLogs
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeLogsBadRequest))
		return
	}

	clean := dst.CleanLogs{
		Game:    reqForm.Game,
		Chat:    reqForm.Chat,
		Steam:   reqForm.Steam,
		Access:  reqForm.Access,
		Runtime: reqForm.Runtime,
	}
	message.Respond(c, h.svc.Clean(middleware.Actor(c), reqForm.RoomID, clean))
}

func (h *Handler) downloadGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Result{Code: 400, Status: nethttp.StatusBadRequest, MsgKey: "bad request"})
		return
	}

	zipFile, cleanup, result := h.svc.Download(middleware.Actor(c), reqForm.RoomID)
	defer cleanup()

	if result.Code != 200 {
		message.Respond(c, result)
		return
	}

	c.File(zipFile)
}
