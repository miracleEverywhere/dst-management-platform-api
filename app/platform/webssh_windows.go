//go:build windows

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
	"strings"

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

	cmd := exec.Command("powershell.exe", "-NoLogo", "-NoProfile", "-ExecutionPolicy", "Bypass")
	cmd.Dir = utils.WorkDir
	cmd.Env = append(os.Environ(), "DMP_HOME="+utils.WorkDir)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		logger.Logger.Errorf("创建PowerShell stdin失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "终端创建失败"})
		return
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Logger.Errorf("创建PowerShell stdout失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "终端创建失败"})
		return
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		logger.Logger.Errorf("创建PowerShell stderr失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "终端创建失败"})
		return
	}
	if err = cmd.Start(); err != nil {
		logger.Logger.Errorf("启动PowerShell失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "终端创建失败"})
		return
	}
	defer func() {
		_ = stdin.Close()
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}()

	m := melody.New()
	m.Config.MaxMessageSize = 1024 * 1024

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	readPipe := func(r io.Reader) {
		buf := make([]byte, 1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := r.Read(buf)
				if err != nil {
					if err != io.EOF {
						logger.Logger.Warnf("PowerShell读取错误: %v", err)
					}
					return
				}
				if n > 0 {
					data := make([]byte, n)
					copy(data, buf[:n])
					if err := m.BroadcastBinary(data); err != nil {
						logger.Logger.Warnf("广播数据失败: %v", err)
					}
				}
			}
		}
	}
	go readPipe(stdout)
	go readPipe(stderr)

	m.HandleMessage(func(s *melody.Session, msg []byte) {
		if len(msg) > 1024 {
			logger.Logger.Warnf("消息过大: %d", len(msg))
			return
		}
		if isResizeMessage(msg) {
			return
		}

		msg = translateWindowsInstallCommand(msg)
		if _, err := stdin.Write(msg); err != nil {
			logger.Logger.Warnf("PowerShell写入失败: %v", err)
		}
	})

	m.HandleClose(func(s *melody.Session, code int, reason string) error {
		logger.Logger.Infof("WebSocket连接关闭 --> code: %d, reason: %s", code, reason)
		cancel()
		_ = stdin.Close()
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

func isResizeMessage(msg []byte) bool {
	if len(msg) == 0 || msg[0] != '{' {
		return false
	}
	var resizeMsg struct {
		Type string `json:"type"`
	}
	return json.Unmarshal(msg, &resizeMsg) == nil && resizeMsg.Type == "resize"
}

func translateWindowsInstallCommand(msg []byte) []byte {
	normalized := strings.TrimSpace(string(msg))
	switch normalized {
	case "bash manual_install.sh":
		return []byte(".\\manual_install.ps1\r\n")
	case "bash manual_update.sh":
		return []byte(".\\manual_update.ps1\r\n")
	default:
		return msg
	}
}
