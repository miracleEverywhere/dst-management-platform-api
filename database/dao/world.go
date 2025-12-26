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

func (d *WorldDAO) UpdateWorlds(worlds *[]models.World) error {
	if worlds == nil || len(*worlds) == 0 {
		return nil
	}

	return d.db.Transaction(func(tx *gorm.DB) error {
		if len(*worlds) == 0 {
			return nil
		}

		// 获取第一个world的room_id（所有world应该有相同的room_id）
		roomID := (*worlds)[0].RoomID
		if roomID == 0 {
			return errors.New("房间id异常")
		}

		// 1. 删除该room_id下的所有记录
		result := tx.Where("room_id = ?", roomID).Delete(&models.World{})
		if result.Error != nil {
			return result.Error
		}

		// 2. 批量插入新记录
		if err := tx.Create(worlds).Error; err != nil {
			return err
		}

		return nil
	})
}

func (d *WorldDAO) GetWorldsByRoomIDWthPage(id int) (*PaginatedResult[models.World], error) {
	// 获取所有的world，一个room最大world数为64
	worlds, err := d.Query(1, 64, "room_id = ?", id)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return worlds, nil
	}

	return worlds, err
}

func (d *WorldDAO) GetWorldsByRoomID(id int) (*[]models.World, error) {
	var worlds []models.World
	err := d.db.Where("room_id = ?", id).Find(&worlds).Error

	return &worlds, err
}
