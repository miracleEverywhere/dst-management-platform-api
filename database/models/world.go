package models

type World struct {
	ID                 int    `gorm:"primaryKey;autoIncrement" json:"id"`
	RoomID             int    `gorm:"not null" json:"roomID"`
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
