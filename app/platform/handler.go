package platform

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *Handler) overviewGet(c *gin.Context) {
	type Data struct {
		RunningTime int64   `json:"runningTime"`
		Memory      uint64  `json:"memory"`
		RoomCount   int64   `json:"roomCount"`
		WorldCount  int64   `json:"worldCount"`
		UserCount   int64   `json:"userCount"`
		UidCount    int64   `json:"uidCount"`
		MaxCpu      float64 `json:"maxCpu"`
		MaxMemory   float64 `json:"maxMemory"`
		MaxNetUp    float64 `json:"maxNetUp"`
		MaxNetDown  float64 `json:"maxNetDown"`
	}

	// 运行时间
	t := time.Since(utils.StartTime).Seconds()
	// 内存占用
	mem := getRES()
	// 房间数
	roomCount, err := h.roomDao.Count(nil)
	if err != nil {
		logger.Logger.Error("统计房间数失败")
		roomCount = 0
	}
	// 世界数
	worldCount, err := h.worldDao.Count(nil)
	if err != nil {
		logger.Logger.Error("统计世界数失败")
		worldCount = 0
	}
	// 用户数
	userCount, err := h.userDao.Count(nil)
	if err != nil {
		logger.Logger.Error("统计用户数失败")
		userCount = 0
	}
	// uid数
	uidCount, err := h.uidMapDao.Count(nil)
	if err != nil {
		logger.Logger.Error("统计用户数失败")
		uidCount = 0
	}
	// 1小时cpu内存网络最大值
	db.SystemMetricsMutex.RLock()
	systemMetricsLength := len(db.SystemMetrics)
	reqLength := 60
	var systemMetricsData []db.SysMetrics
	if systemMetricsLength > reqLength {
		systemMetricsData = db.SystemMetrics[systemMetricsLength-reqLength:]
	} else {
		systemMetricsData = db.SystemMetrics
	}
	db.SystemMetricsMutex.RUnlock()
	var maxCpu, maxMemory, maxNetUp, maxNetDown float64
	for _, m := range systemMetricsData {
		if m.Cpu > maxCpu {
			maxCpu = m.Cpu
		}
		if m.Memory > maxMemory {
			maxMemory = m.Memory
		}
		if m.NetUplink > maxNetUp {
			maxNetUp = m.NetUplink
		}
		if m.NetDownlink > maxNetDown {
			maxNetDown = m.NetDownlink
		}
	}

	// TODO 玩家数最多的的房间Top3

	data := Data{
		RunningTime: int64(t),
		Memory:      mem,
		RoomCount:   roomCount,
		WorldCount:  worldCount,
		UserCount:   userCount,
		UidCount:    uidCount,
		MaxCpu:      maxCpu,
		MaxMemory:   maxMemory,
		MaxNetUp:    maxNetUp,
		MaxNetDown:  maxNetDown,
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}

func gameVersionGet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": scheduler.GetDSTVersion()})
}

func osInfoGet(c *gin.Context) {
	osInfo, err := getOSInfo()
	if err != nil {
		logger.Logger.Errorf("获取系统信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "get os info fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": osInfo})
}

func metricsGet(c *gin.Context) {
	type ReqForm struct {
		TimeRange int `json:"timeRange" form:"timeRange"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	db.SystemMetricsMutex.RLock()
	systemMetricsLength := len(db.SystemMetrics)
	reqLength := reqForm.TimeRange * 60
	if reqLength <= 0 {
		reqLength = 60 // 默认1小时
	}

	if systemMetricsLength > reqLength {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": db.SystemMetrics[systemMetricsLength-reqLength:]})
	} else {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": db.SystemMetrics})
	}
	db.SystemMetricsMutex.RUnlock()
}

func (h *Handler) globalSettingsGet(c *gin.Context) {
	var globalSettings models.GlobalSetting

	err := h.globalSettingDao.GetGlobalSetting(&globalSettings)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": globalSettings})
}

func (h *Handler) globalSettingsPost(c *gin.Context) {
	var reqForm models.GlobalSetting
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	var dbGlobalSettings models.GlobalSetting

	err := h.globalSettingDao.GetGlobalSetting(&dbGlobalSettings)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	needUpdateDB := false

	if dbGlobalSettings.PlayerGetFrequency != reqForm.PlayerGetFrequency || dbGlobalSettings.PlayerInfoSaveTime != reqForm.PlayerInfoSaveTime || dbGlobalSettings.UIDMaintainEnable != reqForm.UIDMaintainEnable {
		needUpdateDB = true
		err = scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     "onlinePlayerGet",
			Func:     scheduler.OnlinePlayerGet,
			Args:     []any{reqForm.PlayerGetFrequency, reqForm.PlayerInfoSaveTime, reqForm.UIDMaintainEnable},
			TimeType: scheduler.SecondType,
			Interval: reqForm.PlayerGetFrequency,
			DayAt:    "",
		})

		db.PlayersStatisticMutex.Lock()
		for roomID := range db.PlayersStatistic {
			if len(db.PlayersStatistic[roomID])*reqForm.PlayerGetFrequency > scheduler.ParsePlayerInfoSaveTime(reqForm.PlayerInfoSaveTime) {
				n := int(scheduler.ParsePlayerInfoSaveTime(reqForm.PlayerInfoSaveTime) / reqForm.PlayerGetFrequency)
				db.PlayersStatistic[roomID] = utils.GetLastNElements(db.PlayersStatistic[roomID], n)
			}
		}
		db.PlayersStatisticMutex.Unlock()

		if err != nil {
			logger.Logger.Errorf("定时任务设置失败, err: %v, name: %v", err, "onlinePlayerGet")
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "update fail"), "data": nil})
			return
		}
	}

	if dbGlobalSettings.SysMetricsEnable != reqForm.SysMetricsEnable || dbGlobalSettings.SysMetricsSetting != reqForm.SysMetricsSetting {
		needUpdateDB = true
		if reqForm.SysMetricsEnable {
			err = scheduler.UpdateJob(&scheduler.JobConfig{
				Name:     "systemMetricsGet",
				Func:     scheduler.SystemMetricsGet,
				Args:     []any{reqForm.SysMetricsSetting},
				TimeType: scheduler.MinuteType,
				Interval: 1,
				DayAt:    "",
			})
			if err != nil {
				logger.Logger.Errorf("定时任务设置失败, err: %v, name: %v", err, "systemMetricsGet")
				c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "update fail"), "data": nil})
				return
			}
		} else {
			scheduler.DeleteJob("systemMetricsGet")
			db.SystemMetricsMutex.Lock()
			db.SystemMetrics = []db.SysMetrics{}
			db.SystemMetricsMutex.Unlock()
		}
	}

	if dbGlobalSettings.AutoUpdateEnable != reqForm.AutoUpdateEnable || dbGlobalSettings.AutoUpdateSetting != reqForm.AutoUpdateSetting || dbGlobalSettings.AutoUpdateRestart != reqForm.AutoUpdateRestart {
		needUpdateDB = true
		if reqForm.AutoUpdateEnable {
			err = scheduler.UpdateJob(&scheduler.JobConfig{
				Name:     "gameUpdate",
				Func:     scheduler.GameUpdate,
				Args:     []any{reqForm.AutoUpdateEnable, reqForm.AutoUpdateRestart},
				TimeType: scheduler.DayType,
				Interval: 0,
				DayAt:    reqForm.AutoUpdateSetting,
			})
			if err != nil {
				logger.Logger.Errorf("定时任务设置失败, err: %v, name: %v", err, "gameUpdate")
				c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "update fail"), "data": nil})
				return
			}
		} else {
			scheduler.DeleteJob("gameUpdate")
		}
	}

	if dbGlobalSettings.WebhookSetting != reqForm.WebhookSetting {
		var webhooks []webhook.GlobalWebhookItem
		if json.Unmarshal([]byte(reqForm.WebhookSetting), &webhooks) == nil {
			for _, w := range webhooks {
				if !utils.IsValidURL(w.URL) {
					logger.Logger.Warnf("非法请求已拦截, api: %s, username: %s", c.Request.URL.Path, c.GetString("username"))
					c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "invalid url"), "data": nil})
					return
				}
			}
			needUpdateDB = true
		} else {
			// 无法解析json，说明配置异常，丢弃所有数据，不写入数据库
			needUpdateDB = false
		}
	}

	if needUpdateDB {
		err = h.globalSettingDao.UpdateGlobalSetting(&reqForm)
		if err != nil {
			logger.Logger.Errorf("更新数据库失败, err: %v", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}

		webhook.Snd.Send(webhook.EventGlobalSettingUpdated, 0, map[string]interface{}{
			"dbUpdated": needUpdateDB,
		})

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "update success"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "bad request"), "data": nil})
}

func (h *Handler) screenRunningGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.RoomID == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	screens, err := game.RunningScreens()
	if err != nil {
		logger.Logger.Errorf("获取正在运行的screen失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "get screens fail"), "data": []string{}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": screens})
}

func screenKillPost(c *gin.Context) {
	type ReqForm struct {
		ScreenName string `json:"screenName"`
	}

	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	if reqForm.ScreenName == "" {
		logger.Logger.Infof("请求参数错误, api: %s", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	// 校验 ScreenName 只允许字母、数字、下划线和连字符，防止命令注入
	for _, ch := range reqForm.ScreenName {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-') {
			logger.Logger.Infof("ScreenName包含非法字符, api: %s", c.Request.URL.Path)
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
			return
		}
	}

	err := dst.KillRuntimeName(reqForm.ScreenName)
	if err != nil {
		logger.Logger.Warnf("关闭Screen失败: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "kill screen fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "kill screen success"), "data": nil})
}

func webhookTestPost(c *gin.Context) {
	var reqForm struct {
		URL    string `json:"url" binding:"required"`
		Secret string `json:"secret"`
	}
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	// webhook url 安全检测
	if !utils.IsValidURL(reqForm.URL) {
		logger.Logger.Warnf("非法请求已拦截, api: %s, username: %s", c.Request.URL.Path, c.GetString("username"))
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "invalid url"), "data": nil})
		return
	}

	err := webhook.Snd.SendTest(reqForm.URL, reqForm.Secret)
	if err != nil {
		logger.Logger.Warnf("webhook 测试失败, url: %s, err: %v", reqForm.URL, err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.GetF(c, "webhook test fail", err.Error()), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "webhook test success"), "data": nil})
}

func webhookEventsGet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": webhook.AllEventTypes})
}

func (h *Handler) pluginListGet(c *gin.Context) {
	type ReqForm struct {
		Partition
		Q string `json:"q" form:"q"`
	}
	var (
		reqForm ReqForm
		data    dao.PaginatedResult[models.Plugin]
	)
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	plugins, err := h.pluginDao.ListPlugins(reqForm.Q, reqForm.Page, reqForm.PageSize)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	data = *plugins

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}

func (h *Handler) pluginInstallPost(c *gin.Context) {
	var reqForm struct {
		Name  string `json:"name" binding:"required"`
		Proxy string `json:"proxy"`
	}
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.Name == models.PluginTmi {
		if !utils.IsValidURL(reqForm.Proxy) && reqForm.Proxy != "" {
			logger.Logger.Warnf("非法代理url已拦截, api: %s, proxy: %s", c.Request.URL.Path, reqForm.Proxy)
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
			return
		}
		plugin, err := h.pluginDao.GetPluginByPluginName(reqForm.Name)
		if err != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}

		if plugin.Step == 100 {
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "install success"), "data": nil})
			return
		}

		step := plugin.Step
		updateDb := func(plugin *models.Plugin) {
			err = h.pluginDao.UpdatePlugin(plugin)
			if err != nil {
				logger.Logger.Errorf("查询数据库失败, err: %v", err)
				c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
				return
			}
		}
		var images []models.DstImage

		step, images, err = initTmi(reqForm.Proxy, step)
		plugin.Step = step
		if err != nil {
			updateDb(plugin)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.GetF(c, "install fail", err.Error()), "data": nil})
			return
		}

		err = h.dstImageDao.InitImages(images)
		if err != nil {
			logger.Logger.Errorf("更新数据库失败, err: %v", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}

		plugin.Status = true
		updateDb(plugin)

		err = h.dstImageDao.DeleteNoName()
		if err != nil {
			logger.Logger.Warnf("清理异常图片失败: %v", err)
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "install success"), "data": nil})
		return
	}

	logger.Logger.Infof("请求参数错误, api: %s", c.Request.URL.Path)
	c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
	return
}

func (h *Handler) pluginActionPost(c *gin.Context) {
	var reqForm struct {
		Name string `json:"name" binding:"required"`
		Type string `json:"type"`
	}
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	plugin, err := h.pluginDao.GetPluginByPluginName(reqForm.Name)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	var pluginDir string

	switch reqForm.Type {
	case "enable":
		plugin.Status = true
	case "disable":
		plugin.Status = false
	case "update":
		if reqForm.Name == models.PluginTmi {
			var images []models.DstImage
			images, err = installTMIR("")
			if err != nil {
				logger.Logger.Errorf("更新插件失败: %v", err)
				c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "update fail"), "data": nil})
				return
			}
			err = h.dstImageDao.InitImages(images)
			if err != nil {
				logger.Logger.Errorf("更新数据库失败, err: %v", err)
				c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
				return
			}
			err = h.dstImageDao.DeleteNoName()
			if err != nil {
				logger.Logger.Warnf("清理异常图片失败: %v", err)
			}
			plugin.Status = true
		}
	case "uninstall":
		if reqForm.Name == models.PluginTmi {
			pluginDir = utils.PluginTmiPath
		}
		err = utils.RemoveDir(pluginDir)
		if err != nil {
			logger.Logger.Errorf("卸载插件失败: %v", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "update fail"), "data": nil})
			return
		}
		plugin.Step = 0
		plugin.Status = false
	default:
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	err = h.pluginDao.UpdatePlugin(plugin)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "update success"), "data": nil})
}

func (h *Handler) pluginStatusGet(c *gin.Context) {
	var reqForm struct {
		Name string `json:"name" form:"name" binding:"required"`
	}
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	plugin, err := h.pluginDao.GetPluginByPluginName(reqForm.Name)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": plugin.Status})
}
