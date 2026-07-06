package models

type Plugin struct {
	Name   string `gorm:"primaryKey;column:name" json:"name" binding:"required"`
	Status bool   `gorm:"column:status" json:"status"`
	Step   int    `gorm:"column:step" json:"step"`
}

func (Plugin) TableName() string {
	return "plugins"
}

var PluginTmi = "tmi"
