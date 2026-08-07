package dao

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/json"
	"errors"

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
	d.initJWTSecret()
}

func (d *SystemDAO) initJWTSecret() {
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

func (d *SystemDAO) GetAIBaseSetting() (*models.AIBaseSetting, error) {
	value, err := d.Get(models.AIBaseSettingKey)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		setting := models.DefaultAIBaseSetting()
		return &setting, nil
	}
	if err != nil {
		return nil, err
	}

	var setting models.AIBaseSetting
	if err = json.Unmarshal([]byte(value.Value), &setting); err != nil {
		return nil, err
	}
	return &setting, nil
}

func (d *SystemDAO) UpdateAIBaseSetting(setting *models.AIBaseSetting) error {
	data, err := json.Marshal(setting)
	if err != nil {
		return err
	}
	return d.Set(models.AIBaseSettingKey, string(data))
}
