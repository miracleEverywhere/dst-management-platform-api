package logs

import (
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"net/http"

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
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.LogType == "game" || reqForm.LogType == "chat" {
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
	} else {
		var logPath string
		switch reqForm.LogType {
		case "steam":
			logPath = "Steam/logs/bootstrap_log.txt"
		case "access":
			logPath = "logs/access.log"
		case "runtime":
			logPath = "logs/runtime.log"
		}

		logContent := utils.GetFileLastNLines(logPath, reqForm.Lines)
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": logContent})
	}
}

func (h *Handler) historyListGet(c *gin.Context) {
	type ReqForm struct {
		RoomID  int    `json:"roomID" form:"roomID"`
		WorldID int    `json:"worldID" form:"worldID"`
		LogType string `json:"logType" form:"logType"`
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
	list := game.HistoryFileList(reqForm.LogType, reqForm.WorldID)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": list})
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
	content := game.HistoryFileContent(reqForm.LogType, reqForm.LogFile, reqForm.WorldID)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": content})
}

func (h *Handler) cleanInfoGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
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

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": game.LogsInfo()})
}

func (h *Handler) cleanDelete(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
		dst.CleanLogs
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
	cl := dst.CleanLogs{
		Game:    reqForm.Game,
		Chat:    reqForm.Chat,
		Steam:   reqForm.Steam,
		Access:  reqForm.Access,
		Runtime: reqForm.Runtime,
	}

	if game.LogsClean(&cl) {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "delete success"), "data": nil})
	} else {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "delete fail"), "data": nil})
	}
}

func (h *Handler) downloadGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := h.fetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	role, _ := c.Get("role")

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))

	fileList := game.LogsList(role.(string) == "admin")
	zipFilePath := fmt.Sprintf("%s/tmp/%d", utils.DmpFiles, reqForm.RoomID)

	defer func(dirPath string) {
		err := utils.RemoveDir(dirPath)
		if err != nil {
			logger.Logger.Error(err.Error())
		}
	}(zipFilePath)

	err = utils.EnsureDirExists(zipFilePath)
	if err != nil {
		logger.Logger.Error("创建目录失败", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": message.Get(c, "download fail"), "data": nil})
		return
	}
	zipFile := fmt.Sprintf("%s/logs.zip", zipFilePath)

	err = utils.ZipFiles(fileList, zipFile)
	if err != nil {
		logger.Logger.Error("创建压缩文件失败", "err", err)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "message": message.Get(c, "download fail"), "data": nil})
		return
	}

	c.File(zipFile)
}
