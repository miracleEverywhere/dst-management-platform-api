package dao

import (
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

type PluginDAO struct {
	BaseDAO[models.Plugin]
}

func NewPluginDAO(db *gorm.DB) *PluginDAO {
	dao := &PluginDAO{
		BaseDAO: *NewBaseDAO[models.Plugin](db),
	}
	dao.initPlugin()

	return dao
}

func (d *PluginDAO) GetPluginByPluginName(name string) (*models.Plugin, error) {
	var plugin models.Plugin
	err := d.db.Where("name = ?", name).First(&plugin).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &plugin, nil
	}
	return &plugin, err
}

func (d *PluginDAO) ListPlugins(q string, page, pageSize int) (*PaginatedResult[models.Plugin], error) {
	var (
		condition string
		args      []any
	)
	if q != "" {
		searchName := "%" + q + "%"
		condition = "name LIKE ?"
		args = []any{searchName}
	}

	plugins, err := d.Query(page, pageSize, condition, args...)

	return plugins, err
}

func (d *PluginDAO) UpdatePlugin(plugin *models.Plugin) error {
	err := d.db.Save(plugin).Error

	return err
}

func (d *PluginDAO) initPlugin() {
	logger.Logger.Debug("正在检查插件配置")
	count, err := d.Count(nil)
	if err != nil {
		logger.Logger.Errorf("数据库检查失败: %v", err)
		panic(fmt.Errorf("数据库检查失败: %v", err))
	}
	if count == 0 {
		logger.Logger.Debug("正在初始化插件配置")
		plugin := models.Plugin{
			Name:   models.PluginTmi,
			Status: false,
			Step:   0,
		}
		err = d.Create(&plugin)
		if err != nil {
			logger.Logger.Errorf("初始化插件配置失败: %v", err)
			panic(fmt.Errorf("初始化插件配置失败: %v", err))
		}
		logger.Logger.Debug("插件配置初始化成功")
	}
	logger.Logger.Debug("插件配置检查成功")
}
