package dao

import (
	"dst-management-platform-api/database/models"
	"fmt"

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

func (d *UidMapDAO) GetUidMapByRoomID(roomID int) (*[]models.UidMap, error) {
	var uidMaps []models.UidMap
	err := d.db.Where("room_id = ?", roomID).Find(&uidMaps).Error

	return &uidMaps, err
}

func (d *UidMapDAO) UpdateUidMap(uidMap *models.UidMap) error {
	if uidMap.UID == "" || uidMap.Nickname == "" || uidMap.RoomID == 0 {
		return fmt.Errorf("三个字段不能为空")
	}

	return d.db.Save(uidMap).Error
}

func (d *UidMapDAO) DeleteUidMapByRoomID(roomID int) error {
	return d.db.Where("room_id = ?", roomID).Delete(&models.UidMap{}).Error
}
