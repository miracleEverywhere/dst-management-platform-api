package dashboard

import "dst-management-platform-api/database/dao"

type Handler struct {
	roomDao        *dao.RoomDAO
	worldDao       *dao.WorldDAO
	roomSettingDao *dao.RoomSettingDAO
}

func NewHandler(roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO) *Handler {
	return &Handler{
		roomDao:        roomDao,
		worldDao:       worldDao,
		roomSettingDao: roomSettingDao,
	}
}
