// Package tools 提供工具模块的业务实现。
//
// 本包不依赖 gin，也不直接写 HTTP 响应：业务结果统一用 service.Result 返回，
// 文案只给 i18n 词条 key，由 app/tools 的 handler 翻译并写响应。
package tools

import (
	"dst-management-platform-api/aichat"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// Service 工具模块业务实现，不依赖 gin
type Service struct {
	roomDao          *dao.RoomDAO
	userDao          *dao.UserDAO
	worldDao         *dao.WorldDAO
	roomSettingDao   *dao.RoomSettingDAO
	dstImageDao      *dao.DstImageDAO
	mapImageDao      *dao.MapImageDAO
	systemDao        *dao.SystemDAO
	roomAISettingDao *dao.RoomAISettingDAO
	aiManager        *aichat.Manager
}

// New 创建工具模块 service
func New(userDao *dao.UserDAO, roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO, dstImageDao *dao.DstImageDAO, mapImageDao *dao.MapImageDAO, systemDao *dao.SystemDAO, roomAISettingDao *dao.RoomAISettingDAO, aiManager *aichat.Manager) *Service {
	return &Service{
		roomDao:          roomDao,
		userDao:          userDao,
		worldDao:         worldDao,
		roomSettingDao:   roomSettingDao,
		dstImageDao:      dstImageDao,
		mapImageDao:      mapImageDao,
		systemDao:        systemDao,
		roomAISettingDao: roomAISettingDao,
		aiManager:        aiManager,
	}
}

// hasPermission 判断当前用户是否有房间权限
func (s *Service) hasPermission(actor service.Actor, roomID string) bool {
	// 管理员直接返回true
	if actor.IsAdmin() {
		return true
	}

	dbUser, err := s.userDao.GetUserByUsername(actor.Username)
	if err != nil {
		logger.Logger.Error("查询数据库失败")
		return false
	}
	roomIDs := strings.Split(dbUser.Rooms, ",")
	for _, id := range roomIDs {
		if id == roomID {
			return true
		}
	}

	return false
}

const gameImagesPath = constant.PluginTmiPath + "/dst_images"

func pngToBase64(prefab string) (string, error) {
	pngPath := filepath.Join(gameImagesPath, prefab+".png")
	data, err := os.ReadFile(pngPath)
	if err != nil {
		logger.Logger.Errorf("读取图片失败: %v", err)
		return "pic_not_found", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func pngsToBase64(prefabs []string, maxConcurrency int) map[string]string {
	if maxConcurrency <= 0 {
		maxConcurrency = 10
	}

	var mu sync.Mutex
	results := make(map[string]string, len(prefabs))
	sem := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	for _, p := range prefabs {
		wg.Add(1)
		sem <- struct{}{}
		go func(prefab string) {
			defer func() { <-sem }()
			defer wg.Done()

			b64, _ := pngToBase64(prefab)

			mu.Lock()
			results[prefab] = b64
			mu.Unlock()
		}(p)
	}

	wg.Wait()
	return results
}
