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
		Args:     []interface{}{globalSetting.PlayerGetFrequency},
		TimeType: "second",
		Interval: globalSetting.PlayerGetFrequency,
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
	}
}
