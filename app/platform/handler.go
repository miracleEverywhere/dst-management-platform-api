package platform

import (
	"context"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/service"
	"dst-management-platform-api/webhook"
	"encoding/json"
	"io"
	nethttp "net/http"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/olahol/melody"
)

// Partition 分页请求参数
type Partition struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"pageSize" form:"pageSize"`
}

func (h *Handler) readmeGet(c *gin.Context) {
	message.Respond(c, h.svc.ReadmeGet())
}

func (h *Handler) readmePost(c *gin.Context) {
	message.Respond(c, h.svc.ReadmePost())
}

func (h *Handler) overviewGet(c *gin.Context) {
	message.Respond(c, h.svc.Overview())
}

func (h *Handler) gameVersionGet(c *gin.Context) {
	message.Respond(c, h.svc.GameVersion())
}

// websshWS 是 WebSocket 双向流，直接绑定 c.Writer / c.Request 做协议升级与 PTY 转发，
// 无法用 service.Result 表达，故传输层保留在 handler，PTY 之外不含业务规则。
func (h *Handler) websshWS(c *gin.Context) {
	// JWT 认证
	//token := c.Query("token")
	//tokenSecret := db.JwtSecret
	//claims, err := utils.ValidateJWT(token, []byte(tokenSecret))
	//if err != nil {
	//	logger.Logger.Errorf("token认证失败: %v", err)
	//	c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
	//	return
	//}
	//user, err := h.userDao.GetUserByUsername(claims.Username)
	//if err != nil || user.Username == "" || user.Disabled || user.Role != "admin" || user.TokenVersion != claims.TokenVersion {
	//	logger.Logger.Warnf("WebSSH用户状态无效, username: %s", claims.Username)
	//	c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
	//	return
	//}
	//if !db.ValidateTokenVersion(claims.Username, claims.TokenVersion) {
	//	logger.Logger.Warnf("WebSSH token已撤销, username: %s", claims.Username)
	//	c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
	//	return
	//}
	//if claims.Role != "admin" {
	//	logger.Logger.Errorf("越权请求: 用户角色为 %s", claims.Role)
	//	c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
	//	return
	//}

	webhook.Snd.Send(webhook.EventWebsocketConnected, 0, map[string]interface{}{
		"username": c.GetString("username"),
	})

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
		logger.Logger.Errorf("创建PTY失败: %v", err)
		c.JSON(nethttp.StatusInternalServerError, gin.H{"error": "终端创建失败"})
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
						logger.Logger.Warnf("PTY读取错误: %v", err)
					}
					return
				}

				// 直接发送原始数据
				if read > 0 {
					data := make([]byte, read)
					copy(data, buf[:read])

					// 使用BroadcastBinary确保二进制数据正确传输
					if err := m.BroadcastBinary(data); err != nil {
						logger.Logger.Warnf("广播数据失败: %v", err)
					}
				}
			}
		}
	}()

	// WebSocket消息处理
	m.HandleMessage(func(s *melody.Session, msg []byte) {
		// 限制消息大小
		if len(msg) > 1024 {
			logger.Logger.Warnf("消息过大: %d", len(msg))
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
					logger.Logger.Warnf("调整终端大小失败: %v", err)
				}
				return
			}
		}

		// 处理普通输入数据
		_, err := f.Write(msg)
		if err != nil {
			logger.Logger.Warnf("PTY写入失败: %v", err)
			//s.CloseWithMessage([]byte("PTY写入失败"))
		}
	})

	// 连接关闭处理
	m.HandleClose(func(s *melody.Session, code int, reason string) error {
		logger.Logger.Infof("WebSocket连接关闭 --> code: %d, reason: %s", code, reason)
		cancel()
		return nil
	})

	// 连接建立处理
	m.HandleConnect(func(s *melody.Session) {
		logger.Logger.Infof("新的WebSSH连接建立, 用户: %s", c.GetString("username"))
	})

	// 处理WebSocket升级
	err = m.HandleRequest(c.Writer, c.Request)
	if err != nil {
		logger.Logger.Errorf("WebSocket升级失败: %v", err)
		return
	}

	// 等待命令结束
	err = cmd.Wait()
	if err != nil {
		logger.Logger.Error(err.Error())
	}

	logger.Logger.Infof("WebSSH会话结束, 用户: %s", c.GetString("username"))
}

func (h *Handler) osInfoGet(c *gin.Context) {
	message.Respond(c, h.svc.OSInfoGet())
}

func (h *Handler) metricsGet(c *gin.Context) {
	type ReqForm struct {
		TimeRange int `json:"timeRange" form:"timeRange"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.Metrics(reqForm.TimeRange))
}

func (h *Handler) globalSettingsGet(c *gin.Context) {
	message.Respond(c, h.svc.GlobalSettingsGet())
}

func (h *Handler) globalSettingsPost(c *gin.Context) {
	var reqForm models.GlobalSetting
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.GlobalSettingsPost(middleware.Actor(c), reqForm))
}

func (h *Handler) screenRunningGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.ScreenRunning(middleware.Actor(c), reqForm.RoomID))
}

func (h *Handler) screenKillPost(c *gin.Context) {
	type ReqForm struct {
		ScreenName string `json:"screenName"`
	}

	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.KillScreen(middleware.Actor(c), reqForm.ScreenName))
}

func (h *Handler) webhookTestPost(c *gin.Context) {
	var reqForm struct {
		URL    string `json:"url" binding:"required"`
		Secret string `json:"secret"`
	}
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.WebhookTest(middleware.Actor(c), reqForm.URL, reqForm.Secret))
}

func (h *Handler) webhookEventsGet(c *gin.Context) {
	message.Respond(c, h.svc.WebhookEvents())
}

func (h *Handler) pluginListGet(c *gin.Context) {
	type ReqForm struct {
		Partition
		Q string `json:"q" form:"q"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.PluginList(reqForm.Page, reqForm.PageSize, reqForm.Q))
}

func (h *Handler) pluginInstallPost(c *gin.Context) {
	var reqForm struct {
		Name       string `json:"name" binding:"required"`
		Proxy      string `json:"proxy"`
		ImageParse *bool  `json:"imageParse"`
	}
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.PluginInstall(middleware.Actor(c), reqForm.Name, reqForm.Proxy, reqForm.ImageParse))
}

func (h *Handler) pluginActionPost(c *gin.Context) {
	var reqForm struct {
		Name string `json:"name" binding:"required"`
		Type string `json:"type"`
	}
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.PluginAction(middleware.Actor(c), reqForm.Name, reqForm.Type))
}

func (h *Handler) pluginStatusGet(c *gin.Context) {
	var reqForm struct {
		Name string `json:"name" form:"name" binding:"required"`
	}
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.PluginStatus(reqForm.Name))
}
