package room

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"strconv"
	"strings"
)

// List 按分页获取集群信息，并附带对应世界信息
func (s *Service) List(actor service.Actor, gameName string, page, pageSize int) service.Result {
	var data dao.PaginatedResult[XRoomWorld]
	var (
		rooms *dao.PaginatedResult[models.Room]
		err   error
	)
	if actor.IsAdmin() {
		// 管理员返回所有房间
		rooms, err = s.roomDao.ListRooms([]int{}, gameName, page, pageSize)
		if err != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", err)
			return service.Result{Code: constant.CodeRoomDBError, Data: data}
		}
	} else {
		user, errUser := s.userDao.GetUserByUsername(actor.Username)
		if errUser != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", errUser)
			return service.Result{Code: constant.CodeRoomDBError, Data: data}
		}
		// 非管理员无房间权限直接返回
		if user.Rooms == "" {
			data.Page = page
			data.PageSize = pageSize
			return service.Success(data)
		}
		// 非管理员返回有权限的房间 user.Rooms like "1,2,3"
		roomSlice := strings.Split(user.Rooms, ",")
		var roomIDs []int
		for _, id := range roomSlice {
			intID, errAtoi := strconv.Atoi(id)
			if errAtoi != nil {
				logger.Logger.Errorf("查询数据库失败, err: %v", errAtoi)
				return service.Result{Code: constant.CodeRoomDBError, Data: data}
			}
			roomIDs = append(roomIDs, intID)
		}
		rooms, err = s.roomDao.ListRooms(roomIDs, gameName, page, pageSize)
		if err != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", err)
			return service.Result{Code: constant.CodeRoomDBError, Data: data}
		}

	}

	var globalSetting models.GlobalSetting
	err = s.globalSettingDao.GetGlobalSetting(&globalSetting)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Result{Code: constant.CodeRoomDBError, Data: data}
	}

	data.Page = rooms.Page
	data.PageSize = rooms.PageSize
	data.TotalCount = rooms.TotalCount

	// 统计窗口按全局采集间隔换算，间隔异常时按默认 60 秒处理，避免除零
	playerGetFrequency := globalSetting.PlayerGetFrequency
	if playerGetFrequency <= 0 {
		playerGetFrequency = 60
	}
	dataLength := 3600 / playerGetFrequency

	// 为房间加上世界信息
	for _, room := range rooms.Data {
		xRoomWorld := XRoomWorld{
			Room:   room,
			Worlds: []models.World{},
		}
		worlds, errWorld := s.worldDao.GetWorldsByRoomIDWthPage(room.ID)
		if errWorld != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", errWorld)
			return service.Result{Code: constant.CodeRoomDBError, Data: data}
		}
		xRoomWorld.Worlds = worlds.Data
		// 统计缓存由调度器持锁写入，这里在锁内复制切片，避免响应编码期间
		// 与调度器的 append/裁剪并发读写同一块内存。
		cache.PlayersStatisticMutex.Lock()
		statistics := cache.PlayersStatistic[room.ID]
		if len(statistics) > 0 {
			// 返回最近一个小时的数据
			if len(statistics) > dataLength {
				statistics = statistics[len(statistics)-dataLength:]
			}
			xRoomWorld.Players = make([]cache.Players, len(statistics))
			copy(xRoomWorld.Players, statistics)
		} else {
			xRoomWorld.Players = []cache.Players{}
		}
		cache.PlayersStatisticMutex.Unlock()
		data.Data = append(data.Data, xRoomWorld)
	}

	return service.Success(data)
}

// Get 返回房间、世界、房间设置等所有信息
func (s *Service) Get(actor service.Actor, roomID int) service.Result {
	if !s.hasRoomPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeRoomNoPermission)
	}

	var data XRoomTotalInfo
	room, err := s.roomDao.GetRoomByID(roomID)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	data.RoomData = *room

	worlds, err := s.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	data.WorldData = *worlds

	// 归一化世界/模组配置，剥离 BOM、Klei 引擎头等非法前缀，避免前端世界配置页解析失败；
	// 同时可自愈历史导入的脏数据：前端拿到干净内容并保存后会写回数据库与磁盘。
	for i := range data.WorldData {
		data.WorldData[i].LevelData = utils.NormalizeLuaConfig(data.WorldData[i].LevelData)
		data.WorldData[i].ModData = utils.NormalizeLuaConfig(data.WorldData[i].ModData)
	}

	roomSetting, err := s.roomSettingDao.GetRoomSettingsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	data.RoomSettingData = *roomSetting

	return service.Success(data)
}

// WorldMetrics 房间内各世界的资源监控数据
type WorldMetrics struct {
	WorldID   int                      `json:"worldID"`
	WorldName string                   `json:"worldName"`
	Metrics   []cache.WorldMetricsData `json:"metrics"`
}

// Metrics 获取房间内各世界的资源监控数据，数据来源于内存缓存
func (s *Service) Metrics(actor service.Actor, roomID, timeRange int) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodeRoomBadRequest)
	}

	if !s.hasRoomPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeRoomNoPermission)
	}

	worlds, err := s.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	// 默认1小时
	reqLength := timeRange * 60
	if reqLength <= 0 {
		reqLength = 60
	}

	worldMetricsData := make([]WorldMetrics, 0, len(*worlds))
	cache.WorldMetricsMutex.RLock()
	for _, world := range *worlds {
		metrics := make([]cache.WorldMetricsData, 0)
		if worldMetrics, ok := cache.WorldMetrics[world.ID]; ok {
			if len(worldMetrics) > reqLength {
				metrics = append(metrics, worldMetrics[len(worldMetrics)-reqLength:]...)
			} else {
				metrics = append(metrics, worldMetrics...)
			}
		}
		worldMetricsData = append(worldMetricsData, WorldMetrics{
			WorldID:   world.ID,
			WorldName: world.WorldName,
			Metrics:   metrics,
		})
	}
	cache.WorldMetricsMutex.RUnlock()

	return service.Success(worldMetricsData)
}
