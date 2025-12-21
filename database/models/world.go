package models

type World struct {
	ID                 int    `gorm:"primaryKey;autoIncrement;column:id" json:"id"` // 自增ID
	RoomID             int    `gorm:"not null;column:room_id" json:"roomID"`
	GameID             int    `gorm:"column:game_id" json:"gameID"` // 饥荒世界ID
	WorldName          string `gorm:"column:world_name" json:"worldName"`
	ServerPort         int    `gorm:"column:server_port" json:"serverPort"`
	MasterServerPort   int    `gorm:"column:master_server_port" json:"masterServerPort"`
	AuthenticationPort int    `gorm:"column:authentication_port" json:"authenticationPort"`
	IsMaster           bool   `gorm:"column:is_master" json:"isMaster"`
	EncodeUserPath     bool   `gorm:"column:encode_user_path" json:"encodeUserPath"`
	LevelData          string `gorm:"column:level_data" json:"levelData"`
	ModData            string `gorm:"column:mod_data" json:"modData"`
	LastAliveTime      string `gorm:"column:last_alive_time" json:"lastAliveTime"`
}

func (World) TableName() string {
	return "worlds"
}
