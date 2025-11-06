package dst

import (
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"strconv"
)

type worldSaveData struct {
	// world 全局文件锁
	worldPath             string
	serverIniPath         string
	levelDataOverridePath string
	modOverridesPath      string
	startCmd              string
	models.World
}

func (g *Game) createWorlds() error {
	g.worldMutex.Lock()
	defer g.worldMutex.Unlock()

	var err error

	for _, world := range g.worldSaveData {

		err = utils.EnsureDirExists(world.worldPath)
		if err != nil {
			return err
		}

		err = utils.TruncAndWriteFile(world.serverIniPath, getServerIni(&world.World))
		if err != nil {
			return err
		}

		err = utils.TruncAndWriteFile(world.levelDataOverridePath, world.LevelData)
		if err != nil {
			return err
		}

		if g.room.ModInOne {
			err = utils.TruncAndWriteFile(world.modOverridesPath, g.room.ModData)
			if err != nil {
				return err
			}
		} else {
			err = utils.TruncAndWriteFile(world.modOverridesPath, world.ModData)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

func (g *Game) startWorld(id int) error {
	for _, world := range g.worldSaveData {
		if world.ID == id {
			logger.Logger.Debug(world.startCmd)
			err := utils.BashCMD(world.startCmd)
			return err
		}
	}

	return nil
}

func (g *Game) startAllWorld() error {
	for _, world := range g.worldSaveData {
		logger.Logger.Debug(world.startCmd)
		err := utils.BashCMD(world.startCmd)
		if err != nil {
			return err
		}
	}

	return nil
}

func getServerIni(world *models.World) string {
	contents := `[NETWORK]
server_port = ` + strconv.Itoa(world.ServerPort) + `

[SHARD]
id = ` + strconv.Itoa(world.GameID) + `
is_master = ` + strconv.FormatBool(world.IsMaster) + `
name = ` + world.WorldName + `

[STEAM]
master_server_port = ` + strconv.Itoa(world.MasterServerPort) + `
authentication_port = ` + strconv.Itoa(world.AuthenticationPort) + `

[ACCOUNT]
encode_user_path = ` + strconv.FormatBool(world.EncodeUserPath)
	return contents
}
