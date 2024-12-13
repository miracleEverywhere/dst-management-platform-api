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
	//config, err := utils.ReadConfig()
	//if err != nil {
	//	utils.Logger.Error("配置文件读取失败", "err", err)
	//	return
	//}

	players, err := getPlayersList()
	if err != nil {
		utils.Logger.Error("获取玩家列表失败", "err", err)
		return
	}
	var playerList []utils.Players
	for _, p := range players {
		var player utils.Players
		uidNickName := strings.Split(p, ",")
		player.UID = uidNickName[0]
		player.NickName = uidNickName[1]
		player.Prefab = uidNickName[2]
		playerList = append(playerList, player)
	}
	//config.Players = playerList

	numPlayer := len(playerList)
	currentTime := utils.GetTimestamp()
	var statistics utils.Statistics
	statistics.Timestamp = currentTime
	statistics.Num = numPlayer
	statistics.Players = playerList
	//statisticsLength := len(config.Statistics)
	statisticsLength := len(utils.STATISTICS)
	if statisticsLength > 2880 {
		// 只保留一天的数据量
		//config.Statistics = append(config.Statistics[:0], config.Statistics[1:]...)
		utils.STATISTICS = append(utils.STATISTICS[:0], utils.STATISTICS[1:]...)
	}
	//config.Statistics = append(config.Statistics, statistics)
	utils.STATISTICS = append(utils.STATISTICS, statistics)

	//err = utils.WriteConfig(config)
	//if err != nil {
	//	utils.Logger.Error("配置文件写入失败", "err", err)
	//}
}

func getPlayersList() ([]string, error) {
	// 先执行命令
	err := utils.BashCMD(utils.PlayersListCMD)
	if err != nil {
		return nil, err
	}
	// 等待命令执行完毕
	time.Sleep(time.Second * 2)
	// 获取日志文件中的list
	file, err := os.Open(utils.MasterLogPath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			utils.Logger.Error("文件关闭失败", "err", err)
		}
	}(file)

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
	pattern := `playerlist 99999999 \[[0-9]+\] (KU_.+) (.+) (.+)?`
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
				prefab := strings.ReplaceAll(matches[3], "\t", "")
				prefab = strings.ReplaceAll(prefab, " ", "")
				player := uid + "," + nickName + "," + prefab
				players = append(players, player)
			}
		}
	}

	players = utils.UniqueSliceKeepOrderString(players)

	return players, nil

}

func execAnnounce(content string) {
	cmd := "c_announce('" + content + "')"
	err := utils.ScreenCMD(cmd, utils.MasterName)
	if err != nil {
		utils.Logger.Error("执行ScreenCMD失败", "err", err, "cmd", cmd)
	}
}

// 将更新时间提前30分钟，提前通知重启服务器，实际重启的时间仍为设置时间
func updateTimeFix(timeStr string) string {
	// 解析时间字符串
	parsedTime, err := time.Parse("15:04:05", timeStr)
	if err != nil {
		utils.Logger.Error("解析时间字符串失败", "err", err)
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
	dstVersion, err := externalApi.GetDSTVersion()
	if err != nil {
		utils.Logger.Error("解析时间字符串失败", "err", err)
		return
	}
	doAnnounce()
	if dstVersion.Local != dstVersion.Server {
		_ = doUpdate()
	}
	doRestart()
}

func doAnnounce() {
	// 重启前进行宣告
	cmd := "c_announce('将在30分钟后自动重启服务器(The server will automatically restart in 30 minutes)')"
	err := utils.ScreenCMD(cmd, utils.MasterName)
	if err != nil {
		utils.Logger.Error("执行ScreenCMD失败", "err", err, "cmd", cmd)
	}
	time.Sleep(10 * time.Minute)
	cmd = "c_announce('将在20分钟后自动重启服务器(The server will automatically restart in 20 minutes)')"
	err = utils.ScreenCMD(cmd, utils.MasterName)
	if err != nil {
		utils.Logger.Error("执行ScreenCMD失败", "err", err, "cmd", cmd)
	}
	time.Sleep(10 * time.Minute)
	cmd = "c_announce('将在10分钟后自动重启服务器(The server will automatically restart in 10 minutes)')"
	err = utils.ScreenCMD(cmd, utils.MasterName)
	if err != nil {
		utils.Logger.Error("执行ScreenCMD失败", "err", err, "cmd", cmd)
	}
	time.Sleep(5 * time.Minute)
	cmd = "c_announce('将在5分钟后自动重启服务器(The server will automatically restart in 5 minutes)')"
	err = utils.ScreenCMD(cmd, utils.MasterName)
	if err != nil {
		utils.Logger.Error("执行ScreenCMD失败", "err", err, "cmd", cmd)
	}
	time.Sleep(4 * time.Minute)
	cmd = "c_announce('将在1分钟后自动重启服务器(The server will automatically restart in 1 minute)')"
	err = utils.ScreenCMD(cmd, utils.MasterName)
	if err != nil {
		utils.Logger.Error("执行ScreenCMD失败", "err", err, "cmd", cmd)
	}
	time.Sleep(1 * time.Minute)
}

func doUpdate() error {
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		return err
	}

	_ = utils.StopGame()

	go func() {
		err = utils.BashCMD(utils.UpdateGameCMD)
		if err != nil {
			utils.Logger.Error("执行BashCMD失败", "err", err, "cmd", utils.UpdateGameCMD)
		}
		err = utils.BashCMD(utils.StartMasterCMD)
		if err != nil {
			utils.Logger.Error("执行BashCMD失败", "err", err, "cmd", utils.StartMasterCMD)
		}
		if config.RoomSetting.Cave != "" {
			err = utils.BashCMD(utils.StartCavesCMD)
			if err != nil {
				utils.Logger.Error("执行BashCMD失败", "err", err, "cmd", utils.StartCavesCMD)
			}
		}
	}()
	return nil
}

func doRestart() {
	_ = utils.StopGame()
	_ = utils.StartGame()
}

func doBackup() {
	err := utils.BackupGame()
	if err != nil {
		utils.Logger.Error("游戏备份失败", "err", err)
	}
}

func getWorldLastTime(logfile string) (string, error) {
	// 获取日志文件中的list
	file, err := os.Open(logfile)
	if err != nil {
		utils.Logger.Error("打开文件失败", "err", err, "file", logfile)
		return "", err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			utils.Logger.Error("关闭文件失败", "err", err, "file", logfile)
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
		utils.Logger.Error("文件scan失败", "err", err)
		return "", err
	}
	// 反向遍历行
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		// 将行添加到结果切片
		match := timeRegex.FindString(line)
		if match != "" {
			// 去掉方括号
			lastTime := strings.Trim(match, "[]")
			return lastTime, nil
		}
	}

	return "", fmt.Errorf("没有找到日志时间戳")
}

func doKeepalive() {
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		return
	}
	// 先执行命令
	// 地面
	err = utils.BashCMD(utils.PlayersListCMD)
	if err != nil {
		utils.Logger.Error("执行BashCMD失败", "err", err, "cmd", utils.PlayersListCMD)
	}
	// 洞穴
	err = utils.BashCMD(utils.PlayersListCavesCMD)
	if err != nil {
		utils.Logger.Error("执行BashCMD失败", "err", err, "cmd", utils.PlayersListCavesCMD)
	}

	masterLastTime, err := getWorldLastTime(utils.MasterLogPath)
	if err != nil {
		utils.Logger.Error("获取日志信息失败", "err", err)
	}

	if config.Keepalive.LastTime == masterLastTime {
		utils.Logger.Info("发现地面异常，执行重启任务")
		doRestart()
		return
	} else {
		config.Keepalive.LastTime = masterLastTime
	}

	if config.RoomSetting.Cave != "" {
		cavesLastTime, err := getWorldLastTime(utils.CavesLogPath)
		if err != nil {
			utils.Logger.Error("获取日志信息失败", "err", err)
		}

		if config.Keepalive.CavesLastTime == cavesLastTime {
			utils.Logger.Info("发现洞穴异常，执行重启任务")
			doRestart()
			return
		} else {
			config.Keepalive.CavesLastTime = cavesLastTime
		}
	}

	err = utils.WriteConfig(config)
	if err != nil {
		if err != nil {
			utils.Logger.Error("配置文件写入失败", "err", err)
		}
	}
}
