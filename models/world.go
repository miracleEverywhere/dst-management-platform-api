package models

type World struct {
	FingerPrint        string `gorm:"primaryKey;not null" json:"fingerPrint"`
	ID                 int    `json:"id"`
	Name               string `json:"name"`
	ServerPort         int    `json:"serverPort"`
	MasterServerPort   int    `json:"masterServerPort"`
	AuthenticationPort int    `json:"authenticationPort"`
	IsMaster           bool   `json:"isMaster"`
	EncodeUserPath     bool   `json:"encodeUserPath"`
	LevelData          string `json:"levelData"`
	ModData            string `json:"modData"`
	LastAliveTime      string `json:"lastAliveTime"`
}
