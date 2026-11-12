package tools

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// AnnounceGet 获取定时通知设置
func (s *Service) AnnounceGet(actor service.Actor, roomID int) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodeToolsBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeToolsNoPermission)
	}

	roomSetting, err := s.roomSettingDao.GetRoomSettingsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeToolsDBError)
	}

	return service.Success(roomSetting.AnnounceSetting)
}

// AnnouncePut 保存定时通知设置并按需重建定时任务
func (s *Service) AnnouncePut(actor service.Actor, roomID int, setting string) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodeToolsBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeToolsNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeToolsDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)

	roomSetting.AnnounceSetting = setting

	err = s.roomSettingDao.UpdateRoomSetting(roomSetting)
	if err != nil {
		logger.Logger.Errorf("更新通知设置失败, err: %v", err)
		return service.Fail(constant.CodeToolsUpdateFail)
	}

	if room.Status {
		// 更新定时任务
		jobNames := scheduler.GetJobsByType(roomID, "Announce")
		logger.Logger.Debug(utils.StructToFlatString(jobNames))
		for _, jobName := range jobNames {
			// 删除所有通知任务
			scheduler.DeleteJob(jobName)
		}
		var announces []scheduler.AnnounceSetting
		if err = json.Unmarshal([]byte(roomSetting.AnnounceSetting), &announces); err != nil {
			logger.Logger.Errorf("获取定时通知设置失败, err: %v", err)
			return service.Fail(constant.CodeToolsUpdateFail)
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
	}

	return service.Msg("update success", nil)
}
