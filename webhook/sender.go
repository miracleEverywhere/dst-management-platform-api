package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var Snd *Sender // 包级变量，server.go 中初始化

type Sender struct {
	globalSettingDao *dao.GlobalSettingDAO
	roomSettingDao   *dao.RoomSettingDAO
	roomDao          *dao.RoomDAO
	client           *http.Client
}

func NewSender(globalSettingDao *dao.GlobalSettingDAO, roomSettingDao *dao.RoomSettingDAO, roomDao *dao.RoomDAO) *Sender {
	return &Sender{
		globalSettingDao: globalSettingDao,
		roomSettingDao:   roomSettingDao,
		roomDao:          roomDao,
		client: &http.Client{
			Timeout: utils.HttpTimeout * time.Second,
		},
	}
}

// Send 异步发送 webhook 通知（fire-and-forget）
// roomID 为 0 表示全局事件（如游戏更新），此时只匹配全局 webhook 且不检查 roomIds 过滤
func (s *Sender) Send(eventType string, roomID int, data interface{}) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Logger.Errorf("webhook 发送 panic, event: %s, roomID: %d, panic: %v", eventType, roomID, r)
			}
		}()

		event, err := getEventInfoByType(eventType)
		if err != nil {
			logger.Logger.Warnf("未识别的Event: %s", eventType)
			event = EventInfo{Type: eventType}
		}

		payload := Payload{
			Event:     event,
			Timestamp: utils.GetTimestamp(),
			Data:      data,
		}

		// 收集所有匹配的 webhook（url + secret + name）
		type target struct {
			url    string
			secret string
			name   string
		}
		var targets []target

		// 1. 房间级 webhook（仅 roomID > 0 时匹配）
		if roomID > 0 {
			roomSetting, err := s.roomSettingDao.GetRoomSettingsByRoomID(roomID)
			if err == nil && roomSetting.WebhookSetting != "" {
				var items []WebhookItem
				if json.Unmarshal([]byte(roomSetting.WebhookSetting), &items) == nil {
					for _, item := range items {
						if item.Enabled && containsEvent(item.Events, eventType) {
							targets = append(targets, target{url: item.URL, secret: item.Secret, name: item.Name})
						}
					}
				}
			}
		}

		// 2. 全局级 webhook
		var globalSetting models.GlobalSetting
		if err := s.globalSettingDao.GetGlobalSetting(&globalSetting); err == nil && globalSetting.WebhookSetting != "" {
			var items []GlobalWebhookItem
			if json.Unmarshal([]byte(globalSetting.WebhookSetting), &items) == nil {
				for _, item := range items {
					if !item.Enabled || !containsEvent(item.Events, eventType) {
						continue
					}
					// roomID == 0 是全局事件，不检查房间过滤
					// roomID > 0 时，如果指定了 roomIds 则必须在列表中；空列表表示所有房间
					if roomID > 0 && len(item.RoomIDs) > 0 && !containsRoom(item.RoomIDs, roomID) {
						continue
					}
					targets = append(targets, target{url: item.URL, secret: item.Secret, name: item.Name})
				}
			}
		}

		if len(targets) == 0 {
			return
		}

		for _, t := range targets {
			payload.Name = t.name
			body, err := json.Marshal(payload)
			if err != nil {
				logger.Logger.Errorf("序列化 webhook payload 失败, event: %s, err: %v", event, err)
				continue
			}
			s.sendOne(t.url, t.secret, body)
		}
	}()
}

func (s *Sender) sendOne(url, secret string, body []byte) {
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		logger.Logger.Errorf("创建 webhook 请求失败, url: %s, err: %v", url, err)
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("DMP-Webhook/%s", utils.Version))

	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		req.Header.Set("X-DMP-Signature", hex.EncodeToString(mac.Sum(nil)))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		logger.Logger.Warnf("发送 webhook 失败, url: %s, err: %v", url, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		logger.Logger.Warnf("webhook 响应异常, url: %s, status: %d", url, resp.StatusCode)
	} else {
		logger.Logger.Debugf("webhook 发送成功, url: %s, status: %d", url, resp.StatusCode)
	}
}

// SendTest 同步发送测试 webhook（供测试按钮使用，返回错误）
func (s *Sender) SendTest(url, secret string) error {
	payload := Payload{
		Event: EventInfo{
			Type: "test",
			EN:   "Test",
			ZH:   "测试",
		},
		Timestamp: utils.GetTimestamp(),
		Data:      "这是一条测试消息 / This is a test message",
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", fmt.Sprintf("DMP-Webhook/%s", utils.Version))

	if secret != "" {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		req.Header.Set("X-DMP-Signature", hex.EncodeToString(mac.Sum(nil)))
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	logger.Logger.Debugf("webhook测试http响应码为：%d", resp.StatusCode)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return nil
}

// containsEvent 检查事件列表中是否包含指定事件
func containsEvent(events []string, eventType string) bool {
	for _, e := range events {
		if e == eventType {
			return true
		}
	}
	return false
}

// containsRoom 检查房间 ID 列表中是否包含指定房间
func containsRoom(roomIDs []int, roomID int) bool {
	for _, id := range roomIDs {
		if id == roomID {
			return true
		}
	}
	return false
}

func getEventInfoByType(eventType string) (EventInfo, error) {
	for _, e := range AllEventTypes {
		if e.Type == eventType {
			return e, nil
		}
	}

	return EventInfo{}, fmt.Errorf("event not found: %s", eventType)
}
