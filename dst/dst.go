package dst

import (
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
	// room全局文件锁
	roomMutex sync.Mutex
	// world全局文件锁
	worldMutex sync.Mutex
	// player全局文件锁
	playerMutex sync.Mutex
}

func NewGameController(room *models.Room, worlds *[]models.World, setting *models.RoomSetting, lang string) *Game {
	game := &Game{
		room:    room,
		worlds:  worlds,
		setting: setting,
		lang:    lang,
	}

	game.initInfo()
	logger.Logger.Debug(utils.StructToFlatString(game))

	return game
}

func (g *Game) SaveAll() error {
	var err error

	// cluster
	err = g.createRoom()
	if err != nil {
		return err
	}

	// worlds
	err = g.createWorlds()
	if err != nil {
		return err
	}

	return nil
}

func (g *Game) StartWorld(id int) error {
	return g.startWorld(id)
}

func (g *Game) StartAllWorld() error {
	return g.startAllWorld()
}

func (g *Game) initInfo() {
	// room
	g.clusterName = fmt.Sprintf("Cluster_%d", g.room.ID)
	g.clusterPath = fmt.Sprintf("%s/%s", utils.ClusterPath, g.clusterName)
	g.clusterIniPath = fmt.Sprintf("%s/cluster.ini", g.clusterPath)
	g.clusterTokenTxtPath = fmt.Sprintf("%s/cluster_token.txt", g.clusterPath)

	// worlds
	for _, world := range *g.worlds {
		worldPath := fmt.Sprintf("%s/%s", g.clusterPath, world.WorldName)
		serverIniPath := fmt.Sprintf("%s/server.ini", worldPath)
		levelDataOverridePath := fmt.Sprintf("%s/leveldataoverride.lua", worldPath)
		modOverridesPath := fmt.Sprintf("%s/modoverrides.lua", worldPath)
		screenName := fmt.Sprintf("DMP_%s_%s", g.clusterName, world.WorldName)

		var startCmd string
		switch g.setting.StartType {
		case "32-bit":
			startCmd = fmt.Sprintf("cd dst/bin/ && screen -d -h 200 -m -S %s ./dontstarve_dedicated_server_nullrenderer -console -cluster %s -shard %s", screenName, g.clusterName, world.WorldName)
		case "64-bit":
			startCmd = fmt.Sprintf("cd dst/bin64/ && screen -d -h 200 -m -S %s ./dontstarve_dedicated_server_nullrenderer_x64 -console -cluster %s -shard %s", screenName, g.clusterName, world.WorldName)
		default:
			startCmd = "exit 1"
		}

		g.worldSaveData = append(g.worldSaveData, worldSaveData{
			worldPath:             worldPath,
			serverIniPath:         serverIniPath,
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
}
