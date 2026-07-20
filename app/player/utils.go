package player

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	userDao          *dao.UserDAO
	roomDao          *dao.RoomDAO
	worldDao         *dao.WorldDAO
	roomSettingDao   *dao.RoomSettingDAO
	uidMapDao        *dao.UidMapDAO
	globalSettingDao *dao.GlobalSettingDAO
}

func NewHandler(userDao *dao.UserDAO, roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO, uidMapDao *dao.UidMapDAO, globalSettingDao *dao.GlobalSettingDAO) *Handler {
	return &Handler{
		userDao:          userDao,
		roomDao:          roomDao,
		worldDao:         worldDao,
		roomSettingDao:   roomSettingDao,
		uidMapDao:        uidMapDao,
		globalSettingDao: globalSettingDao,
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

var client *http.Client = &http.Client{
	Timeout: utils.HttpTimeout * time.Second,
}

func getPublicBlockList() ([]string, error) {
	httpResponse, err := client.Get(utils.DstBlockList)
	if err != nil {
		return []string{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Logger.Errorf("请求关闭失败, err: %v", err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体
	// 检查 HTTP 状态码
	if httpResponse.StatusCode != http.StatusOK {
		return []string{}, fmt.Errorf("获取饥荒黑名单列表失败，HTTP Code: %d", httpResponse.StatusCode)
	}

	var responseData struct {
		Uids []string `json:"uids"`
	}
	if err = json.NewDecoder(httpResponse.Body).Decode(&responseData); err != nil {
		return []string{}, fmt.Errorf("解析饥荒黑名单失败, err: %v", err)
	}

	return responseData.Uids, nil
}
