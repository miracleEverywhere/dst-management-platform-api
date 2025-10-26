package models

type System struct {
	Dmp        string `gorm:"primaryKey;not null"`
	JwtSecret  string `gorm:"not null"`
	InternetIp string
}

func (System) TableName() string {
	return "system"
}
