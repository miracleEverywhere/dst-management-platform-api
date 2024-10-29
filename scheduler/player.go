package scheduler

import (
	"dst-management-platform-api/utils"
	"strings"
)

func setPlayer2DB() {
	config, _ := utils.ReadConfig()

	players, err := getPlayersList()
	if err != nil {
		return
	}
	var playerList []utils.Players
	for _, p := range players {
		var player utils.Players
		uidNickName := strings.Split(p, ",")
		player.UID = uidNickName[0]
		player.NickName = uidNickName[1]
		playerList = append(playerList, player)
	}

	config.Players = playerList
	utils.WriteConfig(config)
}
