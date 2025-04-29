package scheduler

import (
	"dst-management-platform-api/utils"
	"fmt"
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

	/* ** ========== SchedulerSetting 影响全局 ========== ** */
	// 获取当前玩家
	_, _ = Scheduler.Every(config.SchedulerSetting.PlayerGetFrequency).Seconds().Do(getPlayers, config)
	utils.Logger.Info("玩家列表定时任务已配置")

	// 维护UID字典
	if !config.SchedulerSetting.UIDMaintain.Disable {
		_, _ = Scheduler.Every(config.SchedulerSetting.UIDMaintain.Frequency).Minute().Do(maintainUidMap, config)
		utils.Logger.Info("UID字典定时维护任务已配置")
	}

	// 系统监控
	if !config.SchedulerSetting.SysMetricsGet.Disable {
		_, _ = Scheduler.Every(30).Seconds().Do(getSysMetrics)
		utils.Logger.Info("系统监控定时任务已配置")
	}

	// 自动更新
	if config.SchedulerSetting.AutoUpdate.Enable {
		_, _ = Scheduler.Every(1).Day().At(updateTimeFix(config.SchedulerSetting.AutoUpdate.Time)).Do(checkUpdate, config)
		utils.Logger.Info("自动更新定时任务已配置")
	}

	/* ** ========== SysSetting 影响集群 ========== ** */
	// 定时通知
	for _, cluster := range config.Clusters {
		for _, announce := range cluster.SysSetting.AutoAnnounce {
			if announce.Enable {
				_, _ = Scheduler.Every(announce.Frequency).Seconds().Do(execAnnounce, announce.Content, cluster)
				utils.Logger.Info(fmt.Sprintf("[%s]-[%s]定时通知定时任务已配置", cluster.ClusterSetting.ClusterName, announce.Name))
			}
		}
	}

	// 自动重启
	for _, cluster := range config.Clusters {
		if cluster.SysSetting.AutoRestart.Enable {
			if cluster.SysSetting.AutoRestart.Enable {
				_, _ = Scheduler.Every(1).Day().At(updateTimeFix(cluster.SysSetting.AutoRestart.Time)).Do(doRestart, cluster)
				utils.Logger.Info(fmt.Sprintf("[%s]自动重启定时任务已配置", cluster.ClusterSetting.ClusterName))
			}
		}
	}

	// 自动备份
	for _, cluster := range config.Clusters {
		if cluster.SysSetting.AutoBackup.Enable {
			_, _ = Scheduler.Every(1).Day().At(cluster.SysSetting.AutoBackup.Time).Do(doBackup, cluster)
			utils.Logger.Info(fmt.Sprintf("[%s]自动备份定时任务已配置", cluster.ClusterSetting.ClusterName))
		}

	}

	// 自动保活
	for _, cluster := range config.Clusters {
		if cluster.SysSetting.Keepalive.Enable {
			_, _ = Scheduler.Every(cluster.SysSetting.Keepalive.Frequency).Minute().Do(doKeepalive, cluster)
		}
	}
}
