package scheduler

import (
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
)

func Backup(game *dst.Game) {
	logger.Logger.Info("开始备份任务")
	err := game.Backup()
	if err != nil {
		logger.Logger.Error("备份失败", "err", err)
	}
	logger.Logger.Info("备份任务执行成功")
}
