package main

import (
	"dst-management-platform-api/app/room"
	"dst-management-platform-api/app/user"
	"dst-management-platform-api/constants"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/initialize"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	"runtime"
)

func main() {
	// 启动后，执行一些初始化和检查
	initialize.Initialize()

	if utils.VersionShow {
		fmt.Println(constants.Version + "\n" + runtime.Version())
		return
	}

	userDao := dao.NewUserDAO(db.DB)
	systemDao := dao.NewSystemDAO(db.DB)
	roomDao := dao.NewRoomDAO(db.DB)

	r := gin.Default()

	userHandler := user.NewUserHandler(userDao, systemDao)
	userHandler.RegisterRoutes(r)
	roomHandler := room.NewRoomHandler(roomDao, userDao)
	roomHandler.RegisterRoutes(r)

	// 启动服务器
	err := r.Run(fmt.Sprintf(":%d", utils.BindPort))
	if err != nil {
		logger.Logger.Error("启动服务器失败", "err", err)
		panic(err)
	}
}
