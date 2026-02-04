package scheduler

import (
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"encoding/json"
	"fmt"
	"strings"
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
		Func:     OnlinePlayerGet,
		Args:     []any{globalSetting.PlayerGetFrequency, globalSetting.UIDMaintainEnable},
		TimeType: SecondType,
		Interval: globalSetting.PlayerGetFrequency,
		DayAt:    "",
	})

	// 系统监控
	Jobs = append(Jobs, JobConfig{
		Name:     "systemMetricsGet",
		Func:     SystemMetricsGet,
		Args:     []any{globalSetting.SysMetricsSetting},
		TimeType: MinuteType,
		Interval: 1,
		DayAt:    "",
	})

	// 游戏更新
	Jobs = append(Jobs, JobConfig{
		Name:     "gameUpdate",
		Func:     GameUpdate,
		Args:     []any{globalSetting.AutoUpdateEnable, globalSetting.AutoUpdateRestart},
		TimeType: DayType,
		Interval: 0,
		DayAt:    globalSetting.AutoUpdateSetting,
	})

	// 公网IP获取
	Jobs = append(Jobs, JobConfig{
		Name:     "InternetIPUpdate",
		Func:     InternetIPUpdate,
		Args:     nil,
		TimeType: HourType,
		Interval: 6,
		DayAt:    "",
	})

	// 清理临时模组
	Jobs = append(Jobs, JobConfig{
		Name:     "ModDownloadClean",
		Func:     ModDownloadClean,
		Args:     nil,
		TimeType: MinuteType,
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
		// 未激活的房间不添加定时任务
		if !r.Status {
			continue
		}

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
					Args:     []any{game},
					TimeType: DayType,
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
				Args:     []any{room.ID, roomSetting.BackupCleanSetting},
				TimeType: DayType,
				Interval: 0,
				DayAt:    "05:16:27",
			})
		}
		// 重启 "06:30:00"
		if roomSetting.RestartEnable {
			Jobs = append(Jobs, JobConfig{
				Name:     fmt.Sprintf("%d-Restart", room.ID),
				Func:     Restart,
				Args:     []any{game},
				TimeType: DayType,
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
				Args:     []any{game},
				TimeType: DayType,
				Interval: 0,
				DayAt:    scheduledStartStopSetting.Start,
			})
			Jobs = append(Jobs, JobConfig{
				Name:     fmt.Sprintf("%d-ScheduledStop", room.ID),
				Func:     ScheduledStop,
				Args:     []any{game},
				TimeType: DayType,
				Interval: 0,
				DayAt:    scheduledStartStopSetting.Stop,
			})
		}
		// 自动保活
		if roomSetting.KeepaliveEnable {
			Jobs = append(Jobs, JobConfig{
				Name:     fmt.Sprintf("%d-Keepalive", room.ID),
				Func:     Keepalive,
				Args:     []any{game, room.ID},
				TimeType: MinuteType,
				Interval: roomSetting.KeepaliveSetting,
				DayAt:    "",
			})
		}
		// 定时通知 [{id: '', content: '', interval: 0, status: false}]
		var announces []AnnounceSetting
		if err = json.Unmarshal([]byte(roomSetting.AnnounceSetting), &announces); err != nil {
			logger.Logger.Error("获取定时通知设置失败", "err", err)
			continue
		}
		for _, announce := range announces {
			if announce.Status {
				// 注意，-为分隔符，需要删除uuid中的-
				Jobs = append(Jobs, JobConfig{
					Name:     fmt.Sprintf("%d-%s-Announce", room.ID, strings.ReplaceAll(announce.ID, "-", "")),
					Func:     Announce,
					Args:     []any{game, announce.Content},
					TimeType: SecondType,
					Interval: announce.Interval,
					DayAt:    "",
				})
			}
		}
	}
}
