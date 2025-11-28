package logs

import (
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"github.com/gin-gonic/gin"
	"net/http"
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
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.RoomID == 0 {
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

	logContent := game.LogContent(reqForm.LogType, reqForm.WorldID, reqForm.Lines)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": logContent})
}
