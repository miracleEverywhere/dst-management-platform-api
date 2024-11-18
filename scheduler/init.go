package scheduler

import (
	"dst-management-platform-api/utils"
	"github.com/go-co-op/gocron"
	"time"
)

var Scheduler = gocron.NewScheduler(time.Local)

// InitTasks 初始化定时任务
func InitTasks() {
	// 获取当前玩家
	_, _ = Scheduler.Every(30).Seconds().Do(setPlayer2DB)
	utils.Logger.Info("玩家列表定时任务已配置")

	//初始化定时通知
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		return
	}
	for _, announce := range config.AutoAnnounce {
		if announce.Enable {
			_, _ = Scheduler.Every(announce.Frequency).Seconds().Do(execAnnounce, announce.Content)
			utils.Logger.Info("定时通知定时任务已配置", "name", announce.Name)
		}
	}

	// 自动更新
	if config.AutoUpdate.Enable {
		_, _ = Scheduler.Every(1).Day().At(updateTimeFix(config.AutoUpdate.Time)).Do(checkUpdate)
		utils.Logger.Info("自动更新定时任务已配置")
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
