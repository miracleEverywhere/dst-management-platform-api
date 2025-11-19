package dao

import (
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"fmt"
	"gorm.io/gorm"
	"strconv"
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

func (d *RoomDAO) CreateRoom(room *models.Room) (*models.Room, error) {
	err := d.db.Create(room).Error
	return room, err
}

func (d *RoomDAO) UpdateRoom(room *models.Room) error {
	err := d.db.Save(room).Error
	return err
}

func (d *RoomDAO) GetRoomByID(id int) (*models.Room, error) {
	var room models.Room
	err := d.db.Where("id = ?", id).First(&room).Error
	return &room, err
}

func (d *RoomDAO) ListRooms(roomIDs []int, gameName string, page, pageSize int) (*PaginatedResult[models.Room], error) {
	var (
		condition string
		args      []interface{}
	)
	switch {
	case len(roomIDs) == 0 && gameName == "":
		// 无条件查询（返回所有记录）
		condition = "1 = 1" // 或者直接使用 ""，但部分数据库可能不支持空 WHERE

	case len(roomIDs) == 0 && gameName != "":
		// 仅模糊查询 gameName
		searchPattern := "%" + gameName + "%"
		condition = "game_name LIKE ?"
		args = []interface{}{searchPattern}

	case len(roomIDs) != 0 && gameName == "":
		// 仅查询 name 在 roomNames 列表中的记录
		condition = "id IN (?)"
		args = []interface{}{roomIDs}

	case len(roomIDs) != 0 && gameName != "":
		// 查询 name 在 roomNames 列表中，并且 name 或 display_name 匹配模糊搜索
		searchPattern := "%" + gameName + "%"
		condition = "id IN (?) AND (game_name LIKE ?)"
		args = []interface{}{roomIDs, searchPattern}
	}

	rooms, err := d.Query(page, pageSize, condition, args...)
	return rooms, err
}

func (d *RoomDAO) DeleteRoomByID(id int) error {
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
	if err := tx.Where("id = ?", id).Delete(&models.Room{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除worlds表中的数据
	if err := tx.Where("room_id = ?", id).Delete(&models.World{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 删除room_settings表中的数据
	if err := tx.Where("room_id = ?", id).Delete(&models.World{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	// 更新users表中的rooms权限
	if err := d.updateUserRooms(tx, id); err != nil {
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

func (d *RoomDAO) updateUserRooms(tx *gorm.DB, id int) error {
	// 查询所有包含该 roomID 的用户
	var users []models.User
	searchPattern := fmt.Sprintf("%%%d%%", id)

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
		var newRooms []int
		for _, room := range rooms {
			dbID, err := strconv.Atoi(strings.TrimSpace(room))
			if err != nil {
				return err
			}
			if dbID != id {
				newRooms = append(newRooms, id)
			}
		}

		// 重新组合 rooms 字符串
		var newRoomsIntSlice []string
		for _, i := range newRooms {
			newRoomsIntSlice = append(newRoomsIntSlice, strconv.Itoa(i))
		}
		newRoomsStr := strings.Join(newRoomsIntSlice, ",")

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

type RoomBasic struct {
	RoomName string `json:"roomName"`
	RoomID   int    `json:"roomID"`
}

func (d *RoomDAO) GetRoomBasic() (*[]RoomBasic, error) {
	var rooms []models.Room
	var roomBasics []RoomBasic
	err := d.db.Find(&rooms).Error
	if err != nil {
		return &roomBasics, err
	}
	for _, room := range rooms {
		roomBasics = append(roomBasics, RoomBasic{
			RoomName: room.GameName,
			RoomID:   room.ID,
		})
	}

	return &roomBasics, nil
}
