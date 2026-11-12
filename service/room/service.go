package room

import (
	"dst-management-platform-api/aichat"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// Service 房间模块业务实现，不依赖 gin
type Service struct {
	roomDao          *dao.RoomDAO
	userDao          *dao.UserDAO
	worldDao         *dao.WorldDAO
	roomSettingDao   *dao.RoomSettingDAO
	favoriteDao      *dao.FavoriteDAO
	globalSettingDao *dao.GlobalSettingDAO
	uidMapDao        *dao.UidMapDAO
	roomAISettingDao *dao.RoomAISettingDAO
	aiManager        *aichat.Manager
}

// New 创建房间模块 service
func New(userDao *dao.UserDAO, roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO, favoriteDao *dao.FavoriteDAO, globalSettingDao *dao.GlobalSettingDAO, uidMapDao *dao.UidMapDAO, roomAISettingDao *dao.RoomAISettingDAO, aiManager *aichat.Manager) *Service {
	return &Service{
		roomDao:          roomDao,
		userDao:          userDao,
		worldDao:         worldDao,
		roomSettingDao:   roomSettingDao,
		favoriteDao:      favoriteDao,
		globalSettingDao: globalSettingDao,
		uidMapDao:        uidMapDao,
		roomAISettingDao: roomAISettingDao,
		aiManager:        aiManager,
	}
}

// Partition 分页参数
type Partition struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"pageSize" form:"pageSize"`
}

// XRoomWorld 房间列表项，附带世界与玩家统计
type XRoomWorld struct {
	models.Room
	Worlds  []models.World  `json:"worlds"`
	Players []cache.Players `json:"players"`
}

// RoomWorldPage 房间列表分页结构，绑定失败时需要返回零值保持原响应形状
type RoomWorldPage = dao.PaginatedResult[XRoomWorld]

// XRoomTotalInfo 房间完整配置，收藏与云存档元数据共用
type XRoomTotalInfo struct {
	RoomData        models.Room        `json:"roomData"`
	WorldData       []models.World     `json:"worldData"`
	RoomSettingData models.RoomSetting `json:"roomSettingData"`
}

// validateWorldNames 校验世界名是否合法
// 世界名会被拼接进bash命令和文件路径，因此必须限定字符集；
// 同时同一房间内世界名不能重复，否则多个世界会使用同一个目录、互相覆盖存档。
// 返回第一个不合法的世界名和对应的错误码，全部合法时返回空字符串和 0
func validateWorldNames(worlds []models.World) (string, int) {
	seen := make(map[string]struct{}, len(worlds))
	for _, world := range worlds {
		name := world.WorldName
		if !utils.IsSafeString(name) {
			return name, constant.CodeRoomInvalidWorldName
		}
		if _, ok := seen[name]; ok {
			return name, constant.CodeRoomDuplicateWorldName
		}
		seen[name] = struct{}{}
	}

	return "", 0
}

// hasCreatePermission 是否拥有房间创建权限
func (s *Service) hasCreatePermission(actor service.Actor) (bool, error) {
	var (
		has    bool
		err    error
		dbUser *models.User
	)

	// 管理员直接返回true
	if actor.IsAdmin() {
		has = true
	} else {
		dbUser, err = s.userDao.GetUserByUsername(actor.Username)
		if err != nil {
			return has, err
		}
		if dbUser.RoomCreation {
			has = true
		}
	}

	return has, err
}

// hasRoomPermission 是否拥有对应房间权限
func (s *Service) hasRoomPermission(actor service.Actor, roomID string) bool {
	// 管理员直接返回true
	if actor.IsAdmin() {
		return true
	}

	dbUser, err := s.userDao.GetUserByUsername(actor.Username)
	if err != nil {
		logger.Logger.Error("查询数据库失败")
		return false
	}
	roomIDs := strings.Split(dbUser.Rooms, ",")
	for _, id := range roomIDs {
		if id == roomID {
			return true
		}
	}

	return false
}

// CreateGuard 校验房间创建权限，返回 nil 表示允许继续。
// 原实现先做权限判断再解析请求体，handler 需要在绑定前调用它保持响应顺序。
func (s *Service) CreateGuard(actor service.Actor) *service.Result {
	permission, err := s.hasCreatePermission(actor)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		r := service.Fail(constant.CodeRoomDBError)
		return &r
	}
	if !permission {
		r := service.Fail(constant.CodeRoomNoPermission)
		return &r
	}

	return nil
}

// Create 创建房间
func (s *Service) Create(actor service.Actor, reqForm XRoomTotalInfo) service.Result {
	permission, err := s.hasCreatePermission(actor)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	if !permission {
		return service.Fail(constant.CodeRoomNoPermission)
	}

	if len(reqForm.WorldData) == 0 {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, actor.Path)
		return service.Fail(constant.CodeRoomBadRequest)
	}

	// 世界名会被拼接进bash命令和文件路径，必须在入口处校验
	if worldName, code := validateWorldNames(reqForm.WorldData); code != 0 {
		logger.Logger.Warnf("世界名校验失败已拦截, api: %s, username: %s, world: %s, code: %d", actor.Path, actor.Username, worldName, code)
		return service.FailF(code, worldName)
	}

	for _, world := range reqForm.WorldData {
		if world.CustomStartupCmd != "" {
			logger.Logger.Infof("请求参数错误: %v, api: %s", err, actor.Path)
			return service.Fail(constant.CodeRoomBadRequest)
		}
	}

	// 验证游戏模式，防止XSS注入
	if !utils.IsValidGameMode(reqForm.RoomData.GameMode) {
		return service.Fail(constant.CodeRoomBadRequest)
	}

	// 端口冲突检测
	// MasterPort（[SHARD] master_port）不参与检测，允许重复
	var ports []int
	for _, world := range reqForm.WorldData {
		ports = append(ports, world.ServerPort, world.MasterServerPort, world.AuthenticationPort)
	}
	if conflictPort := s.checkGamePort(ports, 0); conflictPort != 0 {
		return service.FailF(constant.CodeRoomPortConflict, conflictPort)
	}

	// webhook url 安全检测
	var webhooks []webhook.WebhookItem
	if json.Unmarshal([]byte(reqForm.RoomSettingData.WebhookSetting), &webhooks) == nil {
		for _, w := range webhooks {
			if !utils.IsValidURL(w.URL) {
				logger.Logger.Warnf("非法请求已拦截, api: %s, username: %s", actor.Path, actor.Username)
				return service.Fail(constant.CodeRoomInvalidURL)
			}
		}
	} else {
		logger.Logger.Infof("webhook参数错误, api: %s", actor.Path)
		return service.Fail(constant.CodeRoomBadRequest)
	}

	reqForm.RoomData.ID = 0
	reqForm.RoomData.Status = true

	room, errCreateRoom := s.roomDao.CreateRoom(&reqForm.RoomData)
	if errCreateRoom != nil {
		logger.Logger.Errorf("创建房间失败, err: %v", errCreateRoom)
		return service.Fail(constant.CodeRoomDBError)
	}

	for i := range reqForm.WorldData {
		reqForm.WorldData[i].RoomID = room.ID
		if errCreateWorld := s.worldDao.Create(&reqForm.WorldData[i]); errCreateWorld != nil {
			logger.Logger.Errorf("创建房间失败, err: %v", errCreateWorld)
			return service.Fail(constant.CodeRoomDBError)
		}
	}

	reqForm.RoomSettingData.RoomID = room.ID
	reqForm.RoomSettingData.AnnounceSetting = "[]"
	if errCreate := s.roomSettingDao.Create(&reqForm.RoomSettingData); errCreate != nil {
		logger.Logger.Errorf("创建房间失败, err: %v", errCreate)
		return service.Fail(constant.CodeRoomDBError)
	}

	game := dst.NewGameController(&reqForm.RoomData, &reqForm.WorldData, &reqForm.RoomSettingData, actor.Lang)
	// 原实现在写盘失败后仍继续后续副作用，只是响应码为 201，这里保留副作用并统一返回 201
	saveResult := service.Result{}
	if err = game.SaveAll(); err != nil {
		logger.Logger.Errorf("配置写入磁盘失败, err: %v", err)
		saveResult = service.Fail(constant.CodeRoomWriteFileFail)
	}

	processJobs(game, reqForm.RoomData.ID, reqForm.RoomSettingData)

	// 如果用户不是管理员，且拥有房间创建权限，需要在rooms字段中新增房间id
	if !actor.IsAdmin() {
		user, errUser := s.userDao.GetUserByUsername(actor.Username)
		if errUser != nil {
			logger.Logger.Errorf("获取用户信息失败, err: %v", errUser)
			return service.Fail(constant.CodeRoomDBError)
		}
		var rooms []string
		if user.Rooms != "" {
			rooms = strings.Split(user.Rooms, ",")
		}
		rooms = append(rooms, strconv.Itoa(reqForm.RoomSettingData.RoomID))
		user.Rooms = strings.Join(rooms, ",")
		if errUpdate := s.userDao.UpdateUser(user); errUpdate != nil {
			logger.Logger.Errorf("更新用户信息失败, err: %v", errUpdate)
			return service.Fail(constant.CodeRoomDBError)
		}
	}

	webhook.Snd.Send(webhook.EventRoomCreated, room.ID, reqForm)

	if saveResult.Code != 0 {
		return saveResult
	}

	return service.Msg("create success", room)
}

// Update 修改房间
func (s *Service) Update(actor service.Actor, reqForm XRoomTotalInfo) service.Result {
	// 验证游戏模式，防止XSS注入
	if !utils.IsValidGameMode(reqForm.RoomData.GameMode) {
		return service.Fail(constant.CodeRoomBadRequest)
	}

	if !s.hasRoomPermission(actor, strconv.Itoa(reqForm.RoomData.ID)) {
		return service.Fail(constant.CodeRoomNoPermission)
	}

	if len(reqForm.WorldData) == 0 {
		return service.Fail(constant.CodeRoomBadRequest)
	}

	// 世界名会被拼接进bash命令和文件路径，必须在入口处校验
	if worldName, code := validateWorldNames(reqForm.WorldData); code != 0 {
		logger.Logger.Warnf("世界名校验失败已拦截, api: %s, username: %s, world: %s, code: %d", actor.Path, actor.Username, worldName, code)
		return service.FailF(code, worldName)
	}

	// 关联数据的房间归属由服务端确定，不能信任客户端提交的 RoomID。
	roomID := reqForm.RoomData.ID
	reqForm.RoomSettingData.RoomID = roomID
	for i := range reqForm.WorldData {
		reqForm.WorldData[i].RoomID = roomID
	}

	// 保存会删除并重建世界（世界ID发生变化），先取出旧世界，用于同步世界监控数据
	oldWorlds, err := s.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	// CustomStartupCmd权限校验
	// 该字段会被拼接进 bash -c 执行，只有管理员才能修改（专用接口同样仅限管理员）。
	// 客户端保存时会丢弃世界ID，因此世界名才是世界在房间内的标识，不能用ID匹配，
	// 非管理员提交的值一律丢弃，以数据库中的值为准，数据库中不存在的世界（新增世界）只能使用默认启动命令
	if !actor.IsAdmin() {
		dbCustomStartupCmds := make(map[string]string, len(*oldWorlds))
		for _, dbWorld := range *oldWorlds {
			if dbWorld.WorldName == "" {
				continue
			}
			dbCustomStartupCmds[dbWorld.WorldName] = dbWorld.CustomStartupCmd
		}
		for i := range reqForm.WorldData {
			dbCustomStartupCmd := dbCustomStartupCmds[reqForm.WorldData[i].WorldName]
			if reqForm.WorldData[i].CustomStartupCmd == dbCustomStartupCmd {
				continue
			}
			logger.Logger.Warnf("非管理员修改自定义启动命令已拦截, api: %s, username: %s, world: %s", actor.Path, actor.Username, reqForm.WorldData[i].WorldName)
			reqForm.WorldData[i].CustomStartupCmd = dbCustomStartupCmd
		}
	}

	// 端口冲突检测
	// 修改房间时 MasterPort 允许重复（[SHARD] master_port 在部分部署方式下可以与其它房间相同），因此不参与检测
	var ports []int
	for _, world := range reqForm.WorldData {
		ports = append(ports, world.ServerPort, world.MasterServerPort, world.AuthenticationPort)
	}
	if conflictPort := s.checkGamePort(ports, reqForm.RoomData.ID); conflictPort != 0 {
		return service.FailF(constant.CodeRoomPortConflict, conflictPort)
	}

	// webhook url 安全检测
	var webhooks []webhook.WebhookItem
	if json.Unmarshal([]byte(reqForm.RoomSettingData.WebhookSetting), &webhooks) == nil {
		for _, w := range webhooks {
			if !utils.IsValidURL(w.URL) {
				logger.Logger.Warnf("非法请求已拦截, api: %s, username: %s", actor.Path, actor.Username)
				return service.Fail(constant.CodeRoomInvalidURL)
			}
		}
	} else {
		logger.Logger.Infof("webhook参数错误, api: %s", actor.Path)
		return service.Fail(constant.CodeRoomBadRequest)
	}

	if err = s.roomDao.UpdateConfiguration(&reqForm.RoomData, &reqForm.WorldData, &reqForm.RoomSettingData); err != nil {
		logger.Logger.Errorf("事务更新房间失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	game := dst.NewGameController(&reqForm.RoomData, &reqForm.WorldData, &reqForm.RoomSettingData, actor.Lang)
	// 原实现在写盘失败后仍继续后续副作用，只是响应码为 201，这里保留副作用并统一返回 201
	saveResult := service.Result{}
	if err = game.SaveAll(); err != nil {
		logger.Logger.Errorf("配置写入磁盘失败, err: %v", err)
		saveResult = service.Fail(constant.CodeRoomWriteFileFail)
	}

	if reqForm.RoomData.Status {
		processJobs(game, reqForm.RoomData.ID, reqForm.RoomSettingData)
	} else {
		// 删除所有的定时任务
		jobNames := scheduler.GetJobsByRoomID(reqForm.RoomData.ID)
		logger.Logger.Debug(utils.StructToFlatString(jobNames))
		for _, jobName := range jobNames {
			scheduler.DeleteJob(jobName)
		}
	}
	// 同步世界监控数据：世界ID已在保存时重建，需要按世界名迁移历史数据
	oldWorldIDMap := worldIDByName(*oldWorlds)
	newWorldIDMap := worldIDByName(reqForm.WorldData)
	if reqForm.RoomData.Status && reqForm.RoomSettingData.RoomMetricsEnable {
		migrateWorldMetrics(oldWorldIDMap, newWorldIDMap)
	} else {
		// 关闭监控或房间未激活时，清理该房间已有的监控数据
		deleteWorldMetrics(oldWorldIDMap, newWorldIDMap)
	}
	if err = s.aiManager.Reload(reqForm.RoomData.ID); err != nil {
		logger.Logger.Errorf("重载房间 AI 对话失败, roomID: %d, err: %v", reqForm.RoomData.ID, err)
	}

	webhook.Snd.Send(webhook.EventRoomSettingsUpdated, reqForm.RoomData.ID, reqForm)

	if saveResult.Code != 0 {
		return saveResult
	}

	return service.Msg("update success", reqForm.RoomData)
}

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

// Factor 前端自动分配端口
func (s *Service) Factor() service.Result {
	roomCount, err := s.roomDao.Count(nil)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	worldCount, err := s.worldDao.Count(nil)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	type Data struct {
		Room  int64 `json:"roomCount"`
		World int64 `json:"worldCount"`
	}

	return service.Success(Data{
		Room:  roomCount,
		World: worldCount,
	})
}

// AllBasic 获取room基本信息 name和id
func (s *Service) AllBasic() service.Result {
	rooms, err := s.roomDao.GetRoomBasic()
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	return service.Success(rooms)
}

// PermittedBasic 获取当前用户有权限的room基本信息
func (s *Service) PermittedBasic(actor service.Actor) service.Result {
	rooms, err := s.roomDao.GetRoomBasic()
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	if actor.IsAdmin() {
		return service.Success(rooms)
	}

	user, err := s.userDao.GetUserByUsername(actor.Username)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	roomIDs := make(map[int]struct{})
	for _, roomID := range strings.Split(user.Rooms, ",") {
		if roomID == "" {
			continue
		}
		id, errAtoi := strconv.Atoi(roomID)
		if errAtoi != nil {
			logger.Logger.Errorf("用户房间权限数据无效, username: %s, roomID: %s", user.Username, roomID)
			return service.Fail(constant.CodeRoomDBError)
		}
		roomIDs[id] = struct{}{}
	}

	permittedRooms := make([]dao.RoomBasic, 0, len(roomIDs))
	for _, room := range *rooms {
		if _, ok := roomIDs[room.RoomID]; ok {
			permittedRooms = append(permittedRooms, room)
		}
	}

	return service.Success(permittedRooms)
}

// Worlds 房间下的世界列表
func (s *Service) Worlds(actor service.Actor, roomID int) service.Result {
	if !s.hasRoomPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeRoomNoPermission)
	}

	worlds, err := s.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
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

	return service.Success(data)
}

// UploadTarget 校验上传权限并解析目标房间ID；newRoom 为 true 表示新建房间
func (s *Service) UploadTarget(actor service.Actor, roomIDStr string) (roomID int, newRoom bool, failed *service.Result) {
	if roomIDStr == "" {
		// 新建房间，新建权限验证
		permission, _ := s.hasCreatePermission(actor)
		if !permission {
			r := service.Fail(constant.CodeRoomNoPermission)
			return 0, true, &r
		}
		return 0, true, nil
	}

	// 修改当前房间，修改权限验证
	if !s.hasRoomPermission(actor, roomIDStr) {
		r := service.Fail(constant.CodeRoomNoPermission)
		return 0, false, &r
	}

	id, err := strconv.Atoi(roomIDStr)
	if err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, actor.Path)
		r := service.Fail(constant.CodeRoomBadRequest)
		return 0, false, &r
	}

	return id, false, nil
}

// UploadImport 从已落盘的 zip 导入存档，newRoom 为 true 表示新建房间
func (s *Service) UploadImport(actor service.Actor, savePath, unzipPath string, roomID int, newRoom bool) service.Result {
	_, code, err := s.importRoomFromZip(actor, savePath, unzipPath, roomID, newRoom)
	if err != nil {
		if code == 0 {
			code = constant.CodeRoomUploadSaveFail
		}
		logger.Logger.Errorf("处理上传文件失败, err: %v", err)
		return service.Fail(code)
	}

	return service.Msg("upload success", nil)
}

// Activate 启用房间
func (s *Service) Activate(actor service.Actor, roomID int) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodeRoomBadRequest)
	}

	if !s.hasRoomPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeRoomNoPermission)
	}

	room, err := s.roomDao.GetRoomByID(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	worlds, err := s.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	roomSetting, err := s.roomSettingDao.GetRoomSettingsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	// 先解析定时通知配置，避免启动游戏、写库、删任务之后才发现配置非法
	announceSetting := strings.TrimSpace(roomSetting.AnnounceSetting)
	if announceSetting == "" {
		announceSetting = "[]"
	}
	var announces []scheduler.AnnounceSetting
	if err = json.Unmarshal([]byte(announceSetting), &announces); err != nil {
		logger.Logger.Errorf("获取定时通知设置失败, roomID: %d, err: %v", roomID, err)
		return service.Fail(constant.CodeRoomActivateFail)
	}

	// 启动游戏
	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	_ = game.StartAllWorld()
	// 更新数据库
	room.Status = true
	if err = s.roomDao.UpdateRoom(room); err != nil {
		logger.Logger.Errorf("写入数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	// 添加定时任务
	processJobs(game, roomID, *roomSetting)
	// 添加定时通知
	jobNames := scheduler.GetJobsByType(roomID, "Announce")
	logger.Logger.Debug(utils.StructToFlatString(jobNames))
	for _, jobName := range jobNames {
		// 删除旧通知任务
		scheduler.DeleteJob(jobName)
	}
	logger.Logger.Debug(utils.StructToFlatString(announces))
	for _, announce := range announces {
		// 创建通知任务
		if announce.Status {
			// 注意，-为分隔符，需要删除uuid中的-
			err = scheduler.UpdateJob(&scheduler.JobConfig{
				Name:     fmt.Sprintf("%d-%s-Announce", room.ID, strings.ReplaceAll(announce.ID, "-", "")),
				Func:     scheduler.Announce,
				Args:     []any{game, announce.Content},
				TimeType: scheduler.SecondType,
				Interval: announce.Interval,
				DayAt:    "",
			})
			if err != nil {
				logger.Logger.Errorf("定时通知定时任务处理失败, err: %v", err)
			}
		}
	}
	if err = s.aiManager.Reload(roomID); err != nil {
		logger.Logger.Errorf("重载房间 AI 对话失败, roomID: %d, err: %v", roomID, err)
	}

	webhook.Snd.Send(webhook.EventRoomActivated, roomID, map[string]interface{}{
		"gameID":   room.ID,
		"gameName": room.GameName,
	})

	return service.Msg("activate success", nil)
}

// Deactivate 停用房间
func (s *Service) Deactivate(actor service.Actor, roomID int) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodeRoomBadRequest)
	}

	if !s.hasRoomPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeRoomNoPermission)
	}

	room, err := s.roomDao.GetRoomByID(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	worlds, err := s.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	roomSetting, err := s.roomSettingDao.GetRoomSettingsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	// 关闭游戏进程
	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	_ = game.StopAllWorld()
	// 删除定时任务
	jobNames := scheduler.GetJobsByRoomID(roomID)
	logger.Logger.Debug(utils.StructToFlatString(jobNames))
	for _, jobName := range jobNames {
		scheduler.DeleteJob(jobName)
	}
	// 删除玩家统计
	cache.PlayersStatisticMutex.Lock()
	delete(cache.PlayersStatistic, roomID)
	cache.PlayersStatisticMutex.Unlock()
	// 删除房间内各世界的资源监控数据
	deleteWorldMetrics(worldIDByName(*worlds))
	// 更新数据库
	room.Status = false
	if err = s.roomDao.UpdateRoom(room); err != nil {
		logger.Logger.Errorf("写入数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	s.aiManager.StopRoom(roomID)

	webhook.Snd.Send(webhook.EventRoomDeactivated, roomID, map[string]interface{}{
		"gameID":   room.ID,
		"gameName": room.GameName,
	})

	return service.Msg("deactivate success", nil)
}

// Delete 删除房间
func (s *Service) Delete(actor service.Actor, roomID int) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodeRoomBadRequest)
	}

	nonAdminUsers, err := s.userDao.GetNonAdminUsers()
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	room, err := s.roomDao.GetRoomByID(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	worlds, err := s.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	roomSetting, err := s.roomSettingDao.GetRoomSettingsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	// 删除游戏相关前先停止 AI 日志监听，避免继续读取即将删除的文件。
	s.aiManager.StopRoom(roomID)
	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	err = game.DeleteRoom()
	if err != nil {
		if reloadErr := s.aiManager.Reload(roomID); reloadErr != nil {
			logger.Logger.Errorf("恢复房间 AI 对话失败, roomID: %d, err: %v", roomID, reloadErr)
		}
		logger.Logger.Errorf("删除游戏相关文件失败, err: %v", err)
		return service.Fail(constant.CodeRoomDeleteFail)
	}
	// 删除定时任务
	jobNames := scheduler.GetJobsByRoomID(roomID)
	logger.Logger.Debug(utils.StructToFlatString(jobNames))
	for _, jobName := range jobNames {
		scheduler.DeleteJob(jobName)
	}
	// 删除玩家统计
	cache.PlayersStatisticMutex.Lock()
	delete(cache.PlayersStatistic, roomID)
	cache.PlayersStatisticMutex.Unlock()
	cache.PlayersOnlineTimeMutex.Lock()
	delete(cache.PlayersOnlineTime, roomID)
	cache.PlayersOnlineTimeMutex.Unlock()
	// 删除重置计时
	cache.RoomNoPlayersSecondsMutex.Lock()
	delete(cache.RoomNoPlayersSeconds, roomID)
	cache.RoomNoPlayersSecondsMutex.Unlock()
	// 删除房间内各世界的资源监控数据
	deleteWorldMetrics(worldIDByName(*worlds))
	// 更新用户权限
	roomIDStr := strconv.Itoa(roomID)
	for _, user := range *nonAdminUsers {
		if user.Rooms != "" {
			roomParts := strings.Split(user.Rooms, ",")
			var newRooms []string
			for _, rid := range roomParts {
				if rid != roomIDStr {
					newRooms = append(newRooms, rid)
				}
			}
			user.Rooms = strings.Join(newRooms, ",")
			if err = s.userDao.UpdateUser(&user); err != nil {
				logger.Logger.Errorf("更新数据库失败, err: %v", err)
				return service.Fail(constant.CodeRoomDBError)
			}
		}

	}
	// webhook 通知
	webhook.Snd.Send(webhook.EventRoomDeleted, roomID, map[string]interface{}{
		"gameID":   room.ID,
		"gameName": room.GameName,
	})

	// 更新数据库
	if err = s.roomDao.Delete(room); err != nil {
		logger.Logger.Errorf("更新数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	if err = s.roomSettingDao.Delete(roomSetting); err != nil {
		logger.Logger.Errorf("更新数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	for _, world := range *worlds {
		if err = s.worldDao.Delete(&world); err != nil {
			logger.Logger.Errorf("更新数据库失败, err: %v", err)
			return service.Fail(constant.CodeRoomDBError)
		}
	}
	if err = s.uidMapDao.DeleteUidMapByRoomID(roomID); err != nil {
		logger.Logger.Errorf("更新数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	if err = s.roomAISettingDao.DeleteByRoomID(roomID); err != nil {
		logger.Logger.Errorf("删除房间 AI 配置失败, roomID: %d, err: %v", roomID, err)
		return service.Fail(constant.CodeRoomDBError)
	}

	// 更新webhook全局通知房间
	var globalSetting models.GlobalSetting
	if err := s.globalSettingDao.GetGlobalSetting(&globalSetting); err == nil && globalSetting.WebhookSetting != "" {
		var (
			items        []webhook.GlobalWebhookItem
			needUpdateDb = false
		)
		if json.Unmarshal([]byte(globalSetting.WebhookSetting), &items) == nil {
			for i, item := range items {
				if len(item.RoomIDs) == 0 {
					continue
				}
				if utils.Contains(item.RoomIDs, room.ID) {
					needUpdateDb = true
					items[i].RoomIDs = utils.RemoveItem(item.RoomIDs, room.ID)
				}
			}
		}

		if needUpdateDb {
			jsonData, errMarshal := json.Marshal(items)
			if errMarshal != nil {
				logger.Logger.Errorf("更新webhook房间ID失败: %v", errMarshal)
			} else {
				globalSetting.WebhookSetting = string(jsonData)
				if errUpdate := s.globalSettingDao.UpdateGlobalSetting(&globalSetting); errUpdate != nil {
					logger.Logger.Errorf("更新webhook房间ID失败: %v", errUpdate)
				}
			}
		}
	}

	return service.Msg("delete success", nil)
}

// WebhookTest 测试 webhook 地址
func (s *Service) WebhookTest(actor service.Actor, url, secret string) service.Result {
	// webhook url 安全检测
	if !utils.IsValidURL(url) {
		logger.Logger.Warnf("非法请求已拦截, api: %s, username: %s", actor.Path, actor.Username)
		return service.Fail(constant.CodeRoomInvalidURL)
	}

	err := webhook.Snd.SendTest(url, secret)
	if err != nil {
		logger.Logger.Warnf("webhook 测试失败, url: %s, err: %v", url, err)
		return service.FailF(constant.CodeRoomWebhookTestFail, err.Error())
	}

	return service.Msg("webhook test success", nil)
}

// WebhookEvents 返回可用的 webhook 事件类型
func (s *Service) WebhookEvents() service.Result {
	return service.Success(webhook.AllEventTypes)
}

// WorldCustomStartupCmd 世界自定义启动命令
type WorldCustomStartupCmd struct {
	WorldID          int    `json:"worldID" form:"worldID"`
	CustomStartupCmd string `json:"customStartupCmd" form:"customStartupCmd"`
}

// CustomStartupCmdPut 修改世界自定义启动命令
func (s *Service) CustomStartupCmdPut(actor service.Actor, roomID int, reqWorlds []WorldCustomStartupCmd) service.Result {
	if len(reqWorlds) == 0 || roomID == 0 {
		return service.Fail(constant.CodeRoomBadRequest)
	}

	worlds, err := s.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	cmdMap := make(map[int]string, len(reqWorlds))
	for _, reqWorld := range reqWorlds {
		cmdMap[reqWorld.WorldID] = reqWorld.CustomStartupCmd
	}

	for i := range *worlds {
		if cmd, ok := cmdMap[(*worlds)[i].ID]; ok {
			(*worlds)[i].CustomStartupCmd = cmd
		}
	}

	if err = s.worldDao.UpdateWorlds(worlds); err != nil {
		logger.Logger.Errorf("更新数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomUpdateFail)
	}

	return service.Msg("update success", nil)
}

// checkGamePort 检测端口是否冲突，excludeRoomID 为 0 时不排除任何房间
// 先检测入参内部是否重复，再检测与数据库中已有的世界端口是否冲突
// MasterPort（[SHARD] master_port）不参与检测，允许重复
// 返回冲突的端口，0 表示无冲突
func (s *Service) checkGamePort(ports []int, excludeRoomID int) int {
	// 1. 检测入参端口自身是否重复
	seen := make(map[int]bool)
	for _, port := range ports {
		if seen[port] {
			return port
		}
		seen[port] = true
	}

	// 2. 获取所有世界端口
	worlds, err := s.worldDao.Query(1, 10000, nil)
	if err != nil {
		logger.Logger.Errorf("checkGamePort 查询世界失败, err: %v", err)
		return 0
	}

	// 3. 已使用的端口集合
	usedPorts := make(map[int]bool)
	for _, world := range worlds.Data {
		if world.RoomID != excludeRoomID {
			usedPorts[world.ServerPort] = true
			usedPorts[world.MasterServerPort] = true
			usedPorts[world.AuthenticationPort] = true
		}
	}

	// 4. 检查与已有端口是否冲突
	for _, port := range ports {
		if usedPorts[port] {
			return port
		}
	}

	return 0
}
