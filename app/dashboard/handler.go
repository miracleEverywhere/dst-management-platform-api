package dashboard

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/utils"
	"fmt"
	"net/http"
	"strconv"
	"time"

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

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
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

			// 如果需要重启，则重启激活的房间
			var globalSettings models.GlobalSetting
			err = h.globalSettingDao.GetGlobalSetting(&globalSettings)
			if err != nil {
				logger.Logger.Error("获取全局设置失败", "err", err)
				return
			}

			if !globalSettings.AutoUpdateRestart {
				return
			}

			roomBasic, err := h.roomDao.GetRoomBasic()
			if err != nil {
				logger.Logger.Error("获取全局房间信息失败", "err", err)
				return
			}
			for _, rb := range *roomBasic {
				if !rb.Status {
					continue
				}
				room, worlds, roomSetting, err = h.fetchGameInfo(rb.RoomID)
				if err != nil {
					logger.Logger.Error("获取基本信息失败", "err", err)
					continue
				}
				game = dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
				_ = game.StopAllWorld()
				_ = game.StartAllWorld()
				time.Sleep(5 * time.Second)
			}
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
		if reqForm.Extra == "" {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "announce fail"), "data": nil})
			return
		}
		err = game.Announce(reqForm.Extra)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "announce fail"), "data": nil})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "announce success"), "data": nil})
		return
	case "console":
		if reqForm.Extra == "" {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "exec fail"), "data": nil})
			return
		}
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
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
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
		Room        models.Room         `json:"room"`
		Worlds      []GameWorldInfo     `json:"worlds"`
		RoomSetting models.RoomSetting  `json:"roomSetting"`
		Session     dst.RoomSessionInfo `json:"session"`
		Players     []db.PlayerInfo     `json:"players"`
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
		Room:        *room,
		Worlds:      gameWorldInfo,
		RoomSetting: *roomSetting,
		Session:     *game.SessionInfo(),
		Players:     players,
	}})
}

func (h *Handler) infoSysGet(c *gin.Context) {
	type Data struct {
		Cpu      float64 `json:"cpu"`
		Memory   float64 `json:"memory"`
		Updating bool    `json:"updating"`
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": Data{
		Cpu:      utils.CpuUsage(),
		Memory:   utils.MemoryUsage(),
		Updating: db.DstUpdating,
	}})
}

func (h *Handler) connectionCodeGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var (
		reqForm ReqForm
		err     error
	)
	if err = c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.RoomID == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := h.fetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	var connectionCode string

	if roomSetting.CustomIP == "" {
		// 返回默认直连代码
		var (
			internetIp string
			masterPort int
		)

		if db.InternetIP == "" {
			internetIp, err = scheduler.GetInternetIP1()
			if err != nil {
				logger.Logger.Warn("调用公网ip接口1失败", "err", err)
				internetIp, err = scheduler.GetInternetIP2()
				if err != nil {
					logger.Logger.Warn("调用公网ip接口2失败", "err", err)
					c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "connection code fail"), "data": nil})
					return
				}
			}
			db.InternetIP = internetIp
		} else {
			logger.Logger.Debug("发现缓存的公网IP")
			internetIp = db.InternetIP
		}

		for _, world := range *worlds {
			if world.IsMaster {
				masterPort = world.ServerPort
			}
		}
		if masterPort == 0 {
			masterPort = (*worlds)[0].ServerPort
		}
		if room.Password == "" {
			connectionCode = fmt.Sprintf("c_connect('%s', %d)", internetIp, masterPort)
		} else {
			connectionCode = fmt.Sprintf("c_connect('%s', %d, '%s')", internetIp, masterPort, room.Password)
		}
	} else {
		// 返回自定义直连代码
		if room.Password == "" {
			connectionCode = fmt.Sprintf("c_connect('%s', %d)", roomSetting.CustomIP, roomSetting.CustomPort)
		} else {
			connectionCode = fmt.Sprintf("c_connect('%s', %d, '%s')", roomSetting.CustomIP, roomSetting.CustomPort, room.Password)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": connectionCode})
}

func (h *Handler) connectionCodePut(c *gin.Context) {
	type ReqForm struct {
		RoomID int    `json:"roomID"`
		IP     string `json:"ip"`
		Port   int    `json:"port"`
	}

	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	roomSetting, err := h.roomSettingDao.GetRoomSettingsByRoomID(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}
	roomSetting.CustomIP = reqForm.IP
	roomSetting.CustomPort = reqForm.Port

	err = h.roomSettingDao.UpdateRoomSetting(roomSetting)
	if err != nil {
		logger.Logger.Error("修改房间设置失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "update fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "update success"), "data": nil})
}

func checkLobbyPost(c *gin.Context) {
	type ReqForm struct {
		GameName  string   `json:"gameName"`
		MaxPlayer int      `json:"maxPlayer"`
		Regions   []string `json:"regions"`
	}

	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.GameName == "" || reqForm.MaxPlayer == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	var urls []string
	for _, region := range reqForm.Regions {
		urls = append(urls, getDSTRoomsApi(region))
	}
	rooms, err := checkDstLobbyRoom(urls, reqForm.GameName)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "check lobby fail"), "data": false})
		return
	}

	for _, room := range rooms {
		if room.MaxConnections == reqForm.MaxPlayer {
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": true})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": false})
}
