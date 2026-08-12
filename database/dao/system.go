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
	d.initReadme()
	d.initAIBaseSetting()
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

func (d *SystemDAO) initReadme() {
	logger.Logger.Debug("正在检查README阅读状态")
	_, err := d.Get(models.ReadmeKey)
	if err == nil {
		logger.Logger.Debug("README阅读状态已存在")
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		panic("数据库初始化失败: " + err.Error())
	}

	if err = d.Set(models.ReadmeKey, "false"); err != nil {
		panic("数据库初始化失败: " + err.Error())
	}
	logger.Logger.Debug("README阅读状态创建完成")
}

func (d *SystemDAO) initAIBaseSetting() {
	logger.Logger.Debug("正在检查AI基础配置")
	_, err := d.Get(models.AIBaseSettingKey)
	if err == nil {
		logger.Logger.Debug("AI基础配置已存在")
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		panic("数据库初始化失败: " + err.Error())
	}

	setting := models.DefaultAIBaseSetting()
	data, err := json.Marshal(&setting)
	if err != nil {
		panic("AI基础配置初始化失败: " + err.Error())
	}
	if err = d.Set(models.AIBaseSettingKey, string(data)); err != nil {
		panic("数据库初始化失败: " + err.Error())
	}
	logger.Logger.Debug("AI基础配置创建完成")
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
