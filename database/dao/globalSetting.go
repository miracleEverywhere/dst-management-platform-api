package dao

import (
	"dst-management-platform-api/database/models"
	"gorm.io/gorm"
)

type GlobalSettingDAO struct {
	BaseDAO[models.GlobalSetting]
}

func NewGlobalSettingDAO(db *gorm.DB) *GlobalSettingDAO {
	return &GlobalSettingDAO{
		BaseDAO: *NewBaseDAO[models.GlobalSetting](db),
	}
}
