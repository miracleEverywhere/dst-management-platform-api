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
	DB, err = gorm.Open(sqlite.Open(fmt.Sprintf("%s/dmp.db?cache=shared", utils.DBPath)), &gorm.Config{
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
	err := DB.AutoMigrate(&models.User{}, &models.System{}, &models.Room{}, &models.World{})
	if err != nil {
		slog.Logger.Error("数据库检查失败", "err", err)
		panic(fmt.Sprintf("数据库检查失败: %s", err.Error()))
	}
}
