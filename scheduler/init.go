package scheduler

import (
	"dst-management-platform-api/utils"
	"fmt"
	"github.com/go-co-op/gocron"
	"strings"
	"time"
)

var Scheduler = gocron.NewScheduler(time.Local)

// InitTasks 初始化定时任务
func InitTasks() {
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		panic("致命错误：定时任务初始化失败")
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
		_, _ = Scheduler.Every(30).Seconds().Do(getSysMetrics, config.SchedulerSetting.SysMetricsGet.MaxSaveHour)
		utils.Logger.Info("系统监控定时任务已配置")
	}

	// 自动更新
	if config.SchedulerSetting.AutoUpdate.Enable {
		_, _ = Scheduler.Every(1).Day().At(updateTimeFix(config.SchedulerSetting.AutoUpdate.Time)).Do(checkUpdate, config)
		utils.Logger.Info("自动更新定时任务已配置")
	}

	// 玩家更新模组
	if !config.SchedulerSetting.PlayerUpdateMod.Disable {
		for _, cluster := range config.Clusters {
			// 关闭的集群直接跳过添加定时任务
			if !cluster.ClusterSetting.Status {
				continue
			}
			_, _ = Scheduler.Every(config.SchedulerSetting.PlayerUpdateMod.Frequency).Minute().Do(modUpdate, cluster, false)
			_, _ = Scheduler.Every(60).Seconds().Do(modUpdate, cluster, true)
		}
		utils.Logger.Info("玩家更新模组定时任务已配置")
	}

	/* ** ========== SysSetting 影响集群 ========== ** */
	for _, cluster := range config.Clusters {
		// 关闭的集群直接跳过添加定时任务
		if !cluster.ClusterSetting.Status {
			continue
		}

		// 定时通知
		for _, announce := range cluster.SysSetting.AutoAnnounce {
			if announce.Enable {
				_, _ = Scheduler.Every(announce.Frequency).Seconds().Do(doAnnounce, announce.Content, cluster)
				utils.Logger.Info(fmt.Sprintf("[%s(%s)]-[%s]定时通知定时任务已配置", cluster.ClusterSetting.ClusterName, cluster.ClusterSetting.ClusterDisplayName, announce.Name))
			}
		}

		// 自动重启
		if cluster.SysSetting.AutoRestart.Enable {
			if cluster.SysSetting.AutoRestart.Enable {
				_, _ = Scheduler.Every(1).Day().At(updateTimeFix(cluster.SysSetting.AutoRestart.Time)).Do(doRestart, cluster)
				utils.Logger.Info(fmt.Sprintf("[%s(%s)]自动重启定时任务已配置", cluster.ClusterSetting.ClusterName, cluster.ClusterSetting.ClusterDisplayName))
			}
		}

		// 自动备份
		if cluster.SysSetting.AutoBackup.Enable {
			times := strings.Split(cluster.SysSetting.AutoBackup.Time, ",")
			for _, t := range times {
				if t != "" {
					_, _ = Scheduler.Every(1).Day().At(t).Do(doBackup, cluster)
					utils.Logger.Info(fmt.Sprintf("[%s(%s)]自动备份定时任务已配置，备份时间：%s", cluster.ClusterSetting.ClusterName, cluster.ClusterSetting.ClusterDisplayName, t))
				}
			}

		}

		// 备份清理
		if cluster.SysSetting.BackupClean.Enable {
			_, _ = Scheduler.Every(1).Day().At("16:43:41").Do(doBackupClean, cluster)
			utils.Logger.Info(fmt.Sprintf("[%s(%s)]备清理定时任务已配置", cluster.ClusterSetting.ClusterName, cluster.ClusterSetting.ClusterDisplayName))
		}

		// 自动保活
		if cluster.SysSetting.Keepalive.Enable {
			_, _ = Scheduler.Every(cluster.SysSetting.Keepalive.Frequency).Minute().Do(doKeepalive, cluster.ClusterSetting.ClusterName)
			utils.Logger.Info(fmt.Sprintf("[%s(%s)]自动保活定时任务已配置", cluster.ClusterSetting.ClusterName, cluster.ClusterSetting.ClusterDisplayName))
		}

		// 定时开启关闭服务器
		if cluster.SysSetting.ScheduledStartStop.Enable {
			_, _ = Scheduler.Every(1).Day().At(cluster.SysSetting.ScheduledStartStop.StartTime).Do(doStart, cluster)
			_, _ = Scheduler.Every(1).Day().At(cluster.SysSetting.ScheduledStartStop.StopTime).Do(doStop, cluster)
			utils.Logger.Info(fmt.Sprintf("[%s(%s)]定时开启关闭服务器已配置", cluster.ClusterSetting.ClusterName, cluster.ClusterSetting.ClusterDisplayName))
		}
	}
}
