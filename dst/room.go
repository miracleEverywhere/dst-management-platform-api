package dst

import (
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"strconv"
	"strings"
)

type roomSaveData struct {
	// dir
	clusterName string
	clusterPath string
	// file
	clusterIniPath      string
	clusterTokenTxtPath string
}

func (g *Game) createRoom() error {
	g.roomMutex.Lock()
	defer g.roomMutex.Unlock()

	var err error

	err = utils.EnsureDirExists(g.clusterPath)
	if err != nil {
		return err
	}

	err = utils.TruncAndWriteFile(g.clusterIniPath, g.getClusterIni())
	if err != nil {
		return err
	}

	err = utils.TruncAndWriteFile(g.clusterTokenTxtPath, g.room.Token)
	if err != nil {
		return err
	}

	return nil
}

func (g *Game) getClusterIni() string {
	var (
		gameMode string
		lang     string
	)

	switch g.room.GameMode {
	case "relaxed":
		gameMode = "survival"
	case "wilderness":
		gameMode = "survival"
	case "lightsOut":
		gameMode = "survival"
	case "custom":
		gameMode = g.room.CustomGameMode
	default:
		gameMode = g.room.GameMode
	}

	switch g.lang {
	case "zh":
		lang = "zh"
	case "en":
		lang = "en"
	default:
		lang = "zh"
	}

	contents := `[GAMEPLAY]
game_mode = ` + gameMode + `
max_players = ` + strconv.Itoa(g.room.MaxPlayer) + `
pvp = ` + strconv.FormatBool(g.room.Pvp) + `
pause_when_empty = ` + strconv.FormatBool(g.room.PauseEmpty) + `
vote_enabled = ` + strconv.FormatBool(g.room.Vote) + `
vote_kick_enabled = ` + strconv.FormatBool(g.room.Vote) + `

[NETWORK]
cluster_description = ` + g.room.Description + `
whitelist_slots = ` + strconv.Itoa(len(g.adminlist)) + `
cluster_name = ` + g.room.GameName + `
cluster_password = ` + g.room.Password + `
cluster_language = ` + lang + `
tick_rate = ` + strconv.Itoa(g.setting.TickRate) + `

[MISC]
console_enabled = true
max_snapshots = ` + strconv.Itoa(g.room.MaxRollBack) + `

[SHARD]
shard_enabled = true
bind_ip = 0.0.0.0
master_ip = ` + g.room.MasterIP + `
master_port = ` + strconv.Itoa(g.room.MasterPort) + `
cluster_key = ` + g.room.ClusterKey + `
`

	logger.Logger.Debug(contents)

	return contents
}

func (g *Game) reset(force bool) error {
	if force {
		defer func() {
			_ = g.startAllWorld()
		}()

		err := g.stopAllWorld()
		if err != nil {
			return err
		}

		allSuccess := true

		for _, world := range g.worldSaveData {
			err = utils.RemoveDir(world.savePath)
			if err != nil {
				allSuccess = false
				logger.Logger.Error("删除存档文件失败", "err", err)
			}
		}

		if allSuccess {
			return nil
		} else {
			return fmt.Errorf("删除存档文件失败")
		}

	} else {
		resetCmd := fmt.Sprintf("c_regenerateworld()")
		return utils.ScreenCMD(resetCmd, g.worldSaveData[0].screenName)
	}
}

func (g *Game) announce(message string) error {
	s := strings.ReplaceAll(message, "'", "")
	s = strings.ReplaceAll(s, "\"", "")
	cmd := fmt.Sprintf("c_announce('%s')", s)
	for _, world := range g.worldSaveData {
		err := utils.ScreenCMD(cmd, world.screenName)
		if err == nil {
			return err
		}
	}

	return fmt.Errorf("执行失败")
}
