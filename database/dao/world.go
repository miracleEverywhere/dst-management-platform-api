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

func (d *WorldDAO) GetWorldsByRoomName(roomName string, page, pageSize int) (*PaginatedResult[models.World], error) {
	worlds, err := d.Query(page, pageSize, "room_name = ?", roomName)
	if err != nil {
		return nil, err
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return worlds, nil
	}

	return worlds, err
}
