//go:build !windows

package platform

import (
	"context"
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"os/exec"

	"github.com/creack/pty"
	"github.com/gin-gonic/gin"
	"github.com/olahol/melody"
)

func websshWS(c *gin.Context) {
	token := c.Query("token")
	claims, err := utils.ValidateJWT(token, []byte(db.JwtSecret))
	if err != nil {
		logger.Logger.Errorf("token认证失败: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "认证失败"})
		return
	}
	if claims.Role != "admin" {
		logger.Logger.Errorf("越权请求: 用户角色为 %s", claims.Role)
		c.JSON(http.StatusForbidden, gin.H{"error": "权限不足"})
		return
	}

	webhook.Snd.Send(webhook.EventWebsocketConnected, 0, map[string]interface{}{
		"username": claims.Username,
	})

	cmd := exec.Command("bash", "-l")
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "终端创建失败"})
		return
	}
	defer func() {
		if cmd.Process != nil {
			if err = cmd.Process.Kill(); err != nil {
				logger.Logger.Error(err.Error())
			}
		}
	}()

	m := melody.New()
	m.Config.MaxMessageSize = 1024 * 1024

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		buf := make([]byte, 1024)
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
				if read > 0 {
					data := make([]byte, read)
					copy(data, buf[:read])
					if err := m.BroadcastBinary(data); err != nil {
						logger.Logger.Warnf("广播数据失败: %v", err)
					}
				}
			}
		}
	}()

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		if len(msg) > 1024 {
			logger.Logger.Warnf("消息过大: %d", len(msg))
			return
		}

		if len(msg) > 0 && msg[0] == '{' {
			var resizeMsg struct {
				Type string `json:"type"`
				Cols int    `json:"cols"`
				Rows int    `json:"rows"`
			}

			if err := json.Unmarshal(msg, &resizeMsg); err == nil && resizeMsg.Type == "resize" {
				if err := pty.Setsize(f, &pty.Winsize{
					Rows: uint16(resizeMsg.Rows),
					Cols: uint16(resizeMsg.Cols),
				}); err != nil {
					logger.Logger.Warnf("调整终端大小失败: %v", err)
				}
				return
			}
		}

		if _, err := f.Write(msg); err != nil {
			logger.Logger.Warnf("PTY写入失败: %v", err)
		}
	})

	m.HandleClose(func(s *melody.Session, code int, reason string) error {
		logger.Logger.Infof("WebSocket连接关闭 --> code: %d, reason: %s", code, reason)
		cancel()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return nil
	})

	m.HandleConnect(func(s *melody.Session) {
		logger.Logger.Infof("新的WebSSH连接建立, 用户: %s", claims.Username)
	})

	if err = m.HandleRequest(c.Writer, c.Request); err != nil {
		logger.Logger.Errorf("WebSocket升级失败: %v", err)
		return
	}

	if err = cmd.Wait(); err != nil {
		logger.Logger.Error(err.Error())
	}

	logger.Logger.Infof("WebSSH会话结束, 用户: %s", claims.Username)
}
