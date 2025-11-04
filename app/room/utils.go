package room

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	roomDao        *dao.RoomDAO
	userDao        *dao.UserDAO
	worldDao       *dao.WorldDAO
	roomSettingDao *dao.RoomSettingDAO
}

func NewRoomHandler(roomDao *dao.RoomDAO, userDao *dao.UserDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO) *Handler {
	return &Handler{
		roomDao:        roomDao,
		userDao:        userDao,
		worldDao:       worldDao,
		roomSettingDao: roomSettingDao,
	}
}

type Partition struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"pageSize" form:"pageSize"`
}

type XRoomWorld struct {
	models.Room
	Worlds []models.World `json:"worlds"`
}

type XRoomTotalInfo struct {
	RoomData        models.Room        `json:"roomData"`
	WorldData       []models.World     `json:"worldData"`
	RoomSettingData models.RoomSetting `json:"roomSettingData"`
}

func (h *Handler) hasPermission(c *gin.Context) (bool, error) {
	role, _ := c.Get("role")
	username, _ := c.Get("username")
	var (
		has    bool
		err    error
		dbUser *models.User
	)

	// 管理员直接返回true
	if role.(string) == "admin" {
		has = true
	} else {
		dbUser, err = h.userDao.GetUserByUsername(username.(string))
		if err != nil {
			return has, err
		}
		if dbUser.RoomCreation {
			has = true
		}
	}

	return has, err
}
