package scheduler

import (
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"time"
)

func Backup(game *dst.Game) {
	logger.Logger.Info("执行自动备份任务")
	err := game.Backup()
	if err != nil {
		logger.Logger.Error("备份失败", "err", err)
	}
	logger.Logger.Info("备份任务执行成功")
}

func BackupClean(roomID int, days int) {
	backupPath := fmt.Sprintf("%s/backup/%d", utils.DmpFiles, roomID)
	count, err := utils.RemoveFilesOlderThan(backupPath, days)
	if err != nil {
		logger.Logger.Error("清理备份文件失败", "err", err)
	}
	logger.Logger.Info(fmt.Sprintf("清理备份文件成功，共计清理备份文件%d个", count))
}

func Restart(game *dst.Game) {
	logger.Logger.Info("执行自动重启任务")
	go func() {
		_ = game.SystemMsg("自动重启任务触发：将在1分钟后重启服务器，在线玩家请在5分钟后重连")
		_ = game.SystemMsg("Automatic restart task triggered: The server will restart in 1 minute. Online players, please reconnect after 5 minutes")
		time.Sleep(60 * time.Second)
		err := game.StopAllWorld()
		if err != nil {
			logger.Logger.Warn("关闭游戏失败", "err", err)
		}
		err = game.StartAllWorld()
		if err != nil {
			logger.Logger.Error("启动游戏失败", "err", err)
			logger.Logger.Error("自动重启任务执行失败")
		} else {
			logger.Logger.Info("自动重启任务执行成功")
		}
	}()
}

func ScheduledStart(game *dst.Game) {
	logger.Logger.Info("执行自动开启游戏")
	err := game.StartAllWorld()
	if err != nil {
		logger.Logger.Error("开启游戏失败", "err", err)
	}
	logger.Logger.Info("自动开启游戏执行成功")
}

func ScheduledStop(game *dst.Game) {
	logger.Logger.Info("执行自动关闭游戏")
	go func() {
		_ = game.SystemMsg("自动关机任务触发：将在1分钟后关闭服务器")
		_ = game.SystemMsg("Automatic shutdown task triggered: The server will restart in 1 minute")
		time.Sleep(60 * time.Second)
		err := game.StopAllWorld()
		if err != nil {
			logger.Logger.Warn("关闭游戏失败", "err", err)
		}
		logger.Logger.Info("自动关闭游戏执行成功")
	}()
}

func Keepalive(game *dst.Game, roomID int) {
	worlds, err := DBHandler.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		logger.Logger.Error("获取世界信息失败，自动保活任务终止", "err", err)
		return
	}

	var (
		updatedWorlds []models.World
		needUpdateDB  bool
	)

	for _, world := range *worlds {
		lastTime, err := game.GetLastAliveTime(world.ID)
		if err != nil {
			logger.Logger.Error("获取日志信息失败，无法判断，跳过", "err", err, "world", world.ID)
			continue
		}
		if lastTime == world.LastAliveTime {
			logger.Logger.Error("发现世界运行异常，即将执行重启操作", "world", world.ID)
			_ = game.StopWorld(world.ID)
			_ = game.StartWorld(world.ID)
		} else {
			world.LastAliveTime = lastTime
			updatedWorlds = append(updatedWorlds, world)
			needUpdateDB = true
		}
	}

	if needUpdateDB {
		err = DBHandler.worldDao.UpdateWorlds(&updatedWorlds)
		if err != nil {
			logger.Logger.Error("更新数据失败", "err", err)
		}
	}
}

func Announce(game *dst.Game, content string) {
	err := game.Announce(content)
	if err != nil {
		logger.Logger.Error("定时通知失败", "err", err)
	}
}
