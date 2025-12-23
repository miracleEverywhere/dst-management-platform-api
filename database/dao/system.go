package dao

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
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

func (d *SystemDAO) Get(key string) (*models.System, error) {
	var system models.System
	err := d.db.First(&system).Error
	return &system, err
}

func (d *SystemDAO) Set(systems []models.System) error {
	err := d.db.Save(&systems).Error
	return err
}

func (d *SystemDAO) initSystem() {
	logger.Logger.Debug("正在检查jwt秘钥")
	jwtSecret, err := d.Get("jwt_secret")
	if err != nil {
		logger.Logger.Debug("没有发现jwt秘钥，创建中")
		secret := utils.GenerateJWTSecret()
		system := []models.System{
			{Key: "jwt_secret", Value: secret},
		}
		err = d.Set(system)
		if err != nil {
			panic("数据库初始化失败: " + err.Error())
		}
		logger.Logger.Debug("jwt秘钥创建完成")
		return
	}

	db.JwtSecret = jwtSecret.Value
	logger.Logger.Debug("jwt秘钥已写入缓存")
}
