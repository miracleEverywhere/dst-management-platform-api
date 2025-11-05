package db

import (
	"dst-management-platform-api/database/models"
	slog "dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"os"
)

var DB *gorm.DB

func InitDB() {
	if _, err := os.Stat(utils.DBPath); os.IsNotExist(err) {
		err = os.MkdirAll(utils.DBPath, os.ModePerm)
		if err != nil {
			panic("无法创建日志目录: " + err.Error())
		}
	}

	var err error
	dsn := fmt.Sprintf("%s/dmp.db?cache=shared", utils.DBPath)
	slog.Logger.Debug(fmt.Sprintf("数据库连接为%s", dsn))

	DB, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		slog.Logger.Error("数据库连接失败", "err", err)
		panic(fmt.Sprintf("数据库连接失败: %s", err.Error()))
	}

	slog.Logger.Info("数据库连接成功")

	CheckTables()
}

func CheckTables() {
	slog.Logger.Debug("正在检查数据库表结构")
	err := DB.AutoMigrate(
		&models.User{},
		&models.System{},
		&models.Room{},
		&models.World{},
		&models.RoomSetting{},
		&models.GlobalSetting{},
	)
	if err != nil {
		slog.Logger.Error("数据库表结构检查失败", "err", err)
		panic(fmt.Sprintf("数据库表结构检查失败: %s", err.Error()))
	}
	slog.Logger.Debug("数据库表结构检查完成")
}
