package scheduler

import (
	"bufio"
	"dst-management-platform-api/app/home"
	"dst-management-platform-api/utils"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
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

	numPlayer := len(playerList)
	currentTime := utils.GetTimestamp()
	var statistics utils.Statistics
	statistics.Timestamp = currentTime
	statistics.Num = numPlayer
	statisticsLength := len(config.Statistics)
	if statisticsLength > 2880 {
		// 只保留一天的数据量
		config.Statistics = append(config.Statistics[:0], config.Statistics[1:]...)
	}
	config.Statistics = append(config.Statistics, statistics)

	utils.WriteConfig(config)
}

func getPlayersList() ([]string, error) {
	// 先执行命令
	_ = utils.BashCMD(utils.PlayersListCMD)
	// 获取日志文件中的list
	file, err := os.Open(utils.MasterLogPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 逐行读取文件
	scanner := bufio.NewScanner(file)
	var linesAfterKeyword []string
	var lines []string
	keyword := "playerlist 99999999 [0]"
	var foundKeyword bool

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// 反向遍历行
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]

		// 将行添加到结果切片
		linesAfterKeyword = append(linesAfterKeyword, line)

		// 检查是否包含关键字
		if strings.Contains(line, keyword) {
			foundKeyword = true
			break
		}
	}

	if !foundKeyword {
		return nil, fmt.Errorf("keyword not found in the file")
	}

	// 正则表达式匹配模式
	pattern := `playerlist 99999999 \[[0-9]+\] (KU_[A-Za-z0-9]+) (.+)`
	re := regexp.MustCompile(pattern)

	var players []string

	// 查找匹配的行并提取所需字段
	for _, line := range linesAfterKeyword {
		if matches := re.FindStringSubmatch(line); matches != nil {
			// 检查是否包含 [Host]
			if !regexp.MustCompile(`\[Host\]`).MatchString(line) {
				uid := strings.ReplaceAll(matches[1], "\t", "")
				uid = strings.ReplaceAll(uid, " ", "")
				nickName := strings.ReplaceAll(matches[2], "\t", "")
				nickName = strings.ReplaceAll(nickName, " ", "")

				player := uid + "," + nickName
				players = append(players, player)
			}
		}
	}

	players = utils.UniqueSliceKeepOrderString(players)

	return players, nil

}

func execAnnounce(content string) {
	cmd := "c_announce('" + content + "')"
	_ = utils.ScreenCMD(cmd, utils.MasterName)
}

func checkUpdate() {
	dstVersion, _ := home.GetDSTVersion()
	if dstVersion.Local != dstVersion.Server {
		doUpdate()
	}
	doRestart()
}

func doUpdate() {
	config, _ := utils.ReadConfig()
	cmd := "c_shutdown()"
	_ = utils.ScreenCMD(cmd, utils.MasterName)
	if config.RoomSetting.Cave != "" {
		_ = utils.ScreenCMD(cmd, utils.CavesName)
	}

	time.Sleep(2 * time.Second)
	_ = utils.BashCMD(utils.StopMasterCMD)
	if config.RoomSetting.Cave != "" {
		_ = utils.BashCMD(utils.StopCavesCMD)
	}

	go func() {
		_ = utils.BashCMD(utils.UpdateGameCMD)
		_ = utils.BashCMD(utils.StartMasterCMD)
		if config.RoomSetting.Cave != "" {
			_ = utils.BashCMD(utils.StartCavesCMD)
		}
	}()
}

func doRestart() {
	config, _ := utils.ReadConfig()
	cmd := "c_shutdown()"
	_ = utils.ScreenCMD(cmd, utils.MasterName)
	if config.RoomSetting.Cave != "" {
		_ = utils.ScreenCMD(cmd, utils.CavesName)
	}

	time.Sleep(2 * time.Second)
	_ = utils.BashCMD(utils.StopMasterCMD)
	if config.RoomSetting.Cave != "" {
		_ = utils.BashCMD(utils.StopCavesCMD)
	}

	time.Sleep(1 * time.Second)
	_ = utils.BashCMD(utils.KillDST)
	_ = utils.BashCMD(utils.ClearScreenCMD)
	_ = utils.BashCMD(utils.StartMasterCMD)
	if config.RoomSetting.Cave != "" {
		_ = utils.BashCMD(utils.StartCavesCMD)
	}
}

func doBackup() {
	_ = utils.BackupGame()
}
