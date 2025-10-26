package models

type System struct {
	ID    uint   `gorm:"primaryKey;autoIncrement"`
	Key   string `gorm:"not null"`
	Value string `gorm:"not null"`
}

func (System) TableName() string {
	return "system"
}
