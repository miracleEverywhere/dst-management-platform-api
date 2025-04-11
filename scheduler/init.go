package scheduler

import (
	"dst-management-platform-api/utils"
	"github.com/go-co-op/gocron"
	"time"
)

var Scheduler = gocron.NewScheduler(time.Local)

// InitTasks 初始化定时任务
func InitTasks() {
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		return
	}

	// 获取当前玩家
	_, _ = Scheduler.Every(config.SysSetting.SchedulerSetting.PlayerGetFrequency).Seconds().Do(setPlayer2DB, config)
	utils.Logger.Info("玩家列表定时任务已配置")

	// 维护UID字典
	if !config.SysSetting.SchedulerSetting.UIDMaintain.Disable {
		_, _ = Scheduler.Every(config.SysSetting.SchedulerSetting.UIDMaintain.Frequency).Minute().Do(maintainUidMap)
		utils.Logger.Info("UID字典定时维护任务已配置")
	}

	// 系统监控
	if !config.SysSetting.SchedulerSetting.SysMetricsGet.Disable {
		_, _ = Scheduler.Every(30).Seconds().Do(getSysMetrics)
		utils.Logger.Info("系统监控定时任务已配置")
	}

	// 定时通知
	for _, announce := range config.SysSetting.AutoAnnounce {
		if announce.Enable {
			_, _ = Scheduler.Every(announce.Frequency).Seconds().Do(execAnnounce, announce.Content)
			utils.Logger.Info("定时通知定时任务已配置", "name", announce.Name)
		}
	}

	// 自动更新
	if config.SysSetting.AutoUpdate.Enable {
		_, _ = Scheduler.Every(1).Day().At(updateTimeFix(config.SysSetting.AutoUpdate.Time)).Do(checkUpdate, config)
		utils.Logger.Info("自动更新定时任务已配置")
	}

	// 自动重启
	if config.SysSetting.AutoRestart.Enable {
		_, _ = Scheduler.Every(1).Day().At(updateTimeFix(config.SysSetting.AutoRestart.Time)).Do(doRestart, config)
		utils.Logger.Info("自动重启定时任务已配置")
	}

	// 自动备份
	if config.AutoBackup.Enable {
		_, _ = Scheduler.Every(1).Day().At(config.AutoBackup.Time).Do(doBackup)
		utils.Logger.Info("自动备份定时任务已配置")
	}

	// 自动保活
	if config.Keepalive.Enable {
		_, _ = Scheduler.Every(config.Keepalive.Frequency).Minute().Do(doKeepalive)
		utils.Logger.Info("自动保活定时任务已配置")
	}

}
