package scheduler

import (
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"encoding/json"
	"fmt"
)

var Jobs []JobConfig

func initJobs() {
	var globalSetting models.GlobalSetting
	err := DBHandler.globalSettingDao.GetGlobalSetting(&globalSetting)
	if err != nil {
		logger.Logger.Error("初始化定时任务失败", "err", err)
		panic("初始化定时任务失败")
	}

	// 全局定时任务
	// players online
	Jobs = append(Jobs, JobConfig{
		Name:     "onlinePlayerGet",
		Func:     onlinePlayerGet,
		Args:     []interface{}{globalSetting.PlayerGetFrequency, globalSetting.UIDMaintainEnable},
		TimeType: "second",
		Interval: globalSetting.PlayerGetFrequency,
		DayAt:    "",
	})

	// 系统监控
	Jobs = append(Jobs, JobConfig{
		Name:     "systemMetricsGet",
		Func:     systemMetricsGet,
		Args:     []interface{}{globalSetting.SysMetricsSetting},
		TimeType: "minute",
		Interval: 1,
		DayAt:    "",
	})

	// 房间定时任务
	roomBasic, err := DBHandler.roomDao.GetRoomBasic()
	if err != nil {
		logger.Logger.Error("获取房间失败", "err", err)
		return
	}

	for _, r := range *roomBasic {
		room, worlds, roomSetting, err := fetchGameInfo(r.RoomID)
		if err != nil {
			logger.Logger.Error("获取房间设置失败", "err", err)
			continue
		}
		game := dst.NewGameController(room, worlds, roomSetting, "zh")

		// 备份 [{"time": "06:00:00"}, ...]
		type BackupSetting struct {
			Time string `json:"time"`
		}
		if roomSetting.BackupEnable {
			var backupSettings []BackupSetting
			if err := json.Unmarshal([]byte(roomSetting.BackupSetting), &backupSettings); err != nil {
				logger.Logger.Error("获取房间备份设置失败", "err", err)
				continue
			}
			for i, backupSetting := range backupSettings {
				// 房间id-time_index-Backup
				Jobs = append(Jobs, JobConfig{
					Name:     fmt.Sprintf("%d-%d-Backup", room.ID, i),
					Func:     Backup,
					Args:     []interface{}{game},
					TimeType: "day",
					Interval: 0,
					DayAt:    backupSetting.Time,
				})
			}
		}
		// 备份清理 30
		if roomSetting.BackupCleanEnable {
			Jobs = append(Jobs, JobConfig{
				Name:     fmt.Sprintf("%d-BackupClean", room.ID),
				Func:     BackupClean,
				Args:     []interface{}{room.ID, roomSetting.BackupCleanSetting},
				TimeType: "day",
				Interval: 0,
				DayAt:    "05:16:27",
			})
		}
		// 重启 "06:30:00"
		if roomSetting.RestartEnable {
			Jobs = append(Jobs, JobConfig{
				Name:     fmt.Sprintf("%d-Restart", room.ID),
				Func:     Restart,
				Args:     []interface{}{game},
				TimeType: "day",
				Interval: 0,
				DayAt:    roomSetting.RestartSetting,
			})
		}
		// 自动开启关闭游戏 {"start":"07:00:00","stop":"01:00:00"}
		if roomSetting.ScheduledStartStopEnable {
			type ScheduledStartStopSetting struct {
				Start string `json:"start"`
				Stop  string `json:"stop"`
			}
			var scheduledStartStopSetting ScheduledStartStopSetting
			if err := json.Unmarshal([]byte(roomSetting.ScheduledStartStopSetting), &scheduledStartStopSetting); err != nil {
				logger.Logger.Error("获取自动开启关闭游戏设置失败", "err", err)
				continue
			}
			Jobs = append(Jobs, JobConfig{
				Name:     fmt.Sprintf("%d-ScheduledStart", room.ID),
				Func:     ScheduledStart,
				Args:     []interface{}{game},
				TimeType: "day",
				Interval: 0,
				DayAt:    scheduledStartStopSetting.Start,
			})
			Jobs = append(Jobs, JobConfig{
				Name:     fmt.Sprintf("%d-ScheduledStop", room.ID),
				Func:     ScheduledStop,
				Args:     []interface{}{game},
				TimeType: "day",
				Interval: 0,
				DayAt:    scheduledStartStopSetting.Stop,
			})
		}
		// 自动保活
		if roomSetting.KeepaliveEnable {
			Jobs = append(Jobs, JobConfig{
				Name:     fmt.Sprintf("%d-Keepalive", room.ID),
				Func:     Keepalive,
				Args:     []interface{}{game, room.ID},
				TimeType: "minute",
				Interval: roomSetting.KeepaliveSetting,
				DayAt:    "",
			})
		}
	}
}
