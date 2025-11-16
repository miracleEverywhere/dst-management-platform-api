package dst

import (
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"strconv"
)

type worldSaveData struct {
	worldPath             string
	serverIniPath         string
	levelDataOverridePath string
	modOverridesPath      string
	startCmd              string
	screenName            string
	models.World
}

func (g *Game) createWorlds() error {
	g.worldMutex.Lock()
	defer g.worldMutex.Unlock()

	var (
		err        error
		worldsName []string
	)

	// 保存文件
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

		worldsName = append(worldsName, world.WorldName)
	}

	// 清理删除的世界
	fileSystemWorlds, err := utils.GetDirs(g.clusterPath, false)
	for _, fileSystemWorld := range fileSystemWorlds {
		if !utils.Contains(worldsName, fileSystemWorld) {
			err = utils.RemoveDir(fmt.Sprintf("%s/%s", g.clusterPath, fileSystemWorld))
			if err != nil {
				logger.Logger.Warn("清理世界失败", "err", err)
			}
		}
	}

	return nil
}

func (g *Game) worldUpStatus(id int) bool {
	var (
		stat  bool
		err   error
		world *worldSaveData
	)

	world, err = g.getWorldByID(id)
	if err != nil {
		return false
	}

	cmd := fmt.Sprintf("ps -ef | grep %s | grep -v grep", world.screenName)
	err = utils.BashCMD(cmd)
	if err != nil {
		stat = false
	} else {
		stat = true
	}

	return stat
}

func (g *Game) startWorld(id int) error {
	// 启动游戏后，删除mod临时下载目录
	g.acfMutex.Lock()
	defer g.acfMutex.Unlock()
	defer func() {
		err := utils.RemoveDir(fmt.Sprintf("dmp_files/mods/ugc/%s", g.clusterName))
		if err != nil {
			logger.Logger.Warn("删除临时模组失败", "err", err)
		}
	}()

	var (
		err   error
		world *worldSaveData
	)

	// 如果正在运行，则跳过
	if g.worldUpStatus(id) {
		logger.Logger.Info("当前世界正在运行中，跳过", "世界ID", id)
		return nil
	}

	world, err = g.getWorldByID(id)
	if err != nil {
		return err
	}

	err = g.dsModsSetup()
	if err != nil {
		return err
	}

	logger.Logger.Debug(world.startCmd)
	err = utils.BashCMD(world.startCmd)

	return err
}

func (g *Game) startAllWorld() error {
	var err error

	err = g.dsModsSetup()
	if err != nil {
		return err
	}

	for _, world := range g.worldSaveData {
		// 如果正在运行，则跳过
		if g.worldUpStatus(world.ID) {
			logger.Logger.Info("当前世界正在运行中，跳过", "世界ID", world.ID)
			continue
		}

		logger.Logger.Debug(world.startCmd)
		err = utils.BashCMD(world.startCmd)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Game) stopWorld() error {

	return nil
}

func (g *Game) getWorldByID(id int) (*worldSaveData, error) {
	for _, world := range g.worldSaveData {
		if world.ID == id {
			return &world, nil
		}
	}

	return &worldSaveData{}, nil
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
