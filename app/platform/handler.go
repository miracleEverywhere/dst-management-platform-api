package platform

import (
	"context"
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/olahol/melody"
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
	systemMetricsLength := len(db.SystemMetrics)
	reqLength := 60
	var systemMetricsData []db.SysMetrics
	if systemMetricsLength > reqLength {
		systemMetricsData = db.SystemMetrics[systemMetricsLength-reqLength:]
	} else {
		systemMetricsData = db.SystemMetrics
	}
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

func websshWS(c *gin.Context) {
	// JWT 认证
	token := c.Query("token")
	tokenSecret := db.JwtSecret
	claims, err := utils.ValidateJWT(token, []byte(tokenSecret))
	if err != nil {
		logger.Logger.ErrorF("token认证失败: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}
	if claims.Role != "admin" {
		logger.Logger.ErrorF("越权请求: 用户角色为 %s", claims.Role)
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	// 创建PTY进程 - 使用login shell确保正确的环境
	cmd := exec.Command("bash", "-l")

	// 设置正确的环境变量
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"LANG=en_US.UTF-8",
		"LC_ALL=en_US.UTF-8",
	)

	f, err := pty.StartWithSize(cmd, &pty.Winsize{
		Rows: 30,
		Cols: 120,
	})
	if err != nil {
		logger.Logger.ErrorF("创建PTY失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "终端创建失败"})
		return
	}
	defer func() {
		if cmd.Process != nil {
			err = cmd.Process.Kill()
			if err != nil {
				logger.Logger.Error(err.Error())
			}
		}
	}()

	// 创建melody实例
	m := melody.New()
	m.Config.MaxMessageSize = 1024 * 1024

	// 使用context管理goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// PTY读取goroutine - 改进的数据读取
	go func() {
		buf := make([]byte, 1024) // 减小缓冲区大小
		for {
			select {
			case <-ctx.Done():
				return
			default:
				read, err := f.Read(buf)
				if err != nil {
					if err != io.EOF {
						logger.Logger.WarnF("PTY读取错误: %v", err)
					}
					return
				}

				// 直接发送原始数据
				if read > 0 {
					data := make([]byte, read)
					copy(data, buf[:read])

					// 使用BroadcastBinary确保二进制数据正确传输
					if err := m.BroadcastBinary(data); err != nil {
						logger.Logger.WarnF("广播数据失败: %v", err)
					}
				}
			}
		}
	}()

	// WebSocket消息处理
	m.HandleMessage(func(s *melody.Session, msg []byte) {
		// 限制消息大小
		if len(msg) > 1024 {
			logger.Logger.WarnF("消息过大: %d", len(msg))
			return
		}

		// 检查是否是调整终端大小的消息
		if len(msg) > 0 && msg[0] == '{' {
			var resizeMsg struct {
				Type string `json:"type"`
				Cols int    `json:"cols"`
				Rows int    `json:"rows"`
			}

			if err := json.Unmarshal(msg, &resizeMsg); err == nil && resizeMsg.Type == "resize" {
				// 调整PTY大小
				if err := pty.Setsize(f, &pty.Winsize{
					Rows: uint16(resizeMsg.Rows),
					Cols: uint16(resizeMsg.Cols),
				}); err != nil {
					logger.Logger.WarnF("调整终端大小失败: %v", err)
				}
				return
			}
		}

		// 处理普通输入数据
		_, err := f.Write(msg)
		if err != nil {
			logger.Logger.WarnF("PTY写入失败: %v", err)
			//s.CloseWithMessage([]byte("PTY写入失败"))
		}
	})

	// 连接关闭处理
	m.HandleClose(func(s *melody.Session, code int, reason string) error {
		logger.Logger.InfoF("WebSocket连接关闭 --> code: %d, reason: %s", code, reason)
		cancel()
		return nil
	})

	// 连接建立处理
	m.HandleConnect(func(s *melody.Session) {
		logger.Logger.InfoF("新的WebSSH连接建立, 用户: %s", claims.Username)
	})

	// 处理WebSocket升级
	err = m.HandleRequest(c.Writer, c.Request)
	if err != nil {
		logger.Logger.ErrorF("WebSocket升级失败: %v", err)
		return
	}

	// 等待命令结束
	err = cmd.Wait()
	if err != nil {
		logger.Logger.Error(err.Error())
	}

	logger.Logger.InfoF("WebSSH会话结束, 用户: %s", claims.Username)
}

func osInfoGet(c *gin.Context) {
	osInfo, err := getOSInfo()
	if err != nil {
		logger.Logger.Error("获取系统信息失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "get os info fail"), "data": osInfo})
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": osInfo})
}

func metricsGet(c *gin.Context) {
	type ReqForm struct {
		TimeRange int `json:"timeRange" form:"timeRange"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	systemMetricsLength := len(db.SystemMetrics)
	reqLength := reqForm.TimeRange * 60

	if systemMetricsLength > reqLength {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": db.SystemMetrics[systemMetricsLength-reqLength:]})
	} else {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": db.SystemMetrics})
	}
}

func (h *Handler) globalSettingsGet(c *gin.Context) {
	var globalSettings models.GlobalSetting

	err := h.globalSettingDao.GetGlobalSetting(&globalSettings)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": globalSettings})
}

func (h *Handler) globalSettingsPost(c *gin.Context) {
	var reqForm models.GlobalSetting
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	var dbGlobalSettings models.GlobalSetting

	err := h.globalSettingDao.GetGlobalSetting(&dbGlobalSettings)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	needUpdateDB := false

	if dbGlobalSettings.PlayerGetFrequency != reqForm.PlayerGetFrequency || dbGlobalSettings.UIDMaintainEnable != reqForm.UIDMaintainEnable {
		needUpdateDB = true
		err = scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     "onlinePlayerGet",
			Func:     scheduler.OnlinePlayerGet,
			Args:     []any{reqForm.PlayerGetFrequency, reqForm.UIDMaintainEnable},
			TimeType: scheduler.SecondType,
			Interval: reqForm.PlayerGetFrequency,
			DayAt:    "",
		})
		if err != nil {
			logger.Logger.Error("定时任务设置失败", "err", err, "name", "onlinePlayerGet")
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
				logger.Logger.Error("定时任务设置失败", "err", err, "name", "systemMetricsGet")
				c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "update fail"), "data": nil})
				return
			}
		} else {
			scheduler.DeleteJob("systemMetricsGet")
			db.SystemMetrics = []db.SysMetrics{}
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
				logger.Logger.Error("定时任务设置失败", "err", err, "name", "gameUpdate")
				c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "update fail"), "data": nil})
				return
			}
		} else {
			scheduler.DeleteJob("gameUpdate")
		}
	}

	if needUpdateDB {
		err = h.globalSettingDao.UpdateGlobalSetting(&reqForm)
		if err != nil {
			logger.Logger.Error("更新数据库失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "update success"), "data": nil})
}

func (h *Handler) screenRunningGet(c *gin.Context) {
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
	room, worlds, roomSetting, err := h.fetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("获取基本信息失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	screens, err := game.RunningScreens()
	if err != nil {
		logger.Logger.Error("获取正在运行的screen失败", "err", err)
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
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	if reqForm.ScreenName == "" {
		logger.Logger.Info("请求参数错误", "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	cmd := fmt.Sprintf("screen -X -S %s quit", reqForm.ScreenName)
	err := utils.BashCMD(cmd)
	if err != nil {
		logger.Logger.Warn("关闭Screen失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "kill screen fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "kill screen success"), "data": nil})
}
