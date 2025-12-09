package dao

import (
	"dst-management-platform-api/database/models"
	"gorm.io/gorm"
)

type UidMapDAO struct {
	BaseDAO[models.UidMap]
}

func NewUidMapDAO(db *gorm.DB) *UidMapDAO {
	return &UidMapDAO{
		BaseDAO: *NewBaseDAO[models.UidMap](db),
	}
}

func (d *UserDAO) GetUidMapByRoomID(roomID int) (*[]models.UidMap, error) {
	var uidMaps []models.UidMap
	err := d.db.Where("room_id = ?", roomID).Find(&uidMaps).Error

	return &uidMaps, err
}
