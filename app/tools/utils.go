package tools

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	roomDao        *dao.RoomDAO
	userDao        *dao.UserDAO
	worldDao       *dao.WorldDAO
	roomSettingDao *dao.RoomSettingDAO
	dstImageDao    *dao.DstImageDAO
}

func NewHandler(userDao *dao.UserDAO, roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO, dstImageDao *dao.DstImageDAO) *Handler {
	return &Handler{
		roomDao:        roomDao,
		userDao:        userDao,
		worldDao:       worldDao,
		roomSettingDao: roomSettingDao,
		dstImageDao:    dstImageDao,
	}
}

func (h *Handler) hasPermission(c *gin.Context, roomID string) bool {
	role, _ := c.Get("role")
	username, _ := c.Get("username")

	// 管理员直接返回true
	if role.(string) == "admin" {
		return true
	} else {
		dbUser, err := h.userDao.GetUserByUsername(username.(string))
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
	}

	return false
}

const gameImagesPath = utils.PluginTmiPath + "/dst_images"

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
