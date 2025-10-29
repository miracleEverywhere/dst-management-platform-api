package dao

import (
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"errors"
	"gorm.io/gorm"
	"strings"
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

func (d *RoomDAO) DeleteRoomByName(name string) error {
	// 开始事务
	tx := d.db.Begin()
	if tx.Error != nil {
		return tx.Error
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			logger.Logger.Error("回滚事务失败", "panic", r)
		}
	}()

	// 删除rooms表中的数据
	if err := tx.Where("name = ?", name).Delete(&models.Room{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除worlds表中的数据
	if err := tx.Where("room_name = ?", name).Delete(&models.World{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除room_settings表中的数据
	if err := tx.Where("room_name = ?", name).Delete(&models.World{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 更新users表中的rooms权限
	if err := d.updateUserRooms(tx, name); err != nil {
		tx.Rollback()
		return err
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return err
	}

	return nil
}

// updateUserRooms 更新用户 rooms 字段，删除指定的 roomName
func (d *RoomDAO) updateUserRooms(tx *gorm.DB, roomName string) error {
	// 查询所有包含该 roomName 的用户
	var users []models.User
	searchPattern := "%" + roomName + "%"

	if err := tx.Where("rooms LIKE ?", searchPattern).Find(&users).Error; err != nil {
		return err
	}

	for _, user := range users {
		if user.Rooms == "" {
			continue
		}

		// 分割 rooms 字符串
		rooms := strings.Split(user.Rooms, ",")

		// 过滤掉要删除的 roomName
		var newRooms []string
		for _, room := range rooms {
			if strings.TrimSpace(room) != roomName {
				newRooms = append(newRooms, room)
			}
		}

		// 重新组合 rooms 字符串
		newRoomsStr := strings.Join(newRooms, ",")

		// 如果 rooms 字段为空，可以设置为空字符串或 NULL
		if newRoomsStr == "" {
			newRoomsStr = ""
		}

		// 更新用户记录
		if err := tx.Model(&models.User{}).
			Where("username = ?", user.Username).
			Update("rooms", newRoomsStr).Error; err != nil {
			return err
		}
	}

	return nil
}
