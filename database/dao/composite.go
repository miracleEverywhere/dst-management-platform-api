package dao

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
)

// FetchGameInfo 传入房间ID，返回房间、世界、房间设置信息
func FetchGameInfo(roomID int) (*models.Room, *[]models.World, *models.RoomSetting, error) {
	roomDAO := NewRoomDAO(db.DB)
	worldDAO := NewWorldDAO(db.DB)
	roomSettingDAO := NewRoomSettingDAO(db.DB)

	room, err := roomDAO.GetRoomByID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}
	worlds, err := worldDAO.GetWorldsByRoomID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}
	roomSetting, err := roomSettingDAO.GetRoomSettingsByRoomID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}

	return room, worlds, roomSetting, nil
}
