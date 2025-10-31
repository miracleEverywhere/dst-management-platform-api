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

func (d *WorldDAO) GetWorldsByRoomID(id int) (*PaginatedResult[models.World], error) {
	// 获取所有的world，一个room最大world数为64
	worlds, err := d.Query(1, 64, "room_id = ?", id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return worlds, nil
	}

	return worlds, err
}
