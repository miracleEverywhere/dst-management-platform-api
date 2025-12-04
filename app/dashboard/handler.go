package dashboard

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
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

	switch reqForm.Type {
	case "startup":
		// 启动
		if reqForm.Extra == "all" {
			err = game.StartAllWorld()
			if err != nil {
				logger.Logger.Error("启动失败", "err", err)
				c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "startup game fail"), "data": nil})
				return
			}
		} else {
			err = game.StartWorld(reqForm.WorldID)
			if err != nil {
				logger.Logger.Error("启动失败", "err", err)
				c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "startup game fail"), "data": nil})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "startup game success"), "data": nil})
		return
	case "shutdown":
		// 关闭
		if reqForm.Extra == "all" {
			err = game.StopAllWorld()
			if err != nil {
				logger.Logger.Error("关闭失败", "err", err)
				c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "shutdown game fail"), "data": nil})
				return
			}
		} else {
			err = game.StopWorld(reqForm.WorldID)
			if err != nil {
				logger.Logger.Error("关闭失败", "err", err)
				c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "shutdown game fail"), "data": nil})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "shutdown game success"), "data": nil})
		return
	case "restart":
		// 重启
		_ = game.StopAllWorld()
		err = game.StartAllWorld()
		if err != nil {
			logger.Logger.Error("启动失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "restart game fail"), "data": nil})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "restart game success"), "data": nil})
		return
	case "update":
		// 更新，需要管理员权限
		role, _ := c.Get("role")
		if role.(string) != "admin" {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
			return
		}

		go func() {
			db.DstUpdating = true
			updateCmd := fmt.Sprintf("cd ~/steamcmd && ./steamcmd.sh +login anonymous +force_install_dir ~/dst +app_update 343050 validate +quit")
			_ = utils.BashCMD(updateCmd)
			db.DstUpdating = false
		}()

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "updating"), "data": nil})
		return
	case "reset":
		if reqForm.Extra == "force" {
			err = game.Reset(true)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "reset game fail"), "data": nil})
				return
			}
		} else {
			err = game.Reset(false)
			if err != nil {
				c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "reset game fail"), "data": nil})
				return
			}
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "reset game success"), "data": nil})
		return
	case "delete":
		err = game.DeleteWorld(reqForm.WorldID)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "delete game fail"), "data": nil})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "delete game success"), "data": nil})
		return
	case "announce":
		err = game.Announce(reqForm.Extra)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "announce fail"), "data": nil})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "announce success"), "data": nil})
		return
	case "console":
		err = game.ConsoleCmd(reqForm.Extra, reqForm.WorldID)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "exec fail"), "data": nil})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "exec success"), "data": nil})
		return
	default:
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
}

func (h *Handler) infoBaseGet(c *gin.Context) {
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

	type GameWorldInfo struct {
		*models.World
		Status            bool                  `json:"status"`
		PerformanceStatus dst.PerformanceStatus `json:"performanceStatus"`
	}

	var gameWorldInfo []GameWorldInfo

	for _, world := range *worlds {
		gameWorldInfo = append(gameWorldInfo, GameWorldInfo{
			World:             &world,
			Status:            game.WorldUpStatus(world.ID),
			PerformanceStatus: game.WorldPerformanceStatus(world.ID),
		})
	}

	type Data struct {
		Room         models.Room         `json:"room"`
		Worlds       []GameWorldInfo     `json:"worlds"`
		WorldSetting models.RoomSetting  `json:"worldSetting"`
		Session      dst.RoomSessionInfo `json:"session"`
		Players      []db.PlayerInfo     `json:"players"`
	}

	db.PlayersStatisticMutex.Lock()
	defer db.PlayersStatisticMutex.Unlock()

	var players []db.PlayerInfo

	if len(db.PlayersStatistic[reqForm.RoomID]) > 0 {
		players = db.PlayersStatistic[reqForm.RoomID][len(db.PlayersStatistic[reqForm.RoomID])-1].PlayerInfo
	} else {
		players = []db.PlayerInfo{}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": Data{
		Room:         *room,
		Worlds:       gameWorldInfo,
		WorldSetting: *roomSetting,
		Session:      *game.SessionInfo(),
		Players:      players,
	}})
}

func (h *Handler) infoSysGet(c *gin.Context) {
	type Data struct {
		Cpu      float64 `json:"cpu"`
		Memory   float64 `json:"memory"`
		Updating bool    `json:"updating"`
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": Data{
		Cpu:      cpuUsage(),
		Memory:   memoryUsage(),
		Updating: db.DstUpdating,
	}})
}
