package dao

import (
	"dst-management-platform-api/database/models"
	"errors"
	"gorm.io/gorm"
)

type WorldDAO struct {
	BaseDAO[models.World]
}

func NewWorldDAO(db *gorm.DB) *WorldDAO {
	return &WorldDAO{
		BaseDAO: *NewBaseDAO[models.World](db),
	}
}

func (d *WorldDAO) GetWorldsByFingerPrints(fingerPrints []string) ([]models.World, error) {
	worlds, err := d.Query("world_finger_print IN ?", fingerPrints)
	if err != nil {
		return nil, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return worlds, nil
	}

	return worlds, err
}
