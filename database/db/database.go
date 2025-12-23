package db

import (
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"fmt"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	dbLogger "gorm.io/gorm/logger"
)

var DB *gorm.DB

func InitDB(dbPath string) {
	if _, err := os.Stat(dbPath); os.IsNotExist(err) {
		err = os.MkdirAll(dbPath, os.ModePerm)
		if err != nil {
			panic("无法创建日志目录: " + err.Error())
		}
	}

	var err error
	dsn := fmt.Sprintf("%s/dmp.db?cache=shared", dbPath)
	logger.Logger.Debug(fmt.Sprintf("数据库连接为%s", dsn))

	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: dbLogger.Default.LogMode(dbLogger.Silent),
	})
	if err != nil {
		logger.Logger.Error("数据库连接失败", "err", err)
		panic(fmt.Sprintf("数据库连接失败: %s", err.Error()))
	}

	logger.Logger.Info("数据库连接成功")

	CheckTables()
}

func CheckTables() {
	logger.Logger.Debug("正在检查数据库表结构")
	err := DB.AutoMigrate(
		&models.User{},
		&models.System{},
		&models.Room{},
		&models.World{},
		&models.RoomSetting{},
		&models.GlobalSetting{},
		&models.UidMap{},
	)
	if err != nil {
		logger.Logger.Error("数据库表结构检查失败", "err", err)
		panic(fmt.Sprintf("数据库表结构检查失败: %s", err.Error()))
	}
	logger.Logger.Debug("数据库表结构检查完成")
}
