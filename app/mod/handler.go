package mod

import (
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *Handler) downloadPost(c *gin.Context) {
	type ReqForm struct {
		RoomID  int    `json:"roomID"`
		ModID   int    `json:"modID"`
		FileURL string `json:"fileURL"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	room, err := h.roomDao.GetRoomByID(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取房间失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}
	worlds, err := h.worldDao.GetWorldsByRoomID(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取世界失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}
	roomSetting, err := h.roomSettingDao.GetRoomSettingsByRoomID(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取房间设置失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	game.DownloadMod(reqForm.ModID, reqForm.FileURL == "")

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "downloading"), "data": nil})
}
