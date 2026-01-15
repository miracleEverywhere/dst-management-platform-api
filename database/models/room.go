package models

type Room struct {
	ID               int    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Status           bool   `gorm:"column:status" json:"status"`
	GameName         string `gorm:"column:game_name" json:"gameName" binding:"required"`
	Description      string `gorm:"column:description" json:"description"`
	GameMode         string `gorm:"column:game_mode" json:"gameMode" binding:"required"`
	CustomGameMode   string `gorm:"column:custom_game_mode" json:"customGameMode"`
	Pvp              bool   `gorm:"column:pvp" json:"pvp"`
	MaxPlayer        int    `gorm:"column:max_player" json:"maxPlayer" binding:"required"`
	MaxRollBack      int    `gorm:"column:max_roll_back" json:"maxRollBack" binding:"required"`
	ModInOne         bool   `gorm:"column:mod_in_one" json:"modInOne"`
	ModData          string `gorm:"column:mod_data" json:"modData"`
	Vote             bool   `gorm:"column:vote" json:"vote"`
	PauseEmpty       bool   `gorm:"column:pause_empty" json:"pauseEmpty"`
	Password         string `gorm:"column:password" json:"password"`
	Token            string `gorm:"column:token" json:"token" binding:"required"`
	MasterIP         string `gorm:"column:master_ip" json:"masterIP" binding:"required"`
	MasterPort       int    `gorm:"column:master_port" json:"masterPort" binding:"required"`
	ClusterKey       string `gorm:"column:cluster_key" json:"clusterKey" binding:"required"`
	Lan              bool   `gorm:"column:lan" json:"lan"`
	Offline          bool   `gorm:"column:offline" json:"offline"`
	SteamGroupOnly   bool   `gorm:"column:steam_group_only" json:"steamGroupOnly"`
	SteamGroupID     string `gorm:"column:steam_group_id" json:"steamGroupID"`
	SteamGroupAdmins bool   `gorm:"column:steam_group_admins" json:"steamGroupAdmins"`
}

func (Room) TableName() string {
	return "rooms"
}
