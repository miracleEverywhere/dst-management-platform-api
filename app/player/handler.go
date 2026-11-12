package player

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/service"
	playersvc "dst-management-platform-api/service/player"
	"dst-management-platform-api/utils"

	"github.com/gin-gonic/gin"
)

func (h *Handler) onlineGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlayerBadRequest))
		return
	}

	message.Respond(c, h.svc.Online(middleware.Actor(c), reqForm.RoomID))
}

func (h *Handler) ganttGet(c *gin.Context) {
	type ReqForm struct {
		RoomID    int `json:"roomID" form:"roomID"`
		TimeRange int `json:"timeRange" form:"timeRange"` // 最近多少小时
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlayerBadRequest))
		return
	}

	message.Respond(c, h.svc.Gantt(middleware.Actor(c), reqForm.RoomID, reqForm.TimeRange))
}

func (h *Handler) listPost(c *gin.Context) {
	type ReqForm struct {
		RoomID     int      `json:"roomID"`
		UIDS       []string `json:"uids"`
		ListType   string   `json:"listType"`
		ActionType string   `json:"actionType"`
	}

	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlayerBadRequest))
		return
	}

	message.Respond(c, h.svc.UpdateList(middleware.Actor(c), reqForm.RoomID, reqForm.UIDS, reqForm.ListType, reqForm.ActionType))
}

func (h *Handler) listGet(c *gin.Context) {
	type ReqForm struct {
		RoomID   int    `json:"roomID" form:"roomID"`
		ListType string `json:"listType" form:"listType"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlayerBadRequest))
		return
	}
	logger.Logger.Debug(utils.StructToFlatString(reqForm))

	message.Respond(c, h.svc.List(middleware.Actor(c), reqForm.RoomID, reqForm.ListType))
}

func (h *Handler) uidMapGet(c *gin.Context) {
	type ReqForm struct {
		RoomID   int    `json:"roomID" form:"roomID"`
		Q        string `json:"q" form:"q"`
		Page     int    `json:"page" form:"page"`
		PageSize int    `json:"pageSize" form:"pageSize"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Result{Code: constant.CodePlayerBadRequest, Data: playersvc.PaginatedUidMap{}})
		return
	}

	message.Respond(c, h.svc.UIDMap(middleware.Actor(c), reqForm.RoomID, reqForm.Q, reqForm.Page, reqForm.PageSize))
}

func (h *Handler) statisticsOnlineTimeGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlayerBadRequest))
		return
	}

	message.Respond(c, h.svc.StatisticsOnlineTime(middleware.Actor(c), reqForm.RoomID))
}

func (h *Handler) statisticsPlayerCountGet(c *gin.Context) {
	type ReqForm struct {
		RoomID    int `json:"roomID" form:"roomID"`
		TimeRange int `json:"timeRange" form:"timeRange"` // 前端传回来需要多少秒的数据
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlayerBadRequest))
		return
	}

	message.Respond(c, h.svc.StatisticsPlayerCount(middleware.Actor(c), reqForm.RoomID, reqForm.TimeRange))
}

func (h *Handler) statisticsPlayerCountV2Get(c *gin.Context) {
	type ReqForm struct {
		RoomID    int `json:"roomID" form:"roomID"`
		TimeRange int `json:"timeRange" form:"timeRange"` // 前端传回来需要多少秒的数据
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlayerBadRequest))
		return
	}

	message.Respond(c, h.svc.StatisticsPlayerCountV2(middleware.Actor(c), reqForm.RoomID, reqForm.TimeRange))
}

func (h *Handler) chatGet(c *gin.Context) {
	type ReqForm struct {
		RoomID   int  `json:"roomID" form:"roomID"`
		Lines    int  `json:"lines" form:"lines"`
		NeedTime bool `json:"needTime" form:"needTime"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlayerBadRequest))
		return
	}
	logger.Logger.Debug(utils.StructToFlatString(reqForm))

	message.Respond(c, h.svc.Chat(middleware.Actor(c), reqForm.RoomID, reqForm.Lines, reqForm.NeedTime))
}
