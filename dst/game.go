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
	worldSaveData
}

type roomSaveData struct {
	// room 全局文件锁
	roomMutex sync.Mutex
	// dir
	clusterName string
	clusterPath string
	// file
	clusterIniPath      string
	clusterTokenTxtPath string
}

type worldSaveData struct {
	// world 全局文件锁
	worldMutex sync.Mutex
}

func NewGameController(room *models.Room, worlds *[]models.World, setting *models.RoomSetting, lang string) *Game {
	return &Game{
		room:    room,
		worlds:  worlds,
		setting: setting,
		lang:    lang,
	}
}

func (g *Game) Save() error {
	g.initInfo()
	logger.Logger.Debug(utils.StructToFlatString(g))

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

func (g *Game) initInfo() {
	g.clusterName = fmt.Sprintf("Cluster_%d", g.room.ID)
	g.clusterPath = fmt.Sprintf("%s/%s", utils.ClusterPath, g.clusterName)
	g.clusterIniPath = fmt.Sprintf("%s/cluster.ini", g.clusterPath)
	g.clusterTokenTxtPath = fmt.Sprintf("%s/cluster_token.txt", g.clusterPath)
}
