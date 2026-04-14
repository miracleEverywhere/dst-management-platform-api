package dao

import (
	"dst-management-platform-api/database/models"

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
		args      []any
	)
	switch {
	case len(roomIDs) == 0 && gameName == "":
		// 无条件查询（返回所有记录）
		condition = "1 = 1" // 或者直接使用 ""，但部分数据库可能不支持空 WHERE

	case len(roomIDs) == 0 && gameName != "":
		// 仅模糊查询 gameName
		searchPattern := "%" + gameName + "%"
		condition = "game_name LIKE ?"
		args = []any{searchPattern}

	case len(roomIDs) != 0 && gameName == "":
		// 仅查询 name 在 roomNames 列表中的记录
		condition = "id IN (?)"
		args = []any{roomIDs}

	case len(roomIDs) != 0 && gameName != "":
		// 查询 name 在 roomNames 列表中，并且 name 或 display_name 匹配模糊搜索
		searchPattern := "%" + gameName + "%"
		condition = "id IN (?) AND (game_name LIKE ?)"
		args = []any{roomIDs, searchPattern}
	}

	rooms, err := d.Query(page, pageSize, condition, args...)
	return rooms, err
}

type RoomBasic struct {
	RoomName string `json:"roomName"`
	RoomID   int    `json:"roomID"`
	Status   bool   `json:"status"`
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
			Status:   room.Status,
		})
	}

	return &roomBasics, nil
}
