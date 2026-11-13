package room

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
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
	if err = validateAnnounceIntervals(announces); err != nil {
		logger.Logger.Errorf("定时通知间隔无效, roomID: %d, err: %v", roomID, err)
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
