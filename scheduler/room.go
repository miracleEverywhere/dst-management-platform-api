package scheduler

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"fmt"
	"regexp"
	"strings"
	"time"
)

var rePlayerUpdateMod = regexp.MustCompile(`([A-Z0-9]{7})-LKGX`)

func Backup(game *dst.Game) {
	logger.Logger.Info("[定时任务]：执行自动备份任务")
	err := game.Backup()
	if err != nil {
		logger.Logger.Errorf("备份失败, err: %v", err)
	}
	logger.Logger.Info("[定时任务]：备份任务执行成功")
}

func BackupClean(roomID int, days int) {
	backupPath := fmt.Sprintf("%s/backup/%d", utils.DmpFiles, roomID)
	count, err := utils.RemoveFilesOlderThan(backupPath, days)
	if err != nil {
		logger.Logger.Errorf("清理备份文件失败, err: %v", err)
	}
	logger.Logger.Infof("[定时任务]：清理备份文件成功，共计清理备份文件%d个", count)
}

func Restart(game *dst.Game) {
	logger.Logger.Info("[定时任务]：执行自动重启任务")
	go func() {
		_ = game.SystemMsg("自动重启任务触发：将在1分钟后重启服务器，在线玩家请在5分钟后重连")
		_ = game.SystemMsg("Automatic restart task triggered: The server will restart in 1 minute. Online players, please reconnect after 5 minutes")
		time.Sleep(60 * time.Second)
		err := game.StopAllWorld()
		if err != nil {
			logger.Logger.Warnf("关闭游戏失败, err: %v", err)
		}
		err = game.StartAllWorld()
		if err != nil {
			logger.Logger.Errorf("启动游戏失败, err: %v", err)
			logger.Logger.Error("自动重启任务执行失败")
		} else {
			logger.Logger.Info("[定时任务]：自动重启任务执行成功")
		}
	}()
}

func Reset(game *dst.Game, roomID int, force bool, days int) {
	reset := func() {
		logger.Logger.Info("[定时任务]：正在执行自动重置任务")
		err := game.Reset(false)
		if err != nil {
			logger.Logger.Warnf("游戏内重置失败，尝试执行强制重置, err: %v", err)
			err = game.Reset(true)
			if err != nil {
				logger.Logger.Errorf("重置失败, err: %v", err)
			} else {
				logger.Logger.Info("[定时任务]：自动重置(强制重置)任务执行成功")
			}
		} else {
			logger.Logger.Info("[定时任务]：自动重置(游戏内重置)任务执行成功")
		}
	}

	if force {
		// 直接进行重置
		go reset()
	} else {
		// 空闲重置
		secs := days * 24 * 60 * 60
		cache.RoomNoPlayersSecondsMutex.Lock()
		if cache.RoomNoPlayersSeconds[roomID] > secs {
			go reset()
			cache.RoomNoPlayersSeconds[roomID] = 0
		}
		cache.RoomNoPlayersSecondsMutex.Unlock()
	}
}

func ScheduledStart(game *dst.Game) {
	logger.Logger.Info("[定时任务]：执行自动开启游戏")
	err := game.StartAllWorld()
	if err != nil {
		logger.Logger.Errorf("开启游戏失败, err: %v", err)
	}
	logger.Logger.Info("[定时任务]：自动开启游戏执行成功")
}

func ScheduledStop(game *dst.Game) {
	logger.Logger.Info("[定时任务]：执行自动关闭游戏")
	go func() {
		_ = game.SystemMsg("自动关机任务触发：将在1分钟后关闭服务器")
		_ = game.SystemMsg("Automatic shutdown task triggered: The server will restart in 1 minute")
		time.Sleep(60 * time.Second)
		err := game.StopAllWorld()
		if err != nil {
			logger.Logger.Warnf("关闭游戏失败, err: %v", err)
		}
		logger.Logger.Info("[定时任务]：自动关闭游戏执行成功")
	}()
}

func Keepalive(game *dst.Game, roomID int) {
	worlds, err := DBHandler.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("获取世界信息失败，自动保活任务终止, err: %v", err)
		return
	}

	allWorlds := *worlds
	needUpdateDB := false

	for i := range allWorlds {
		lastTime, err := game.GetLastAliveTime(allWorlds[i].ID)
		if err != nil {
			logger.Logger.Errorf("获取日志信息失败，无法判断，跳过, err: %v, world: %v", err, allWorlds[i].ID)
			continue
		}
		if lastTime == allWorlds[i].LastAliveTime {
			errMsg := fmt.Sprintf("发现世界运行异常，即将执行重启操作, world: %v", allWorlds[i].ID)
			logger.Logger.Error(errMsg)
			webhook.Snd.Send(webhook.EventKeepaliveTriggered, roomID, errMsg)
			_ = game.StopWorld(allWorlds[i].ID)
			_ = game.StartWorld(allWorlds[i].ID)
		} else {
			allWorlds[i].LastAliveTime = lastTime
			needUpdateDB = true
		}
	}

	if needUpdateDB {
		err = DBHandler.worldDao.UpdateWorlds(&allWorlds)
		if err != nil {
			logger.Logger.Errorf("更新数据失败, err: %v", err)
		}
	}
}

func Announce(game *dst.Game, content string) {
	err := game.Announce(content)
	if err != nil {
		logger.Logger.Errorf("定时通知失败, err: %v", err)
	}
}

// PlayerUpdateMod check指已经发送了更新ID
func PlayerUpdateMod(game *dst.Game, check bool) {
	cache.PlayerUpdateModStatusMutex.Lock()
	defer cache.PlayerUpdateModStatusMutex.Unlock()

	if cache.PlayerUpdateModStatus[game.RoomID()] {
		return
	}

	worlds := game.Worlds()
	var (
		logContent []string
		logGot     bool
	)

	for _, world := range worlds {
		if game.WorldUpStatus(world.ID) {
			logContent = game.LogContent("chat", world.ID, 100)
			if len(logContent) > 0 {
				logGot = true
				break
			}
		}
	}

	if !logGot {
		return
	}

	var (
		updateTip string
		err       error
	)

	if check {
		// 已发送更新ID，开始处理业务
		for _, line := range logContent {
			subMatches := rePlayerUpdateMod.FindStringSubmatch(line)
			if len(subMatches) > 1 {
				if utils.VerifyUpdateModID(subMatches[1]) {
					cache.PlayerUpdateModStatus[game.RoomID()] = true
					updateTip = "模组更新命令校验成功"
					err = game.SystemMsg(updateTip)
					if err != nil {
						logger.Logger.Errorf("玩家更新模组定时任务异常，发送通知失败, err: %v", err)
					}
					time.Sleep(500 * time.Millisecond)

					updateTip = "将在1分钟后自动重启服务器并更新模组"
					err = game.SystemMsg(updateTip)
					if err != nil {
						logger.Logger.Errorf("玩家更新模组定时任务异常，发送通知失败, err: %v", err)
					}
					time.Sleep(1 * time.Minute)

					_ = game.StopAllWorld()
					time.Sleep(5 * time.Second)

					err = game.StartAllWorld()
					if err != nil {
						logger.Logger.Errorf("玩家更新模组定时任务异常，启动游戏失败, err: %v", err)
					}
					time.Sleep(10 * time.Minute)
					cache.PlayerUpdateModStatus[game.RoomID()] = false
					return
				}
			}
		}

		return
	}

	// 未发送更新ID，开始发送

	sendTip := func() error {
		updateModID := utils.GenerateUpdateModID()
		if len(updateModID) == 0 {
			return fmt.Errorf("更新ID生成异常")
		}
		updateTip = fmt.Sprintf("饥荒管理平台检测到模组需要更新，本次更新ID为%s，请输入 ID-LKGX 进行模组更新", updateModID)

		return game.SystemMsg(updateTip)
	}

	for _, line := range logContent {
		if strings.Contains(line, "服务器需要从Steam创意工坊获得最新版本") || strings.Contains(line, "is out of date and needs to be updated for new users to be able to join the server") {
			err = sendTip()
			if err != nil {
				logger.Logger.Errorf("玩家更新模组定时任务异常，发送通知失败, err: %v", err)
			}

			return
		}
	}
}
