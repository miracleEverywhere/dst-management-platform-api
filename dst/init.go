package dst

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"sync"
)

type Game struct {
	room    *models.Room
	worlds  *[]models.World
	setting *models.RoomSetting
	lang    string
	roomSaveData
	worldSaveData []worldSaveData
	playerSaveData
	modSaveData
	// room全局文件锁
	roomMutex sync.Mutex
	// world全局文件锁
	worldMutex sync.Mutex
	// player全局文件锁
	playerMutex sync.Mutex
	// acf文件锁
	acfMutex sync.Mutex
	// mod 文件、map锁
	modMutex sync.Mutex
}

func NewGameController(room *models.Room, worlds *[]models.World, setting *models.RoomSetting, lang string) *Game {
	game := &Game{
		room:    room,
		worlds:  worlds,
		setting: setting,
		lang:    lang,
	}

	game.initInfo()

	return game
}

func (g *Game) initInfo() {
	// room
	g.clusterName = fmt.Sprintf("Cluster_%d", g.room.ID)
	g.clusterPath = fmt.Sprintf("%s/%s", utils.ClusterPath, g.clusterName)
	g.clusterIniPath = fmt.Sprintf("%s/cluster.ini", g.clusterPath)
	g.clusterTokenTxtPath = fmt.Sprintf("%s/cluster_token.txt", g.clusterPath)

	// worlds
	for _, world := range *g.worlds {
		customGameStartupCmd := world.CustomStartupCmd
		if !utils.IsSafeString(world.WorldName) {
			logger.Logger.Warnf("世界名 %s 可能存在注入风险，跳过", world.WorldName)
			continue
		}
		worldPath := fmt.Sprintf("%s/%s", g.clusterPath, world.WorldName)
		serverIniPath := fmt.Sprintf("%s/server.ini", worldPath)
		savePath := fmt.Sprintf("%s/save", worldPath)
		sessionPath := fmt.Sprintf("%s/session", savePath)
		levelDataOverridePath := fmt.Sprintf("%s/leveldataoverride.lua", worldPath)
		modOverridesPath := fmt.Sprintf("%s/modoverrides.lua", worldPath)
		screenName := fmt.Sprintf("DMP_%s_%s", g.clusterName, world.WorldName)

		startCmd := g.generateGameStartCmd(screenName, customGameStartupCmd, world.WorldName)

		g.worldSaveData = append(g.worldSaveData, worldSaveData{
			worldPath:             worldPath,
			serverIniPath:         serverIniPath,
			savePath:              savePath,
			sessionPath:           sessionPath,
			levelDataOverridePath: levelDataOverridePath,
			modOverridesPath:      modOverridesPath,
			startCmd:              startCmd,
			screenName:            screenName,
			World:                 world,
		})
	}

	// players
	g.adminlistPath = fmt.Sprintf("%s/adminlist.txt", g.clusterPath)
	g.whitelistPath = fmt.Sprintf("%s/whitelist.txt", g.clusterPath)
	g.blocklistPath = fmt.Sprintf("%s/blocklist.txt", g.clusterPath)
	g.adminlist = getPlayerList(g.adminlistPath)
	g.whitelist = getPlayerList(g.whitelistPath)
	g.blocklist = getPlayerList(g.blocklistPath)

	// mods
	if cache.OsType == utils.Darwin {
		g.ugcPath = fmt.Sprintf("%s/%s/%s", cache.CurrentDir, utils.DarwinGameUgcPath, g.clusterName)
		g.notUgcPath = utils.DarwinGameNotUgcPath
	} else {
		g.ugcPath = fmt.Sprintf("%s/%s/%s", cache.CurrentDir, utils.GameUgcPath, g.clusterName)
		g.notUgcPath = utils.GameNotUgcPath
	}

}

func (g *Game) generateGameStartCmd(screenName, customGameStartupCmd, worldName string) string {
	var startCmd string

	if cache.OsType == utils.Darwin {
		startCmd = fmt.Sprintf("cd %s && export DYLD_LIBRARY_PATH=$DYLD_LIBRARY_PATH:%s/steamcmd && screen -d -h 200 -m -S %s %s ./dontstarve_dedicated_server_nullrenderer -console -cluster %s -shard %s", utils.DarwinDstBinDir, cache.CurrentDir, screenName, customGameStartupCmd, g.clusterName, worldName)
	} else {
		switch g.setting.StartType {
		case "32-bit":
			startCmd = fmt.Sprintf("cd dst/bin/ && screen -d -h 200 -m -S %s %s ./dontstarve_dedicated_server_nullrenderer -console -cluster %s -shard %s", screenName, customGameStartupCmd, g.clusterName, worldName)
		case "64-bit":
			startCmd = fmt.Sprintf("cd dst/bin64/ && screen -d -h 200 -m -S %s %s ./dontstarve_dedicated_server_nullrenderer_x64 -console -cluster %s -shard %s", screenName, customGameStartupCmd, g.clusterName, worldName)
		case "luajit":
			startCmd = fmt.Sprintf("cd dst/bin64/ && screen -d -h 200 -m -S %s %s ./dontstarve_dedicated_server_nullrenderer_x64_luajit -console -cluster %s -shard %s", screenName, customGameStartupCmd, g.clusterName, worldName)
		default:
			startCmd = "exit 1"
		}
	}

	return startCmd
}
