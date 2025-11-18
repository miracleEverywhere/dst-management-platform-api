package platform

import (
	"context"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/json"
	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/olahol/melody"
	"io"
	"net/http"
	"os"
	"os/exec"
	"time"
)

func (h *Handler) overviewGet(c *gin.Context) {
	type Data struct {
		RunningTime int64  `json:"runningTime"`
		Memory      uint64 `json:"memory"`
		RoomCount   int64  `json:"roomCount"`
		WorldCount  int64  `json:"worldCount"`
		UserCount   int64  `json:"userCount"`
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
	// TODO 1小时cpu、内存、网络上行、网络下行最大值
	// TODO 玩家数最多的的房间Top3

	data := Data{
		RunningTime: int64(t),
		Memory:      mem,
		RoomCount:   roomCount,
		WorldCount:  worldCount,
		UserCount:   userCount,
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}

func gameVersionGet(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": GetDSTVersion()})
}

func websshWS(c *gin.Context) {
	// JWT 认证
	token := c.Query("token")
	tokenSecret := db.JwtSecret
	claims, err := utils.ValidateJWT(token, []byte(tokenSecret))
	if err != nil {
		logger.Logger.Error("token认证失败: ", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}
	if claims.Role != "admin" {
		logger.Logger.Error("越权请求: 用户角色为 ", claims.Role)
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
		logger.Logger.Error("创建PTY失败: ", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "终端创建失败"})
		return
	}
	defer func() {
		if cmd.Process != nil {
			cmd.Process.Kill()
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
						logger.Logger.Warn("PTY读取错误: ", err)
					}
					return
				}

				// 直接发送原始数据
				if read > 0 {
					data := make([]byte, read)
					copy(data, buf[:read])

					// 使用BroadcastBinary确保二进制数据正确传输
					if err := m.BroadcastBinary(data); err != nil {
						logger.Logger.Warn("广播数据失败: ", err)
					}
				}
			}
		}
	}()

	// WebSocket消息处理
	m.HandleMessage(func(s *melody.Session, msg []byte) {
		// 限制消息大小
		if len(msg) > 1024 {
			logger.Logger.Warn("消息过大: ", len(msg))
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
					logger.Logger.Warn("调整终端大小失败: ", err)
				}
				return
			}
		}

		// 处理普通输入数据
		_, err := f.Write(msg)
		if err != nil {
			logger.Logger.Warn("PTY写入失败: ", err)
			//s.CloseWithMessage([]byte("PTY写入失败"))
		}
	})

	// 连接关闭处理
	m.HandleClose(func(s *melody.Session, code int, reason string) error {
		logger.Logger.Info("WebSocket连接关闭: ", code, reason)
		cancel()
		return nil
	})

	// 连接建立处理
	m.HandleConnect(func(s *melody.Session) {
		logger.Logger.Info("新的WebSSH连接建立, 用户: ", claims.Username)
	})

	// 处理WebSocket升级
	err = m.HandleRequest(c.Writer, c.Request)
	if err != nil {
		logger.Logger.Error("WebSocket升级失败: ", err)
		return
	}

	// 等待命令结束
	cmd.Wait()
	logger.Logger.Info("WebSSH会话结束, 用户: ", claims.Username)
}

func osInfoGet(c *gin.Context) {
	osInfo, err := getOSInfo()
	if err != nil {
		logger.Logger.Error("获取系统信息失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "get os info fail"), "data": osInfo})
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": osInfo})
}

func (h *Handler) userListGet(c *gin.Context) {
	type ReqForm struct {
		Partition
		Username string `json:"username" form:"username"`
	}
	var (
		reqForm ReqForm
		data    dao.PaginatedResult[models.User]
	)
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": data})
		return
	}

	role, _ := c.Get("role")
	if role.(string) != "admin" {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": data})
		return
	}

	users, err := h.userDao.ListUsers(reqForm.Username, reqForm.Page, reqForm.PageSize)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": data})
		return
	}

	for _, user := range users.Data {
		user.Password = ""
		data.Data = append(data.Data, user)
	}
	data.Page = users.Page
	data.PageSize = users.PageSize
	data.TotalCount = users.TotalCount

	if data.TotalCount == 0 {
		data.Data = []models.User{}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}
