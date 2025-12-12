package models

type UidMap struct {
	UID      string `gorm:"primaryKey;not null;column:uid" json:"uid"`
	Nickname string `gorm:"not null;column:nickname" json:"nickname"`
	RoomID   int    `gorm:"not null;column:room_id" json:"room_id"`
}

func (UidMap) TableName() string {
	return "uid_map"
}
