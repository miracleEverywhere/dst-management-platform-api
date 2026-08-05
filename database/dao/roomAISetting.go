package dao

import (
	"dst-management-platform-api/database/models"
	"errors"

	"gorm.io/gorm"
)

type RoomAISettingDAO struct {
	BaseDAO[models.RoomAISetting]
}

func NewRoomAISettingDAO(db *gorm.DB) *RoomAISettingDAO {
	return &RoomAISettingDAO{
		BaseDAO: *NewBaseDAO[models.RoomAISetting](db),
	}
}

func (d *RoomAISettingDAO) GetByRoomID(roomID int) (*models.RoomAISetting, error) {
	var setting models.RoomAISetting
	err := d.db.Where("room_id = ?", roomID).First(&setting).Error
	return &setting, err
}

func (d *RoomAISettingDAO) UpdateSetting(setting *models.RoomAISetting) error {
	return d.db.Save(setting).Error
}

func (d *RoomAISettingDAO) ListEnabled() ([]models.RoomAISetting, error) {
	var settings []models.RoomAISetting
	err := d.db.Where("enabled = ?", true).Find(&settings).Error
	return settings, err
}

func (d *RoomAISettingDAO) DeleteByRoomID(roomID int) error {
	_, err := d.GetByRoomID(roomID)
	if err == nil {
		return d.db.Where("room_id = ?", roomID).Delete(&models.RoomAISetting{}).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return d.db.Where("room_id = ?", roomID).Delete(&models.RoomAISetting{}).Error
	}

	return err
}
