package dao

import (
	"dst-management-platform-api/database/models"

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

func (d *RoomSettingDAO) GetRoomSettingsByRoomID(id int) (*models.RoomSetting, error) {
	var roomSettings models.RoomSetting
	err := d.db.Where("room_id = ?", id).First(&roomSettings).Error
	return &roomSettings, err
}

func (d *RoomSettingDAO) UpdateRoomSetting(roomSetting *models.RoomSetting) error {
	err := d.db.Save(roomSetting).Error
	return err
}
