package dao

import (
	"dst-management-platform-api/database/models"
	"errors"
	"gorm.io/gorm"
)

type RoomSettingDAO struct {
	BaseDAO[models.RoomSetting]
}

func NewRoomSettingDAO(db *gorm.DB) *RoomSettingDAO {
	return &RoomSettingDAO{
		BaseDAO: *NewBaseDAO[models.RoomSetting](db),
	}
}

func (d *RoomSettingDAO) GetRoomSettingsByRoomID(id int) (*[]models.RoomSetting, error) {
	var roomSettings *[]models.RoomSetting
	err := d.db.Where("room_id = ?", id).Find(roomSettings).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return roomSettings, nil
	}

	return roomSettings, nil
}
