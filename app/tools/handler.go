package tools

import (
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
)

func (h *Handler) backupGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
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

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 420, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := h.fetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	backups, err := game.GetBackups()
	if err != nil {
		logger.Logger.Error("获取备份文件失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "get backup fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": backups})
}

func (h *Handler) backupPost(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.RoomID == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 420, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := h.fetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	err = game.Backup()
	if err != nil {
		logger.Logger.Error("创建备份文件失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "create backup fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "create backup success"), "data": nil})
}

func (h *Handler) backupDelete(c *gin.Context) {
	type ReqForm struct {
		RoomID    int      `json:"roomID"`
		Filenames []string `json:"filenames"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.RoomID == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 420, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := h.fetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	count := game.DeleteBackups(reqForm.Filenames)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "?", "data": count})
}

func (h *Handler) backupRestorePost(c *gin.Context) {
	type ReqForm struct {
		RoomID   int    `json:"roomID"`
		Filename string `json:"filename"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.RoomID == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 420, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := h.fetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	saveData, err := game.Restore(reqForm.Filename)
	if err != nil {
		logger.Logger.Error("恢复失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "restore fail"), "data": nil})
		return
	}

	err = h.roomDao.UpdateRoom(&saveData.Room)
	if err != nil {
		logger.Logger.Error("更新房间失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "restore fail"), "data": nil})
		return
	}

	err = h.worldDao.UpdateWorlds(&saveData.Worlds)
	if err != nil {
		logger.Logger.Error("更新房间失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "restore fail"), "data": nil})
		return
	}

	err = h.roomSettingDao.UpdateRoomSetting(&saveData.RoomSetting)
	if err != nil {
		logger.Logger.Error("更新房间失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "restore fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "restore success"), "data": nil})
}

func (h *Handler) backupDownloadGet(c *gin.Context) {
	// 1. 获取路径参数
	type ReqForm struct {
		RoomID   int    `json:"roomID" form:"roomID"`
		Filename string `json:"filename" form:"filename"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	// 2. 参数验证
	if reqForm.RoomID == 0 || reqForm.Filename == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	// 3. 安全验证（防止路径遍历）
	if strings.Contains(reqForm.Filename, "..") {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	// 4. 构建文件路径
	filePath := fmt.Sprintf("dmp_files/backup/%d/%s", reqForm.RoomID, reqForm.Filename)

	c.File(filePath)
}
