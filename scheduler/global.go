package scheduler

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"strings"
)

func onlinePlayerGet() {
	db.PlayersStatisticMutex.Lock()
	defer db.PlayersStatisticMutex.Unlock()
	roomsBasic, err := DBHandler.roomDao.GetRoomBasic()
	if err != nil {
		logger.Logger.Error("查询数据库失败，添加定时任务失败", "err", err)
		return
	}

	for _, rbs := range *roomsBasic {
		room, worlds, roomSetting, err := fetchGameInfo(rbs.RoomID)
		if err != nil {
			logger.Logger.Error("查询数据库失败，添加定时任务失败", "err", err)
			return
		}
		game := dst.NewGameController(room, worlds, roomSetting, "zh")
		var Players db.Players // 当前房间总的玩家结构体
		for _, world := range *worlds {
			if game.WorldUpStatus(world.ID) {
				players, err := game.GetPlayerList(world.ID)
				if err == nil {
					var ps []db.PlayerInfo
					for _, player := range players {
						var playerInfo db.PlayerInfo // 单个玩家
						uidNickName := strings.Split(player, "<-@dmp@->")
						playerInfo.UID = uidNickName[0]
						playerInfo.Nickname = uidNickName[1]
						playerInfo.Prefab = uidNickName[2]
						ps = append(ps, playerInfo)
					}
					if ps == nil {
						ps = []db.PlayerInfo{}
					}
					Players.PlayerInfo = ps
					Players.Timestamp = utils.GetTimestamp()
					db.PlayersStatistic[rbs.RoomID] = append(db.PlayersStatistic[rbs.RoomID], Players)
					// 获取到数据就执行下一个房间
					goto LOOP
				}
			}
		}
	LOOP:
	}
}
