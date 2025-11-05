package dst

import (
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
)

type Game struct {
	room    *models.Room
	worlds  *[]models.World
	setting *models.RoomSetting
	saveData
}

type saveData struct {
	clusterName string
	clusterPath string
}

func NewGameController(room *models.Room, worlds *[]models.World, setting *models.RoomSetting) *Game {
	return &Game{
		room:    room,
		worlds:  worlds,
		setting: setting,
	}
}

func (g *Game) Save() error {
	logger.Logger.Debug("哈哈哈哈哈哈哈")
	return nil
}

func (g *Game) initInfo() {
	g.clusterName = fmt.Sprintf("Cluster_%d", g.room.ID)
	g.clusterPath = fmt.Sprintf("%s/%s", utils.ClusterPath, g.clusterName)
}
