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
