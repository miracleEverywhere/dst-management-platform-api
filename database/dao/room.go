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

func (d *RoomDAO) ListRooms(roomNames []string, roomName string, page, pageSize int) (*PaginatedResult[models.Room], error) {
	var (
		condition string
		args      []interface{}
	)
	switch {
	case len(roomNames) == 0 && roomName == "":
		// 无条件查询（返回所有记录）
		condition = "1 = 1" // 或者直接使用 ""，但部分数据库可能不支持空 WHERE

	case len(roomNames) == 0 && roomName != "":
		// 仅模糊查询 name 或 display_name
		searchPattern := "%" + roomName + "%"
		condition = "name LIKE ? OR display_name LIKE ?"
		args = []interface{}{searchPattern, searchPattern}

	case len(roomNames) != 0 && roomName == "":
		// 仅查询 name 在 roomNames 列表中的记录
		condition = "name IN (?)"
		args = []interface{}{roomNames}

	case len(roomNames) != 0 && roomName != "":
		// 查询 name 在 roomNames 列表中，并且 name 或 display_name 匹配模糊搜索
		searchPattern := "%" + roomName + "%"
		condition = "name IN (?) AND (name LIKE ? OR display_name LIKE ?)"
		args = []interface{}{roomNames, searchPattern, searchPattern}
	}

	rooms, err := d.Query(page, pageSize, condition, args...)
	return rooms, err
}
