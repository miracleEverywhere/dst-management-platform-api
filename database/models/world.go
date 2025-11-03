package models

type World struct {
	ID                 int    `gorm:"primaryKey;autoIncrement" json:"id"` // 自增ID
	RoomID             int    `gorm:"not null" json:"roomID" `
	GameID             int    `json:"gameID"` // 饥荒世界ID
	ServerPort         int    `json:"serverPort"`
	MasterServerPort   int    `json:"masterServerPort"`
	AuthenticationPort int    `json:"authenticationPort"`
	IsMaster           bool   `json:"isMaster"`
	EncodeUserPath     bool   `json:"encodeUserPath"`
	LevelData          string `json:"levelData"`
	ModData            string `json:"modData"`
	LastAliveTime      string `json:"lastAliveTime"`
}

func (World) TableName() string {
	return "worlds"
}
