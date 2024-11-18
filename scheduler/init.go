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

	//初始化定时通知
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		return
	}
	for _, announce := range config.AutoAnnounce {
		if announce.Enable {
			_, _ = Scheduler.Every(announce.Frequency).Seconds().Do(execAnnounce, announce.Content)
		}
	}

	// 自动更新
	if config.AutoUpdate.Enable {
		_, _ = Scheduler.Every(1).Day().At(updateTimeFix(config.AutoUpdate.Time)).Do(checkUpdate)
	}

	// 自动备份
	if config.AutoBackup.Enable {
		_, _ = Scheduler.Every(1).Day().At(config.AutoBackup.Time).Do(doBackup)
	}

	// 自动保活
	if config.Keepalive.Enable {
		_, _ = Scheduler.Every(config.Keepalive.Frequency).Minute().Do(doKeepalive)
	}

}
