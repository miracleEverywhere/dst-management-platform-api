package room

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"encoding/json"
	"fmt"
)

// validateAnnounceIntervals 校验启用的通知间隔，避免注册任务时 time.Duration 溢出。
func validateAnnounceIntervals(announces []scheduler.AnnounceSetting) error {
	for _, announce := range announces {
		if !announce.Status {
			continue
		}
		if err := scheduler.ValidateInterval(scheduler.SecondType, announce.Interval); err != nil {
			return err
		}
	}

	return nil
}

// processJobs 处理定时任务
func processJobs(game *dst.Game, roomID int, roomSetting models.RoomSetting) {
	// 备份 //
	backupNames := scheduler.GetJobsByType(roomID, "Backup")
	type BackupSetting struct {
		Time string `json:"time"`
	}
	var backupSettings []BackupSetting
	if err := json.Unmarshal([]byte(roomSetting.BackupSetting), &backupSettings); err != nil {
		logger.Logger.Errorf("获取房间备份设置失败, err: %v", err)
	} else {
		if roomSetting.BackupEnable {
			// 任务名按序号生成，必须按序号更新，不能依赖 GetJobsByType 的返回顺序
			keep := make(map[string]struct{}, len(backupSettings))
			for i, s := range backupSettings {
				jobName := fmt.Sprintf("%d-%d-Backup", roomID, i)
				keep[jobName] = struct{}{}
				err := scheduler.UpdateJob(&scheduler.JobConfig{
					Name:     jobName,
					Func:     scheduler.Backup,
					Args:     []any{game},
					TimeType: scheduler.DayType,
					Interval: 0,
					DayAt:    s.Time,
				})
				if err != nil {
					logger.Logger.Errorf("备份定时任务处理失败, err: %v", err)
				}
			}
			// 删除新设置之外的旧任务，避免缩减备份时按 map 顺序删掉刚更新的任务
			for _, jobName := range backupNames {
				if _, ok := keep[jobName]; !ok {
					scheduler.DeleteJob(jobName)
				}
			}
		} else {
			// 删除所有备份任务
			for _, jobName := range backupNames {
				scheduler.DeleteJob(jobName)
			}
		}
	}

	// 备份清理 //
	if roomSetting.BackupCleanEnable {
		err := scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     fmt.Sprintf("%d-BackupClean", roomID),
			Func:     scheduler.BackupClean,
			Args:     []any{roomID, roomSetting.BackupCleanSetting},
			TimeType: scheduler.DayType,
			Interval: 0,
			DayAt:    "05:16:27",
		})
		if err != nil {
			logger.Logger.Errorf("备份清理定时任务处理失败, err: %v", err)
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-BackupClean", roomID))
	}

	// 重启 //
	if roomSetting.RestartEnable {
		err := scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     fmt.Sprintf("%d-Restart", roomID),
			Func:     scheduler.Restart,
			Args:     []any{game},
			TimeType: scheduler.DayType,
			Interval: 0,
			DayAt:    roomSetting.RestartSetting,
		})
		if err != nil {
			logger.Logger.Errorf("自动重启定时任务处理失败, err: %v", err)
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-Restart", roomID))
	}
	// 重置 //
	if roomSetting.ResetEnable {
		var resetSetting struct {
			Force bool   `json:"force"`
			Time  string `json:"time"`
			Days  int    `json:"days"`
		}
		if err := json.Unmarshal([]byte(roomSetting.ResetSetting), &resetSetting); err != nil {
			logger.Logger.Errorf("获取自动重置设置失败, err: %v", err)
		} else {
			err := scheduler.UpdateJob(&scheduler.JobConfig{
				Name:     fmt.Sprintf("%d-Reset", roomID),
				Func:     scheduler.Reset,
				Args:     []any{game, roomID, resetSetting.Force, resetSetting.Days},
				TimeType: scheduler.DayType,
				Interval: 0,
				DayAt:    resetSetting.Time,
			})
			if err != nil {
				logger.Logger.Errorf("自动重置定时任务处理失败, err: %v", err)
			}
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-Reset", roomID))
	}
	// 自动开启关闭游戏
	if roomSetting.ScheduledStartStopEnable {
		type ScheduledStartStopSetting struct {
			Start string `json:"start"`
			Stop  string `json:"stop"`
		}
		var scheduledStartStopSetting ScheduledStartStopSetting
		if err := json.Unmarshal([]byte(roomSetting.ScheduledStartStopSetting), &scheduledStartStopSetting); err != nil {
			logger.Logger.Errorf("获取自动开启关闭游戏设置失败, err: %v", err)
		} else {
			err := scheduler.UpdateJob(&scheduler.JobConfig{
				Name:     fmt.Sprintf("%d-ScheduledStart", roomID),
				Func:     scheduler.ScheduledStart,
				Args:     []any{game},
				TimeType: scheduler.DayType,
				Interval: 0,
				DayAt:    scheduledStartStopSetting.Start,
			})
			if err != nil {
				logger.Logger.Errorf("自动开启游戏任务处理失败, err: %v", err)
			}
			err = scheduler.UpdateJob(&scheduler.JobConfig{
				Name:     fmt.Sprintf("%d-ScheduledStop", roomID),
				Func:     scheduler.ScheduledStop,
				Args:     []any{game},
				TimeType: scheduler.DayType,
				Interval: 0,
				DayAt:    scheduledStartStopSetting.Stop,
			})
			if err != nil {
				logger.Logger.Errorf("自动关闭游戏任务处理失败, err: %v", err)
			}
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-ScheduledStart", roomID))
		scheduler.DeleteJob(fmt.Sprintf("%d-ScheduledStop", roomID))
	}
	// 自动保活 //
	if roomSetting.KeepaliveEnable {
		err := scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     fmt.Sprintf("%d-Keepalive", roomID),
			Func:     scheduler.Keepalive,
			Args:     []any{game, roomID},
			TimeType: scheduler.MinuteType,
			Interval: roomSetting.KeepaliveSetting,
			DayAt:    "",
		})
		if err != nil {
			logger.Logger.Errorf("自动保活定时任务处理失败, err: %v", err)
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-Keepalive", roomID))
	}
	// 房间资源监控 //
	if roomSetting.RoomMetricsEnable {
		err := scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     fmt.Sprintf("%d-RoomMetrics", roomID),
			Func:     scheduler.RoomMetricsGet,
			Args:     []any{game, roomID},
			TimeType: scheduler.MinuteType,
			Interval: scheduler.RoomMetricsGetInInterval,
			DayAt:    "",
		})
		if err != nil {
			logger.Logger.Errorf("房间资源监控定时任务处理失败, err: %v", err)
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-RoomMetrics", roomID))
	}
	// 玩家更新模组
	if roomSetting.PlayerUpdateModEnable {
		errFalse := scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     fmt.Sprintf("%d-PlayerUpdateModFalse", roomID),
			Func:     scheduler.PlayerUpdateMod,
			Args:     []any{game, false},
			TimeType: scheduler.SecondType,
			Interval: scheduler.PlayerUpdateModFalseInterval,
			DayAt:    "",
		})
		if errFalse != nil {
			logger.Logger.Errorf("玩家更新模组定时任务处理失败, err: %v", errFalse)
		}
		errTrue := scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     fmt.Sprintf("%d-PlayerUpdateModTrue", roomID),
			Func:     scheduler.PlayerUpdateMod,
			Args:     []any{game, true},
			TimeType: scheduler.SecondType,
			Interval: scheduler.PlayerUpdateModTrueInterval,
			DayAt:    "",
		})
		if errTrue != nil {
			logger.Logger.Errorf("玩家更新模组定时任务处理失败, err: %v", errTrue)
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-PlayerUpdateModFalse", roomID))
		scheduler.DeleteJob(fmt.Sprintf("%d-PlayerUpdateModTrue", roomID))
	}
}

// worldIDByName 世界名到世界ID的映射，世界名是房间内世界的唯一标识
func worldIDByName(worlds []models.World) map[string]int {
	worldIDs := make(map[string]int, len(worlds))
	for _, world := range worlds {
		worldIDs[world.WorldName] = world.ID
	}
	return worldIDs
}

// deleteWorldMetrics 删除世界监控数据
func deleteWorldMetrics(worldIDMaps ...map[string]int) {
	cache.WorldMetricsMutex.Lock()
	defer cache.WorldMetricsMutex.Unlock()

	for _, worldIDMap := range worldIDMaps {
		for _, worldID := range worldIDMap {
			delete(cache.WorldMetrics, worldID)
		}
	}
}

// migrateWorldMetrics 房间保存时按世界名迁移世界监控数据
// 房间保存会删除并重建世界（世界ID发生变化），直接按ID清理会丢失已有的监控历史
func migrateWorldMetrics(oldWorldIDMap, newWorldIDMap map[string]int) {
	cache.WorldMetricsMutex.Lock()
	defer cache.WorldMetricsMutex.Unlock()

	// 先取出旧世界的数据，避免新旧世界ID交叉时互相覆盖
	oldMetrics := make(map[string][]cache.WorldMetricsData, len(oldWorldIDMap))
	for worldName, oldWorldID := range oldWorldIDMap {
		if metrics, ok := cache.WorldMetrics[oldWorldID]; ok {
			oldMetrics[worldName] = metrics
		}
	}

	// 清理该房间全部旧世界的数据
	for _, oldWorldID := range oldWorldIDMap {
		delete(cache.WorldMetrics, oldWorldID)
	}

	// 按世界名写回到新的世界ID，已删除的世界不再写入
	for worldName, newWorldID := range newWorldIDMap {
		if metrics, ok := oldMetrics[worldName]; ok {
			cache.WorldMetrics[newWorldID] = metrics
		}
	}
}
