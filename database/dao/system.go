package dao

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
	err := d.db.Where("key = ?", key).First(&system).Error
	return &system, err
}

func (d *SystemDAO) Set(key, value string) error {
	return d.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "key"}},
		DoUpdates: clause.AssignmentColumns([]string{"value"}),
	}).Create(&models.System{Key: key, Value: value}).Error
}

func (d *SystemDAO) initSystem() {
	logger.Logger.Debug("正在检查jwt秘钥")
	jwtSecret, err := d.Get(models.JwtSecret)
	if err != nil {
		logger.Logger.Debug("没有发现jwt秘钥，创建中")
		secret := utils.GenerateJWTSecret()
		err = d.Set(models.JwtSecret, secret)
		if err != nil {
			panic("数据库初始化失败: " + err.Error())
		}
		db.JwtSecret = secret
		logger.Logger.Debug("jwt秘钥创建完成")

		return
	}

	db.JwtSecret = jwtSecret.Value
	logger.Logger.Debug("jwt秘钥已写入缓存")
}
