package models

type System struct {
	Key   string `gorm:"primaryKey;not null"`
	Value string `gorm:"not null"`
}

func (System) TableName() string {
	return "system"
}
