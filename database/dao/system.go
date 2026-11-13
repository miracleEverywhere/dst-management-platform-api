package dao

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/base64"
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
	d.initCloudKey()
	d.initReadme()
	d.initAIBaseSetting()
}

func (d *SystemDAO) initJWTSecret() {
	logger.Logger.Debug("正在检查jwt秘钥")
	jwtSecret, err := d.Get(models.JwtSecret)
	if err != nil {
		logger.Logger.Debug("没有发现jwt秘钥，创建中")
		secret, genErr := utils.GenerateJWTSecret()
		if genErr != nil {
			panic("生成jwt秘钥失败: " + genErr.Error())
		}
		err = d.Set(models.JwtSecret, secret)
		if err != nil {
			panic("数据库初始化失败: " + err.Error())
		}
		cache.JwtSecret = secret
		logger.Logger.Debug("jwt秘钥创建完成")
		return
	}

	cache.JwtSecret = jwtSecret.Value
	logger.Logger.Debug("jwt秘钥已写入缓存")
}

// initCloudKey 确保 system 表中存在云凭据主密钥，并载入缓存。
// 密钥不再写入 dmp_files/cloud.key，随数据库一起备份与迁移。
func (d *SystemDAO) initCloudKey() {
	logger.Logger.Debug("正在检查云凭据密钥")
	row, err := d.Get(models.CloudKey)
	if err == nil {
		key, decodeErr := base64.RawURLEncoding.DecodeString(row.Value)
		if decodeErr == nil && len(key) == constant.CloudKeyBytes {
			cache.CloudKey = key
			logger.Logger.Debug("云凭据密钥已写入缓存")
			return
		}
		logger.Logger.Warn("云凭据密钥格式不正确，重新生成")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		panic("数据库初始化失败: " + err.Error())
	}

	if err = d.createCloudKey(); err != nil {
		panic("数据库初始化失败: " + err.Error())
	}
}

func (d *SystemDAO) createCloudKey() error {
	secret, err := utils.GenerateRandomKey(constant.CloudKeyBytes)
	if err != nil {
		return err
	}
	key, err := base64.RawURLEncoding.DecodeString(secret)
	if err != nil {
		return err
	}
	if err = d.Set(models.CloudKey, secret); err != nil {
		return err
	}
	cache.CloudKey = key
	logger.Logger.Debug("云凭据密钥创建完成")

	return nil
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
