package platform

import (
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func (h *Handler) overviewGet(c *gin.Context) {
	type Data struct {
		RunningTime int64  `json:"runningTime"`
		Memory      uint64 `json:"memory"`
		RoomCount   int64  `json:"roomCount"`
		WorldCount  int64  `json:"worldCount"`
		UserCount   int64  `json:"userCount"`
	}

	// 运行时间
	t := time.Since(utils.StartTime).Seconds()
	// 内存占用
	mem := getRES()
	// 房间数
	roomCount, err := h.roomDao.Count(nil)
	if err != nil {
		logger.Logger.Error("统计房间数失败")
		roomCount = 0
	}
	// 世界数
	worldCount, err := h.worldDao.Count(nil)
	if err != nil {
		logger.Logger.Error("统计世界数失败")
		worldCount = 0
	}
	// 用户数
	userCount, err := h.userDao.Count(nil)
	if err != nil {
		logger.Logger.Error("统计用户数失败")
		userCount = 0
	}
	// TODO 1小时cpu、内存、网络上行、网络下行最大值
	// TODO 玩家数最多的的房间Top3

	data := Data{
		RunningTime: int64(t),
		Memory:      mem,
		RoomCount:   roomCount,
		WorldCount:  worldCount,
		UserCount:   userCount,
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}
