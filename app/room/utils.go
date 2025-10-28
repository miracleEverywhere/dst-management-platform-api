package room

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
)

type Handler struct {
	roomDao  *dao.RoomDAO
	userDao  *dao.UserDAO
	worldDao *dao.WorldDAO
}

func NewRoomHandler(roomDao *dao.RoomDAO, userDao *dao.UserDAO, worldDao *dao.WorldDAO) *Handler {
	return &Handler{
		roomDao:  roomDao,
		userDao:  userDao,
		worldDao: worldDao,
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
