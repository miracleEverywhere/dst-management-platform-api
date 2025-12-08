package room

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	roomDao          *dao.RoomDAO
	userDao          *dao.UserDAO
	worldDao         *dao.WorldDAO
	roomSettingDao   *dao.RoomSettingDAO
	globalSettingDao *dao.GlobalSettingDAO
}

func NewHandler(userDao *dao.UserDAO, roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO, globalSettingDao *dao.GlobalSettingDAO) *Handler {
	return &Handler{
		roomDao:          roomDao,
		userDao:          userDao,
		worldDao:         worldDao,
		roomSettingDao:   roomSettingDao,
		globalSettingDao: globalSettingDao,
	}
}

type Partition struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"pageSize" form:"pageSize"`
}

type XRoomWorld struct {
	models.Room
	Worlds  []models.World `json:"worlds"`
	Players []db.Players   `json:"players"`
}

type XRoomTotalInfo struct {
	RoomData        models.Room        `json:"roomData"`
	WorldData       []models.World     `json:"worldData"`
	RoomSettingData models.RoomSetting `json:"roomSettingData"`
}

func (h *Handler) hasPermission(c *gin.Context) (bool, error) {
	role, _ := c.Get("role")
	username, _ := c.Get("username")
	var (
		has    bool
		err    error
		dbUser *models.User
	)

	// 管理员直接返回true
	if role.(string) == "admin" {
		has = true
	} else {
		dbUser, err = h.userDao.GetUserByUsername(username.(string))
		if err != nil {
			return has, err
		}
		if dbUser.RoomCreation {
			has = true
		}
	}

	return has, err
}

// 处理定时任务
func processJobs(game *dst.Game, roomID int, roomSetting models.RoomSetting) {
	// 备份 //
	backupNames := scheduler.GetJobs(roomID, "Backup")
	type BackupSetting struct {
		Time string `json:"time"`
	}
	var backupSettings []BackupSetting
	if err := json.Unmarshal([]byte(roomSetting.BackupSetting), &backupSettings); err != nil {
		logger.Logger.Error("获取房间备份设置失败", "err", err)
	}
	if roomSetting.BackupEnable {
		if len(backupSettings) >= len(backupNames) {
			// 新设置长度大于旧设置，直接更新
			for i, s := range backupSettings {
				err := scheduler.UpdateJob(&scheduler.JobConfig{
					Name:     fmt.Sprintf("%d-%d-Backup", roomID, i),
					Func:     scheduler.Backup,
					Args:     []interface{}{game},
					TimeType: "day",
					Interval: 0,
					DayAt:    s.Time,
				})
				if err != nil {
					logger.Logger.Error("备份定时任务处理失败", "err", err)
				}
			}
		} else {
			// 新设置长度小于旧设置，超出的删除
			for i, jobName := range backupNames {
				if i >= len(backupSettings) {
					scheduler.DeleteJob(jobName)
				} else {
					err := scheduler.UpdateJob(&scheduler.JobConfig{
						Name:     fmt.Sprintf("%d-%d-Backup", roomID, i),
						Func:     scheduler.Backup,
						Args:     []interface{}{game},
						TimeType: "day",
						Interval: 0,
						DayAt:    backupSettings[i].Time,
					})
					if err != nil {
						logger.Logger.Error("备份定时任务处理失败", "err", err)
					}
				}
			}
		}
	} else {
		// 删除所有备份任务
		for _, jobName := range backupNames {
			scheduler.DeleteJob(jobName)
		}
	}
	// 备份清理 //
	if roomSetting.BackupCleanEnable {
		err := scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     fmt.Sprintf("%d-BackupClean", roomID),
			Func:     scheduler.BackupClean,
			Args:     []interface{}{roomID, roomSetting.BackupCleanSetting},
			TimeType: "day",
			Interval: 0,
			DayAt:    "05:16:27",
		})
		if err != nil {
			logger.Logger.Error("备份清理定时任务处理失败", "err", err)
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-BackupClean", roomID))
	}
	// 重启 //
	if roomSetting.RestartEnable {
		err := scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     fmt.Sprintf("%d-Restart", roomID),
			Func:     scheduler.Restart,
			Args:     []interface{}{game},
			TimeType: "day",
			Interval: 0,
			DayAt:    roomSetting.RestartSetting,
		})
		if err != nil {
			logger.Logger.Error("重启定时任务处理失败", "err", err)
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-Restart", roomID))
	}
	// 自动开启关闭游戏
	if roomSetting.ScheduledStartStopEnable {
		type ScheduledStartStopSetting struct {
			Start string `json:"start"`
			Stop  string `json:"stop"`
		}
		var scheduledStartStopSetting ScheduledStartStopSetting
		if err := json.Unmarshal([]byte(roomSetting.ScheduledStartStopSetting), &scheduledStartStopSetting); err != nil {
			logger.Logger.Error("获取自动开启关闭游戏设置失败", "err", err)
		}
		err := scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     fmt.Sprintf("%d-ScheduledStart", roomID),
			Func:     scheduler.ScheduledStart,
			Args:     []interface{}{game},
			TimeType: "day",
			Interval: 0,
			DayAt:    scheduledStartStopSetting.Start,
		})
		if err != nil {
			logger.Logger.Error("自动开启游戏任务处理失败", "err", err)
		}
		err = scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     fmt.Sprintf("%d-ScheduledStop", roomID),
			Func:     scheduler.ScheduledStop,
			Args:     []interface{}{game},
			TimeType: "day",
			Interval: 0,
			DayAt:    scheduledStartStopSetting.Stop,
		})
		if err != nil {
			logger.Logger.Error("自动关闭游戏任务处理失败", "err", err)
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
			Args:     []interface{}{game, roomID},
			TimeType: "minute",
			Interval: roomSetting.KeepaliveSetting,
			DayAt:    "",
		})
		if err != nil {
			logger.Logger.Error("自动保活定时任务处理失败", "err", err)
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-Keepalive", roomID))
	}
}
