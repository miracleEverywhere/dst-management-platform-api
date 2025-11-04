package external

import "dst-management-platform-api/database/dao"

type Handler struct {
	roomDao        *dao.RoomDAO
	userDao        *dao.UserDAO
	worldDao       *dao.WorldDAO
	roomSettingDao *dao.RoomSettingDAO
}

func NewExternalHandler(roomDao *dao.RoomDAO, userDao *dao.UserDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO) *Handler {
	return &Handler{
		roomDao:        roomDao,
		userDao:        userDao,
		worldDao:       worldDao,
		roomSettingDao: roomSettingDao,
	}
}
