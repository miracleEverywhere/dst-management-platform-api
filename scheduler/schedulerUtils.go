package scheduler

import (
	"bufio"
	"dst-management-platform-api/app/externalApi"
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

// 将更新时间提前30分钟，提前通知重启服务器，实际重启的时间仍为设置时间
func updateTimeFix(timeStr string) string {
	// 解析时间字符串
	parsedTime, err := time.Parse("15:04:05", timeStr)
	if err != nil {
		fmt.Println("解析时间字符串失败:", err)
		return timeStr
	}

	// 减去30分钟
	duration, _ := time.ParseDuration("-30m")
	newTime := parsedTime.Add(duration)

	// 格式化新的时间字符串
	newTimeStr := newTime.Format("15:04:05")
	return newTimeStr
}

func checkUpdate() {
	dstVersion, _ := externalApi.GetDSTVersion()
	doAnnounce()
	if dstVersion.Local != dstVersion.Server {
		doUpdate()
	}
	doRestart()
}

func doAnnounce() {
	// 重启前进行宣告
	cmd := "c_announce('将在30分钟后自动重启服务器(The server will automatically restart in 30 minutes)')"
	_ = utils.ScreenCMD(cmd, utils.MasterName)
	time.Sleep(10 * time.Minute)
	cmd = "c_announce('将在20分钟后自动重启服务器(The server will automatically restart in 20 minutes)')"
	_ = utils.ScreenCMD(cmd, utils.MasterName)
	time.Sleep(10 * time.Minute)
	cmd = "c_announce('将在10分钟后自动重启服务器(The server will automatically restart in 10 minutes)')"
	_ = utils.ScreenCMD(cmd, utils.MasterName)
	time.Sleep(5 * time.Minute)
	cmd = "c_announce('将在5分钟后自动重启服务器(The server will automatically restart in 5 minutes)')"
	_ = utils.ScreenCMD(cmd, utils.MasterName)
	time.Sleep(4 * time.Minute)
	cmd = "c_announce('将在1分钟后自动重启服务器(The server will automatically restart in 1 minute)')"
	_ = utils.ScreenCMD(cmd, utils.MasterName)
	time.Sleep(1 * time.Minute)
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

func doKeepalive() {
	config, _ := utils.ReadConfig()
	// 先执行命令
	_ = utils.BashCMD(utils.PlayersListCMD)
	// 获取日志文件中的list
	file, err := os.Open(utils.MasterLogPath)
	if err != nil {
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			return
		}
	}(file)

	// 逐行读取文件
	scanner := bufio.NewScanner(file)
	var lines []string
	timeRegex := regexp.MustCompile(`^\[\d{2}:\d{2}:\d{2}\]`)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		return
	}
	// 反向遍历行
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		// 将行添加到结果切片
		match := timeRegex.FindString(line)
		if match != "" {
			// 去掉方括号
			lastTime := strings.Trim(match, "[]")
			if config.Keepalive.LastTime == lastTime {
				doRestart()
			} else {
				config.Keepalive.LastTime = lastTime
				utils.WriteConfig(config)
			}
			return
		}
	}

}
