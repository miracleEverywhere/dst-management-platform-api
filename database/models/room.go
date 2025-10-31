package models

type Room struct {
	ID             int    `gorm:"primaryKey;autoIncrement" json:"id"`
	Status         bool   `json:"status"`
	GameName       string `json:"gameName"`
	Description    string `json:"description"`
	GameMode       string `json:"gameMode"`
	CustomGameMode string `json:"customGameMode"`
	Pvp            bool   `json:"pvp"`
	MaxPlayer      int    `json:"maxPlayer"`
	MaxRollBack    int    `json:"maxRollBack"`
	ModInOne       bool   `json:"modInOne"`
	ModData        string `json:"modData"`
	Vote           bool   `json:"vote"`
	PauseEmpty     bool   `json:"pauseEmpty"`
	Password       string `json:"password"`
	Token          string `json:"token"`
	MasterIP       string `json:"masterIP"`
	MasterPort     int    `json:"masterPort"`
	ClusterKey     string `json:"clusterKey"`
}

func (Room) TableName() string {
	return "rooms"
}
