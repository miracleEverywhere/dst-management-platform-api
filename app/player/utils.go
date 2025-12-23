package player

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	userDao        *dao.UserDAO
	roomDao        *dao.RoomDAO
	worldDao       *dao.WorldDAO
	roomSettingDao *dao.RoomSettingDAO
	uidMapDao      *dao.UidMapDAO
}

func NewHandler(userDao *dao.UserDAO, roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO, uidMapDao *dao.UidMapDAO) *Handler {
	return &Handler{
		userDao:        userDao,
		roomDao:        roomDao,
		worldDao:       worldDao,
		roomSettingDao: roomSettingDao,
		uidMapDao:      uidMapDao,
	}
}

func (h *Handler) fetchGameInfo(roomID int) (*models.Room, *[]models.World, *models.RoomSetting, error) {
	room, err := h.roomDao.GetRoomByID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}
	worlds, err := h.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}
	roomSetting, err := h.roomSettingDao.GetRoomSettingsByRoomID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}

	return room, worlds, roomSetting, nil
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
