package scheduler

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"strings"
	"sync/atomic"
	"time"
)

func OnlinePlayerGet(interval int, uidMapEnable bool) {
	roomsBasic, err := DBHandler.roomDao.GetRoomBasic()
	if err != nil {
		logger.Logger.Error("查询数据库失败，添加定时任务失败", "err", err)
		return
	}

	for _, rbs := range *roomsBasic {
		// 未激活的房间不添加定时任务
		if !rbs.Status {
			continue
		}

		room, worlds, roomSetting, err := fetchGameInfo(rbs.RoomID)
		if err != nil {
			logger.Logger.Error("查询数据库失败，添加定时任务失败", "err", err)
			return
		}
		game := dst.NewGameController(room, worlds, roomSetting, "zh")
		var Players db.Players // 当前房间总的玩家结构体
		for _, world := range *worlds {
			if game.WorldUpStatus(world.ID) {
				players, err := game.GetOnlinePlayerList(world.ID)
				if err == nil {
					var ps []db.PlayerInfo
					for _, player := range players {
						var playerInfo db.PlayerInfo // 单个玩家
						uidNickName := strings.Split(player, "<-@dmp@->")
						playerInfo.UID = uidNickName[0]
						playerInfo.Nickname = uidNickName[1]
						playerInfo.Prefab = uidNickName[2]
						ps = append(ps, playerInfo)

						// 玩家在线时长统计
						db.PlayersOnlineTimeMutex.Lock()
						if db.PlayersOnlineTime[rbs.RoomID] == nil {
							db.PlayersOnlineTime[rbs.RoomID] = make(map[string]int)
						}
						db.PlayersOnlineTime[rbs.RoomID][playerInfo.Nickname] = db.PlayersOnlineTime[rbs.RoomID][playerInfo.Nickname] + interval
						db.PlayersOnlineTimeMutex.Unlock()

						// 更新uidMap
						if uidMapEnable {
							uidMap := models.UidMap{
								UID:      playerInfo.UID,
								Nickname: playerInfo.Nickname,
								RoomID:   rbs.RoomID,
							}
							err = DBHandler.uidMapDao.UpdateUidMap(&uidMap)
							if err != nil {
								logger.Logger.Error("更新UID MAP失败", "err", err)
							}
						}
					}
					if ps == nil {
						ps = []db.PlayerInfo{}
					}
					Players.PlayerInfo = ps
					Players.Timestamp = utils.GetTimestamp()

					db.PlayersStatisticMutex.Lock()

					if len(db.PlayersStatistic[rbs.RoomID]) > (86400 / interval) {
						// 只保留一天的数据量
						db.PlayersStatistic[rbs.RoomID] = append(db.PlayersStatistic[rbs.RoomID][:0], db.PlayersStatistic[rbs.RoomID][1:]...)
					}
					db.PlayersStatistic[rbs.RoomID] = append(db.PlayersStatistic[rbs.RoomID], Players)

					db.PlayersStatisticMutex.Unlock()

					// 获取到数据就执行下一个房间
					goto LOOP
				}
			}
		}
	LOOP:
	}
}

func SystemMetricsGet(maxHour int) {
	netUP, netDown := utils.NetStatus()
	sysMetrics := db.SysMetrics{
		Timestamp:   utils.GetTimestamp(),
		Cpu:         utils.CpuUsage(),
		Memory:      utils.MemoryUsage(),
		NetUplink:   netUP,
		NetDownlink: netDown,
		Disk:        utils.DiskUsage(),
	}

	if len(db.SystemMetrics) > maxHour*60 {
		db.SystemMetrics = append(db.SystemMetrics[:0], sysMetrics)
	} else {
		db.SystemMetrics = append(db.SystemMetrics, sysMetrics)
	}
}

func GameUpdate(enable bool, restart bool) {
	if !enable {
		return
	}

	if db.DstUpdating {
		return
	}

	logger.Logger.Info("[定时任务]：开始检测游戏是否需要更新")

	v := GetDSTVersion()
	if v.Local < v.Server {
		logger.Logger.Info("检测到游戏需要更新")
		logger.Logger.Info("开始执行游戏更新")

		db.DstUpdating = true

		updateCmd := fmt.Sprintf("cd ~/steamcmd && ./steamcmd.sh +login anonymous +force_install_dir ~/dst +app_update 343050 validate +quit")
		_ = utils.BashCMD(updateCmd)

		logger.Logger.Info("游戏更新结束")

		db.DstUpdating = false

		// 如果设置了更新后重启游戏
		if restart {
			// 1. 获取所有的房间
			roomsBasic, err := DBHandler.roomDao.GetRoomBasic()
			if err != nil {
				logger.Logger.Error("查询数据库失败，添加定时任务失败", "err", err)
				return
			}

			for _, rbs := range *roomsBasic {
				// 2. 如果房间未激活，则跳过重启
				if !rbs.Status {
					logger.Logger.DebugF("房间%s(%d)未激活，跳过重启", rbs.RoomName, rbs.RoomID)
					continue
				}

				logger.Logger.DebugF("开始重启房间：%s(%d)", rbs.RoomName, rbs.RoomID)

				// 3. 重启房间内所有的世界
				room, worlds, roomSetting, err := fetchGameInfo(rbs.RoomID)
				if err != nil {
					logger.Logger.Error("查询数据库失败，添加定时任务失败", "err", err)
					return
				}
				game := dst.NewGameController(room, worlds, roomSetting, "zh")
				_ = game.StopAllWorld()
				_ = game.StartAllWorld()

				logger.Logger.DebugF("重启房间完成：%s(%d)，休眠5秒", rbs.RoomName, rbs.RoomID)

				time.Sleep(5 * time.Second)
			}
		}
	} else {
		logger.Logger.Info("[定时任务]：未发现新版本，游戏无需更新，跳过")
	}
}

func InternetIPUpdate() {
	var (
		internetIp string
		err        error
	)
	internetIp, err = GetInternetIP1()
	if err != nil {
		logger.Logger.Warn("调用公网ip接口1失败", "err", err)
		internetIp, err = GetInternetIP2()
		if err != nil {
			logger.Logger.Warn("调用公网ip接口2失败", "err", err)
			return
		}
	}

	db.InternetIP = internetIp
}

func ModDownloadClean() {
	if atomic.LoadInt32(&db.ModDownloadExecuting) == 0 {
		err := utils.RemoveDir(fmt.Sprintf("%s/mods/ugc", utils.DmpFiles))
		if err != nil {
			logger.Logger.Warn("删除临时模组失败", "err", err)
		}
	}
}
