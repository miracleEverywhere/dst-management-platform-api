package dao

import (
	"dst-management-platform-api/models"
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
