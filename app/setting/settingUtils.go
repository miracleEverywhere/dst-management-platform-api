package setting

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"os"
	"strconv"
)

func clusterTemplate(base utils.RoomSettingBase) string {
	contents := `
[GAMEPLAY]
game_mode = ` + base.GameMode + `
max_players = ` + strconv.Itoa(base.PlayerNum) + `
pvp = ` + strconv.FormatBool(base.PVP) + `
pause_when_empty = true
vote_enabled = ` + strconv.FormatBool(base.Vote) + `

[NETWORK]
cluster_description = ` + base.Description + `
cluster_name = ` + base.Name + `
cluster_password = ` + base.Password + `

[MISC]
console_enabled = true
max_snapshots = ` + strconv.Itoa(base.BackDays) + `

[SHARD]
shard_enabled = true
bind_ip = 127.0.0.1
master_ip = 127.0.0.1
master_port = 10889
cluster_key = supersecretkey
`
	return contents
}

func masterServerTemplate() string {
	content := `
[NETWORK]
server_port = 11000

[SHARD]
is_master = true

[STEAM]
master_server_port = 27018
authentication_port = 8768
`
	return content
}

func cavesServerTemplate() string {
	content := `
[NETWORK]
server_port = 11001

[SHARD]
is_master = false
name = Caves

[STEAM]
master_server_port = 27019
authentication_port = 8769
`
	return content
}

func saveSetting(c *gin.Context, config utils.Config, langStr string) {
	utils.DeleteDir(utils.ServerPath + utils.MasterName)
	utils.DeleteDir(utils.ServerPath + utils.CavesName)
	clusterIniFileContent := clusterTemplate(config.RoomSetting.Base)
	utils.TruncAndWriteFile(utils.ServerSettingPath, clusterIniFileContent)
	utils.TruncAndWriteFile(utils.ServerTokenPath, config.RoomSetting.Base.Token)
	err := os.MkdirAll(utils.MasterPath, 0755)
	if err != nil {
		utils.RespondWithError(c, 500, langStr)
		return
	}
	utils.TruncAndWriteFile(utils.MasterSettingPath, config.RoomSetting.Ground)
	utils.TruncAndWriteFile(utils.MasterModPath, config.RoomSetting.Mod)
	utils.TruncAndWriteFile(utils.MasterServerPath, masterServerTemplate())
	if config.RoomSetting.Cave != "" {
		err := os.MkdirAll(utils.CavesPath, 0755)
		if err != nil {
			utils.RespondWithError(c, 500, langStr)
			return
		}
		utils.TruncAndWriteFile(utils.CavesSettingPath, config.RoomSetting.Cave)
		utils.TruncAndWriteFile(utils.CavesModPath, config.RoomSetting.Mod)
		utils.TruncAndWriteFile(utils.CavesServerPath, cavesServerTemplate())
	}
}
