package dao

import (
	"dst-management-platform-api/database/models"
	"errors"
	"gorm.io/gorm"
)

type RoomDAO struct {
	BaseDAO[models.Room]
}

func NewRoomDAO(db *gorm.DB) *RoomDAO {
	return &RoomDAO{
		BaseDAO: *NewBaseDAO[models.Room](db),
	}
}

func (d *RoomDAO) GetRoomByName(name string) (*models.Room, error) {
	var room models.Room
	err := d.db.Where("name = ?", name).First(&room).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &room, nil
	}
	return &room, err
}

func (d *RoomDAO) ListRooms(roomName string, page, pageSize int) (*PaginatedResult[models.Room], error) {
	searchPattern := "%" + roomName + "%"
	rooms, err := d.Query(page, pageSize, "name LIKE ? OR display_name LIKE ?", searchPattern, searchPattern)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return rooms, nil
	}
	return rooms, err
}
