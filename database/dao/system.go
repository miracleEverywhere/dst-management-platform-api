package dao

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/utils"
	"gorm.io/gorm"
)

type SystemDAO struct {
	BaseDAO[models.System]
}

func NewSystemDAO(db *gorm.DB) *SystemDAO {
	dao := &SystemDAO{
		BaseDAO: *NewBaseDAO[models.System](db),
	}
	dao.initSystem()

	return dao
}

func (d *SystemDAO) GetSystem() (*models.System, error) {
	var system models.System
	err := d.db.First(&system).Error
	return &system, err
}

func (d *SystemDAO) initSystem() {
	dbSystem, err := d.GetSystem()
	if err != nil {
		var system models.System
		system.Dmp = "dmp"
		system.JwtSecret = utils.GenerateJWTSecret()
		db.JwtSecret = system.JwtSecret

		err = d.Create(&system)
		if err != nil {
			panic("数据库初始化失败: " + err.Error())
		}
		return
	}

	db.JwtSecret = dbSystem.JwtSecret
}
