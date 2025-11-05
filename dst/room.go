package dst

import (
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"strconv"
)

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
	default:
		gameMode = g.room.GameMode
	}

	switch lang {
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
whitelist_slots = ` + strconv.Itoa(0) + `
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
