package models

type UidMap struct {
	UID      string `gorm:"primaryKey;not null" json:"uid"`
	Nickname string `gorm:"not null" json:"nickname"`
}

func (UidMap) TableName() string {
	return "uid_map"
}
