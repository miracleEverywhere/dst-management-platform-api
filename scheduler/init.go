package scheduler

import (
	"dst-management-platform-api/utils"
	"github.com/go-co-op/gocron"
	"time"
)

var Scheduler = gocron.NewScheduler(time.UTC)

// InitTasks 初始化定时任务
func InitTasks() {
	// 获取当前玩家
	_, _ = Scheduler.Every(15).Seconds().Do(setPlayer2DB)

	//初始化定时通知
	config, _ := utils.ReadConfig()
	for _, announce := range config.AutoAnnounce {
		if announce.Enable {
			_, _ = Scheduler.Every(announce.Frequency).Seconds().Do(execAnnounce, announce.Content)
		}
	}

	// 自动更新
	if config.AutoUpdate.Enable {
		_, _ = Scheduler.Every(1).Day().At(config.AutoUpdate.Time).Do(checkUpdate)
	}

	// 自动备份
	if config.AutoBackup.Enable {
		_, _ = Scheduler.Every(1).Day().At(config.AutoBackup.Time).Do(doBackup)
	}
}
