package room

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
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

		game := dst.NewGameController(&reqForm.RoomData, &reqForm.WorldData, &reqForm.RoomSettingData, c.Request.Header.Get("X-I18n-Lang"))
		err = game.SaveAll()
		if err != nil {
			logger.Logger.Error("配置写入磁盘失败", "err", err)
			c.JSON(http.StatusOK, gin.H{
				"code":    201,
				"message": message.Get(c, "write file fail"),
				"data":    nil,
			})
		}

		processJobs(game, reqForm.RoomData.ID, reqForm.RoomSettingData)

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
		// logger.Logger.Debug(utils.StructToFlatString(reqForm))

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

		game := dst.NewGameController(&reqForm.RoomData, &reqForm.WorldData, &reqForm.RoomSettingData, c.Request.Header.Get("X-I18n-Lang"))
		err = game.SaveAll()
		if err != nil {
			logger.Logger.Error("配置写入磁盘失败", "err", err)
			c.JSON(http.StatusOK, gin.H{
				"code":    201,
				"message": message.Get(c, "write file fail"),
				"data":    nil,
			})
		}

		processJobs(game, reqForm.RoomData.ID, reqForm.RoomSettingData)

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
		// 非管理员无房间权限直接返回
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

	var globalSetting models.GlobalSetting
	err = h.globalSettingDao.GetGlobalSetting(&globalSetting)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": data})
		return
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
		if len(db.PlayersStatistic[room.ID]) > 0 {
			dataLength := 3600 / globalSetting.PlayerGetFrequency
			// 返回最近一个小时的数据
			if len(db.PlayersStatistic[room.ID]) > dataLength {
				xRoomWorld.Players = db.PlayersStatistic[room.ID][len(db.PlayersStatistic[room.ID])-dataLength:]
			} else {
				xRoomWorld.Players = db.PlayersStatistic[room.ID]
			}

		} else {
			xRoomWorld.Players = []db.Players{}
		}
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

// factorGet 前端自动分配端口
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

// allRoomBasicGet 获取room基本信息 name和id
func (h *Handler) allRoomBasicGet(c *gin.Context) {
	rooms, err := h.roomDao.GetRoomBasic()
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": rooms})
}

func (h *Handler) roomWorldsGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	worlds, err := h.worldDao.GetWorldsByRoomID(reqForm.RoomID)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	type Data struct {
		ID        int    `json:"id"`
		WorldName string `json:"worldName"`
	}

	var data []Data

	for _, world := range *worlds {
		data = append(data, Data{
			ID:        world.ID,
			WorldName: world.WorldName,
		})
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}

func (h *Handler) uploadPost(c *gin.Context) {
	roomIDStr := c.PostForm("roomID")
	roomID, err := strconv.Atoi(roomIDStr)
	if err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	newRoom := false
	if roomIDStr == "" {
		// 新建房间，新建权限验证
		permission, _ := h.hasPermission(c)
		if !permission {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
			return
		}
		newRoom = true
	} else {
		// 修改当前房间，修改权限验证
		permission := h.hasRoomPermission(c, roomIDStr)
		if !permission {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
			return
		}
	}

	file, err := c.FormFile("file")
	if err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
	}

	// 创建上传文件保存目录
	err = utils.EnsureDirExists(fmt.Sprintf("%s/upload", utils.DmpFiles))
	if err != nil {
		logger.Logger.Error("创建上传目录失败", "err", err)
		c.JSON(http.StatusOK, gin.H{
			"code":    201,
			"message": message.Get(c, "upload save fail"),
			"data":    nil,
		})
	}
	//保存上传的文件
	savePath := fmt.Sprintf("%s/upload/", utils.DmpFiles) + file.Filename
	if err = c.SaveUploadedFile(file, savePath); err != nil {
		logger.Logger.Error("文件保存失败", "err", err)
		c.JSON(http.StatusOK, gin.H{
			"code":    201,
			"message": message.Get(c, "upload save fail"),
			"data":    nil,
		})
		return
	}

	var (
		room            models.Room
		worlds          []models.World
		roomSetting     models.RoomSetting
		uploadExtraInfo UploadExtraInfo
	)

	errMsg, err := handleUpload(savePath, &room, &worlds, &roomSetting, &uploadExtraInfo)
	if err != nil {
		logger.Logger.Error("处理上传文件失败", "err", err)
		c.JSON(http.StatusOK, gin.H{
			"code":    201,
			"message": message.Get(c, errMsg),
			"data":    nil,
		})
		return
	}

	if len(uploadExtraInfo.worldPath) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":    201,
			"message": message.Get(c, "no available worlds found"),
			"data":    nil,
		})
		return
	}

	// 设置所有的port和roomSetting
	if newRoom {
		room.Status = true
		// port
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

		room.MasterPort = 21000 + int(roomCount) + 1
		for index, world := range worlds {
			world.ServerPort = 11000 + int(worldCount) + index + 1
			world.MasterServerPort = 31000 + int(worldCount) + index + 1
			world.AuthenticationPort = 41000 + int(worldCount) + index + 1
		}

		// roomSetting
		roomSetting.BackupEnable = true
		roomSetting.BackupSetting = "[{\"time\":\"06:00:00\"}]"
		roomSetting.BackupCleanEnable = false
		roomSetting.BackupCleanSetting = 30
		roomSetting.RestartEnable = false
		roomSetting.RestartSetting = "06:30:00"
		roomSetting.KeepaliveEnable = false
		roomSetting.KeepaliveSetting = 30
		roomSetting.ScheduledStartStopEnable = false
		roomSetting.ScheduledStartStopSetting = "{\"start\":\"07:00:00\",\"stop\":\"01:00:00\"}"
		roomSetting.StartType = "32-bit"
	} else {
		dbRoom, err := h.roomDao.GetRoomByID(roomID)
		if err != nil {
			logger.Logger.Error("查询数据库失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}
		room.MasterPort = dbRoom.MasterPort
		// 设置roomID
		room.ID = roomID

		dbWorlds, err := h.worldDao.GetWorldsByRoomID(roomID)
		if err != nil {
			logger.Logger.Error("查询数据库失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}
		if len(worlds) != len(*dbWorlds) {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "number of worlds does not match"), "data": nil})
			return
		}
		for index, world := range worlds {
			world.ServerPort = (*dbWorlds)[index].ServerPort
			world.MasterServerPort = (*dbWorlds)[index].MasterServerPort
			world.AuthenticationPort = (*dbWorlds)[index].AuthenticationPort
			// 设置roomID
			world.RoomID = roomID
		}
	}

	// 写入数据库
	if newRoom {
		_, err = h.roomDao.CreateRoom(&room)
		if err != nil {
			logger.Logger.Error("创建房间失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}
		for _, world := range worlds {
			world.RoomID = room.ID
			if errCreateWorld := h.worldDao.Create(&world); errCreateWorld != nil {
				logger.Logger.Error("创建房间失败", "err", errCreateWorld)
				c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
				return
			}
		}

		roomSetting.RoomID = room.ID
		if errCreate := h.roomSettingDao.Create(&roomSetting); errCreate != nil {
			logger.Logger.Error("创建房间失败", "err", errCreate)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}
	} else {
		err = h.roomDao.UpdateRoom(&room)
		if err != nil {
			logger.Logger.Error("更新房间失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}

		err = h.worldDao.UpdateWorlds(&worlds)
		if err != nil {
			logger.Logger.Error("更新房间失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}
		//不更新roomSetting
	}

	game := dst.NewGameController(&room, &worlds, &roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	err = game.SaveAll()
	if err != nil {
		logger.Logger.Error("配置写入磁盘失败", "err", err)
		c.JSON(http.StatusOK, gin.H{
			"code":    201,
			"message": message.Get(c, "write file fail"),
			"data":    nil,
		})
		return
	}

	clusterPath := fmt.Sprintf("%s/Cluster_%d", utils.ClusterPath, room.ID)

	// 设置三个名单
	err = utils.TruncAndWriteFile(fmt.Sprintf("%s/adminlist.txt", clusterPath), uploadExtraInfo.adminlist)
	if err != nil {
		logger.Logger.Error("设置管理员失败", "err", err)
	}
	err = utils.TruncAndWriteFile(fmt.Sprintf("%s/blocklist.txt", clusterPath), uploadExtraInfo.blocklist)
	if err != nil {
		logger.Logger.Error("设置黑名单失败", "err", err)
	}
	err = utils.TruncAndWriteFile(fmt.Sprintf("%s/whitelist.txt", clusterPath), uploadExtraInfo.whitelist)
	if err != nil {
		logger.Logger.Error("设置预留位失败", "err", err)
	}

	// 覆盖save目录
	for _, world := range uploadExtraInfo.worldPath {
		err = utils.RemoveDir(fmt.Sprintf("%s/%s/save", clusterPath, world.name))
		if err != nil {
			logger.Logger.Error("删除旧存档数据失败", "err", err)
			continue
		}
		err = utils.BashCMD(fmt.Sprintf("cp -r %s %s", world.path, fmt.Sprintf("%s/%s/", clusterPath, world.name)))
		if err != nil {
			logger.Logger.Error("复制存档数据失败", "err", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "upload success"), "data": nil})
}
