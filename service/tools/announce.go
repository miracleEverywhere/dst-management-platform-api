package tools

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/service"
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

	// 先解析并校验，校验失败时不能写库、不能删旧任务
	setting, announces, err := parseAnnounceSetting(setting)
	if err != nil {
		logger.Logger.Infof("通知设置校验失败, roomID: %d, err: %v", roomID, err)
		return service.Fail(constant.CodeToolsBadRequest)
	}

	oldSetting := roomSetting.AnnounceSetting
	_, oldAnnounces, oldErr := parseAnnounceSetting(oldSetting)
	if oldErr != nil {
		logger.Logger.Warnf("旧通知设置无法解析，更新失败时无法恢复旧任务, roomID: %d, err: %v", roomID, oldErr)
	}

	if room.Status {
		if err = replaceAnnounceJobs(game, room.ID, announces); err != nil {
			logger.Logger.Errorf("定时通知定时任务处理失败, roomID: %d, err: %v", roomID, err)
			if oldErr == nil {
				if restoreErr := replaceAnnounceJobs(game, room.ID, oldAnnounces); restoreErr != nil {
					logger.Logger.Errorf("恢复旧通知定时任务失败, roomID: %d, err: %v", roomID, restoreErr)
				}
			}
			return service.Fail(constant.CodeToolsUpdateFail)
		}
	}

	roomSetting.AnnounceSetting = setting

	err = s.roomSettingDao.UpdateRoomSetting(roomSetting)
	if err != nil {
		logger.Logger.Errorf("更新通知设置失败, err: %v", err)
		if room.Status && oldErr == nil {
			if restoreErr := replaceAnnounceJobs(game, room.ID, oldAnnounces); restoreErr != nil {
				logger.Logger.Errorf("恢复旧通知定时任务失败, roomID: %d, err: %v", roomID, restoreErr)
			}
		}
		return service.Fail(constant.CodeToolsUpdateFail)
	}

	return service.Msg("update success", nil)
}

// parseAnnounceSetting 解析并校验通知配置，空配置按 [] 处理。
// 只有启用中的通知需要 ID 与大于 0 的间隔，否则无法注册定时任务。
func parseAnnounceSetting(raw string) (string, []scheduler.AnnounceSetting, error) {
	setting := strings.TrimSpace(raw)
	if setting == "" {
		setting = "[]"
	}

	var announces []scheduler.AnnounceSetting
	if err := json.Unmarshal([]byte(setting), &announces); err != nil {
		return "", nil, err
	}
	if announces == nil {
		// JSON null 按空列表处理，避免前端拿到 null 后无法操作列表
		announces = []scheduler.AnnounceSetting{}
		setting = "[]"
	}

	for _, announce := range announces {
		if !announce.Status {
			continue
		}
		if strings.TrimSpace(announce.ID) == "" {
			return "", nil, fmt.Errorf("启用中的通知缺少 ID")
		}
		if err := scheduler.ValidateInterval(scheduler.SecondType, announce.Interval); err != nil {
			return "", nil, fmt.Errorf("启用中的通知间隔无效: %w", err)
		}
	}

	return setting, announces, nil
}

// replaceAnnounceJobs 用新的通知列表替换房间的定时通知任务。
// 先注册启用的通知，全部成功后再删除新列表之外的任务；注册失败时旧任务保持不变。
func replaceAnnounceJobs(game *dst.Game, roomID int, announces []scheduler.AnnounceSetting) error {
	jobNames := scheduler.GetJobsByType(roomID, "Announce")
	keep := make(map[string]struct{}, len(announces))

	for _, announce := range announces {
		if !announce.Status {
			continue
		}
		// 注意，-为分隔符，需要删除uuid中的-
		name := fmt.Sprintf("%d-%s-Announce", roomID, strings.ReplaceAll(announce.ID, "-", ""))
		if _, exists := keep[name]; exists {
			continue
		}
		keep[name] = struct{}{}

		err := scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     name,
			Func:     scheduler.Announce,
			Args:     []any{game, announce.Content},
			TimeType: scheduler.SecondType,
			Interval: announce.Interval,
			DayAt:    "",
		})
		if err != nil {
			return err
		}
	}

	for _, jobName := range jobNames {
		if _, ok := keep[jobName]; !ok {
			scheduler.DeleteJob(jobName)
		}
	}

	return nil
}
