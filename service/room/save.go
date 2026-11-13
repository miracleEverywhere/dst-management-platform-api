package room

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"encoding/json"
	"strconv"
	"strings"
)

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

	// 房间保存也可能携带通知设置，写库前先校验间隔，避免非法值进入数据库。
	if announceSetting := strings.TrimSpace(reqForm.RoomSettingData.AnnounceSetting); announceSetting != "" {
		var announces []scheduler.AnnounceSetting
		if err = json.Unmarshal([]byte(announceSetting), &announces); err != nil {
			logger.Logger.Infof("通知设置解析失败, roomID: %d, err: %v", reqForm.RoomData.ID, err)
			return service.Fail(constant.CodeRoomBadRequest)
		}
		if err = validateAnnounceIntervals(announces); err != nil {
			logger.Logger.Infof("通知间隔校验失败, roomID: %d, err: %v", reqForm.RoomData.ID, err)
			return service.Fail(constant.CodeRoomBadRequest)
		}
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
