package models

type User struct {
	Username      string `gorm:"primaryKey;not null;column:username" json:"username" binding:"required"`
	Nickname      string `gorm:"not null;column:nickname" json:"nickname"`
	Role          string `gorm:"not null;column:role" json:"role"`
	Avatar        string `gorm:"not null;column:avatar" json:"avatar"`
	Password      string `gorm:"not null;column:password" json:"password"`
	Disabled      bool   `gorm:"not null;column:disabled" json:"disabled"`
	Rooms         string `gorm:"column:rooms" json:"rooms"`
	RoomCreation  bool   `gorm:"not null;column:room_creation" json:"roomCreation"`
	MaxWorlds     int    `gorm:"not null;column:max_worlds" json:"maxWorlds"`
	MaxPlayers    int    `gorm:"column:max_players" json:"maxPlayers"`
	CustomSetting string `gorm:"column:custom_setting" json:"customSetting"`
}

func (User) TableName() string {
	return "users"
}
