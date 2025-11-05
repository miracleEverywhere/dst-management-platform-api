package room

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
)

// createPost 创建房间
func (h *Handler) roomPost(c *gin.Context) {
	permission, err := h.hasPermission(c)
	if err != nil {
		if err != nil {
			logger.Logger.Error("查询数据库失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}
	}

	if permission {
		var reqForm XRoomTotalInfo
		if err := c.ShouldBindJSON(&reqForm); err != nil {
			logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
			return
		}
		logger.Logger.Debug(utils.StructToFlatString(reqForm))

		reqForm.RoomData.ID = 0
		reqForm.RoomData.Status = true

		room, errCreateRoom := h.roomDao.CreateRoom(&reqForm.RoomData)
		if errCreateRoom != nil {
			logger.Logger.Error("创建房间失败", "err", errCreateRoom)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}

		for _, world := range reqForm.WorldData {
			world.RoomID = room.ID
			if errCreateWorld := h.worldDao.Create(&world); errCreateWorld != nil {
				logger.Logger.Error("创建房间失败", "err", errCreateWorld)
				c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
				return
			}
		}

		reqForm.RoomSettingData.RoomID = room.ID
		if errCreate := h.roomSettingDao.Create(&reqForm.RoomSettingData); errCreate != nil {
			logger.Logger.Error("创建房间失败", "err", errCreate)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "create success"), "data": room})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
	return
}

// roomPut 修改房间
func (h *Handler) roomPut(c *gin.Context) {
	permission, err := h.hasPermission(c)
	if err != nil {
		if err != nil {
			logger.Logger.Error("查询数据库失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}
	}

	if permission {
		var reqForm XRoomTotalInfo
		if err := c.ShouldBindJSON(&reqForm); err != nil {
			logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
			return
		}
		logger.Logger.Debug(utils.StructToFlatString(reqForm))

		err = h.roomDao.UpdateRoom(&reqForm.RoomData)
		if err != nil {
			logger.Logger.Error("更新房间失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}

		err = h.worldDao.UpdateWorlds(&reqForm.WorldData)
		if err != nil {
			logger.Logger.Error("更新房间失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}

		err = h.roomSettingDao.UpdateRoomSetting(&reqForm.RoomSettingData)
		if err != nil {
			logger.Logger.Error("更新房间失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}

		//game := dst.NewGameController(&reqForm.RoomData, &reqForm.WorldData, &reqForm.RoomSettingData)
		//err = game.Save()
		//if err != nil {
		//	logger.Logger.Error("配置写入磁盘失败", "err", err)
		//}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "update success"), "data": reqForm.RoomData})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
	return
}

// listGet 按分页获取集群信息，并附带对应世界信息
func (h *Handler) listGet(c *gin.Context) {
	type ReqForm struct {
		Partition
		GameName string `json:"gameName" form:"gameName"`
	}
	var reqForm ReqForm
	var data dao.PaginatedResult[XRoomWorld]
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": data})
		return
	}
	logger.Logger.Debug(utils.StructToFlatString(reqForm))

	role, _ := c.Get("role")
	var (
		rooms *dao.PaginatedResult[models.Room]
		err   error
	)
	if role.(string) == "admin" {
		// 管理员返回所有房间
		rooms, err = h.roomDao.ListRooms([]int{}, reqForm.GameName, reqForm.Page, reqForm.PageSize)
		if err != nil {
			logger.Logger.Error("查询数据库失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": data})
			return
		}
	} else {
		username, _ := c.Get("username")
		user, err := h.userDao.GetUserByUsername(username.(string))
		if err != nil {
			logger.Logger.Error("查询数据库失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": data})
			return
		}
		if user.Rooms == "" {
			data.Page = reqForm.Page
			data.PageSize = reqForm.PageSize
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
			return
		}
		// 非管理员返回有权限的房间 user.Rooms like "1,2,3"
		roomSlice := strings.Split(user.Rooms, ",")
		var roomIDs []int
		for _, id := range roomSlice {
			intID, err := strconv.Atoi(id)
			if err != nil {
				logger.Logger.Error("查询数据库失败", "err", err)
				c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": data})
				return
			}
			roomIDs = append(roomIDs, intID)
		}
		rooms, err = h.roomDao.ListRooms(roomIDs, reqForm.GameName, reqForm.Page, reqForm.PageSize)
		if err != nil {
			logger.Logger.Error("查询数据库失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": data})
			return
		}

	}

	data.Page = rooms.Page
	data.PageSize = rooms.PageSize
	data.TotalCount = rooms.TotalCount

	// 为房间加上世界信息
	for _, room := range rooms.Data {
		xRoomWorld := XRoomWorld{
			Room:   room,
			Worlds: []models.World{},
		}
		worlds, errWorld := h.worldDao.GetWorldsByRoomIDWthPage(room.ID)
		if errWorld != nil {
			logger.Logger.Error("查询数据库失败", "err", errWorld)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": data})
			return
		}
		xRoomWorld.Worlds = worlds.Data
		data.Data = append(data.Data, xRoomWorld)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}

// roomGet 返回房间、世界、房间设置等所有信息
func (h *Handler) roomGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"id" form:"id"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	logger.Logger.Debug(utils.StructToFlatString(reqForm))

	var data XRoomTotalInfo
	room, err := h.roomDao.GetRoomByID(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}
	data.RoomData = *room

	worlds, err := h.worldDao.GetWorldsByRoomID(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}
	data.WorldData = *worlds

	roomSetting, err := h.roomSettingDao.GetRoomSettingsByRoomID(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}
	data.RoomSettingData = *roomSetting

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}

func (h *Handler) factorGet(c *gin.Context) {
	roomCount, err := h.roomDao.Count(nil)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	worldCount, err := h.worldDao.Count(nil)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	type Data struct {
		Room  int64 `json:"roomCount"`
		World int64 `json:"worldCount"`
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": Data{
		Room:  roomCount,
		World: worldCount,
	}})
}
