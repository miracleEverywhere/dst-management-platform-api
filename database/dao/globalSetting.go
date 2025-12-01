package dao

import (
	"dst-management-platform-api/database/models"
	"gorm.io/gorm"
)

type GlobalSettingDAO struct {
	BaseDAO[models.GlobalSetting]
}

func NewGlobalSettingDAO(db *gorm.DB) *GlobalSettingDAO {
	dao := &GlobalSettingDAO{
		BaseDAO: *NewBaseDAO[models.GlobalSetting](db),
	}
	dao.initGlobalSetting()
	return dao
}

func (d *GlobalSettingDAO) GetGlobalSetting(setting *models.GlobalSetting) error {
	return d.db.First(setting).Error
}

func (d *GlobalSettingDAO) initGlobalSetting() {
	count, err := d.Count(nil)
	if err != nil {
		panic("数据库初始化失败: " + err.Error())
	}
	if count == 0 {
		globalSetting := models.GlobalSetting{
			PlayerGetFrequency: 60,
			// 其他默认值...
		}
		err = d.db.Create(&globalSetting).Error
		if err != nil {
			panic("数据库初始化失败: " + err.Error())
		}
	}
}
