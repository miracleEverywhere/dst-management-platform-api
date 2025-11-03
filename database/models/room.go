package models

type Room struct {
	ID             int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Status         bool   `json:"status"`
	GameName       string `json:"gameName" binding:"required"`
	Description    string `json:"description"`
	GameMode       string `json:"gameMode" binding:"required"`
	CustomGameMode string `json:"customGameMode"`
	Pvp            bool   `json:"pvp"`
	MaxPlayer      int    `json:"maxPlayer" binding:"required"`
	MaxRollBack    int    `json:"maxRollBack" binding:"required"`
	ModInOne       bool   `json:"modInOne"`
	ModData        string `json:"modData"`
	Vote           bool   `json:"vote"`
	PauseEmpty     bool   `json:"pauseEmpty"`
	Password       string `json:"password"`
	Token          string `json:"token" binding:"required"`
	MasterIP       string `json:"masterIP" binding:"required"`
	MasterPort     int    `json:"masterPort" binding:"required"`
	ClusterKey     string `json:"clusterKey" binding:"required"`
}

func (Room) TableName() string {
	return "rooms"
}
