package dashboard

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/service"

	"github.com/gin-gonic/gin"
)

func (h *Handler) execGamePost(c *gin.Context) {
	type ReqForm struct {
		Type    string `json:"type"`
		RoomID  int    `json:"roomID"`
		WorldID int    `json:"worldID"`
		Extra   string `json:"extra"`
	}

	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeDashboardBadRequest))
		return
	}

	message.Respond(c, h.svc.ExecGame(middleware.Actor(c), reqForm.Type, reqForm.RoomID, reqForm.WorldID, reqForm.Extra))
}

func (h *Handler) infoBaseGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeDashboardBadRequest))
		return
	}

	message.Respond(c, h.svc.InfoBase(middleware.Actor(c), reqForm.RoomID))
}

func (h *Handler) infoSysGet(c *gin.Context) {
	message.Respond(c, h.svc.InfoSys())
}

func (h *Handler) connectionCodeGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeDashboardBadRequest))
		return
	}

	message.Respond(c, h.svc.ConnectionCode(middleware.Actor(c), reqForm.RoomID))
}

func (h *Handler) connectionCodePut(c *gin.Context) {
	type ReqForm struct {
		RoomID int    `json:"roomID"`
		IP     string `json:"ip"`
		Port   int    `json:"port"`
	}

	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeDashboardBadRequest))
		return
	}

	message.Respond(c, h.svc.ConnectionCodeUpdate(middleware.Actor(c), reqForm.RoomID, reqForm.IP, reqForm.Port))
}

func (h *Handler) checkLobbyPost(c *gin.Context) {
	type ReqForm struct {
		GameName  string   `json:"gameName"`
		MaxPlayer int      `json:"maxPlayer"`
		Regions   []string `json:"regions"`
	}

	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeDashboardBadRequest))
		return
	}

	message.Respond(c, h.svc.CheckLobby(reqForm.GameName, reqForm.MaxPlayer, reqForm.Regions))
}
