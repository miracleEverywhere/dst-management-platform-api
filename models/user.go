package models

type User struct {
	Username     string `gorm:"primaryKey;not null" json:"username"`
	Nickname     string `gorm:"not null" json:"nickname"`
	Role         string `gorm:"not null" json:"role"`
	Password     string `gorm:"not null" json:"password"`
	Disabled     bool   `gorm:"not null" json:"disabled"`
	Menus        string `json:"menus"`
	Rooms        string `json:"rooms"`
	RoomCreation bool   `gorm:"not null" json:"roomCreation"`
	MaxWorlds    int    `gorm:"not null" json:"maxWorlds"`
}

func (User) TableName() string {
	return "users"
}
