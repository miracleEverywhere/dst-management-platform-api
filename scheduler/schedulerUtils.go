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

func getPlayers(config utils.Config) {
	var (
		players    []string
		playerList []utils.Players
		err        error
	)

	for _, cluster := range config.Clusters {
		err = nil
		for _, world := range cluster.Worlds {
			if world.GetStatus() {
				players, err = getPlayersList(world, cluster.ClusterSetting.ClusterName)
				if err == nil {
					break
				}
			}
		}
		if err != nil {
			utils.Logger.Warn("获取玩家列表失败", "err", err, "cluster", cluster.ClusterSetting.ClusterName)
			continue
		}

		for _, p := range players {
			var player utils.Players
			uidNickName := strings.Split(p, "<-@dmp@->")
			player.UID = uidNickName[0]
			player.NickName = uidNickName[1]
			player.Prefab = uidNickName[2]
			playerList = append(playerList, player)
		}

		numPlayer := len(playerList)
		currentTime := utils.GetTimestamp()

		var statistics utils.Statistics
		statistics.Timestamp = currentTime
		statistics.Num = numPlayer
		statistics.Players = playerList

		statisticsLength := len(utils.STATISTICS[cluster.ClusterSetting.ClusterName])
		if statisticsLength > 2879 {
			// 只保留一天的数据量
			utils.STATISTICS[cluster.ClusterSetting.ClusterName] = append(utils.STATISTICS[cluster.ClusterSetting.ClusterName][:0], utils.STATISTICS[cluster.ClusterSetting.ClusterName][1:]...)
		}
		utils.STATISTICS[cluster.ClusterSetting.ClusterName] = append(utils.STATISTICS[cluster.ClusterSetting.ClusterName], statistics)
	}
}

func getPlayersList(world utils.World, clusterName string) ([]string, error) {
	var file *os.File

	// 先执行命令
	err := utils.BashCMD(world.GeneratePlayersListCMD())
	if err != nil {
		return nil, err
	}
	// 等待命令执行完毕
	time.Sleep(time.Second * 2)
	// 获取日志文件中的list
	file, err = os.Open(world.GetServerLogFile(clusterName))
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
	pattern := `playerlist 99999999 \[[0-9]+\] (KU_.+) <-@dmp@-> (.*) <-@dmp@-> (.+)?`
	re := regexp.MustCompile(pattern)

	var players []string

	// 查找匹配的行并提取所需字段
	for _, line := range linesAfterKeyword {
		if matches := re.FindStringSubmatch(line); matches != nil {
			// 检查是否包含 [Host]
			if !regexp.MustCompile(`\[Host]`).MatchString(line) {
				uid := strings.ReplaceAll(matches[1], "\t", "")
				//uid = strings.ReplaceAll(uid, " ", "")
				nickName := strings.ReplaceAll(matches[2], "\t", "")
				//nickName = strings.ReplaceAll(nickName, " ", "")
				prefab := strings.ReplaceAll(matches[3], "\t", "")
				//prefab = strings.ReplaceAll(prefab, " ", "")
				player := uid + "<-@dmp@->" + nickName + "<-@dmp@->" + prefab
				players = append(players, player)
			}
		}
	}

	players = utils.UniqueSliceKeepOrderString(players)

	return players, nil
}

// doAnnounce 定时通知
func doAnnounce(content string, cluster utils.Cluster) {
	cmd := "c_announce('" + content + "')"

	for _, world := range cluster.Worlds {
		if world.GetStatus() {
			err := utils.ScreenCMD(cmd, world.ScreenName)
			if err != nil {
				utils.Logger.Error("执行ScreenCMD失败", "err", err, "cmd", cmd)
			} else {
				break
			}
		}
	}
}

// 将更新时间提前15分钟，提前通知重启服务器，实际重启的时间仍为设置时间
func updateTimeFix(timeStr string) string {
	// 解析时间字符串
	parsedTime, err := time.Parse("15:04:05", timeStr)
	if err != nil {
		utils.Logger.Error("解析时间字符串失败", "err", err)
		return timeStr
	}

	// 减去30分钟
	duration, _ := time.ParseDuration("-15m")
	newTime := parsedTime.Add(duration)

	// 格式化新的时间字符串
	newTimeStr := newTime.Format("15:04:05")
	return newTimeStr
}

func checkUpdate(config utils.Config) {
	dstVersion, err := externalApi.GetDSTVersion()
	if err != nil {
		utils.Logger.Error("获取饥荒版本失败，跳过自动更新", "err", err)
		return
	}
	if dstVersion.Local != dstVersion.Server {
		for _, cluster := range config.Clusters {
			for _, world := range cluster.Worlds {
				if world.IsMaster {
					go func() {
						restartAnnounce(world)
					}()
				}
			}
		}
		// 异步执行宣告，宣告需要15分钟，因此sleep 15 分钟
		time.Sleep(15 * time.Minute)
		_ = doUpdate(config)
	}
}

// restartAnnounce 重启前通知
func restartAnnounce(world utils.World) {
	var (
		cmd string
		err error
	)
	// 重启前进行宣告
	cmd = "c_announce('将在15分钟后自动重启服务器(The server will automatically restart in 15 minutes)')"
	err = utils.ScreenCMD(cmd, world.ScreenName)
	if err != nil {
		utils.Logger.Error("执行ScreenCMD失败", "err", err, "cmd", cmd)
	}
	time.Sleep(5 * time.Minute)
	cmd = "c_announce('将在10分钟后自动重启服务器(The server will automatically restart in 10 minutes)')"
	err = utils.ScreenCMD(cmd, world.ScreenName)
	if err != nil {
		utils.Logger.Error("执行ScreenCMD失败", "err", err, "cmd", cmd)
	}
	time.Sleep(5 * time.Minute)
	cmd = "c_announce('将在5分钟后自动重启服务器(The server will automatically restart in 5 minutes)')"
	err = utils.ScreenCMD(cmd, world.ScreenName)
	if err != nil {
		utils.Logger.Error("执行ScreenCMD失败", "err", err, "cmd", cmd)
	}
	time.Sleep(4 * time.Minute)
	cmd = "c_announce('将在1分钟后自动重启服务器(The server will automatically restart in 1 minute)')"
	err = utils.ScreenCMD(cmd, world.ScreenName)
	if err != nil {
		utils.Logger.Error("执行ScreenCMD失败", "err", err, "cmd", cmd)
	}
	time.Sleep(1 * time.Minute)
}

func doUpdate(config utils.Config) error {

	_ = utils.StopAllClusters(config.Clusters)

	go func() {
		err := utils.BashCMD(utils.GetDSTUpdateCmd())
		if err != nil {
			utils.Logger.Error("执行BashCMD失败", "err", err, "cmd", utils.GetDSTUpdateCmd())
		}
		_ = utils.StartAllClusters(config.Clusters)
	}()
	return nil
}

func doRestart(cluster utils.Cluster) {
	_ = utils.StopClusterAllWorlds(cluster)
	time.Sleep(3 * time.Second)
	_ = utils.StartClusterAllWorlds(cluster)
}

func doBackup(cluster utils.Cluster) {
	err := utils.BackupGame(cluster)
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
	timeRegex := regexp.MustCompile(`^\[\d{2}:\d{2}:\d{2}]`)

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

func doKeepalive(cluster utils.Cluster) {
	for _, world := range cluster.Worlds {
		if world.LevelData != "" {
			_ = utils.BashCMD(world.GeneratePlayersListCMD())
			time.Sleep(1 * time.Second)
			lastAliveTime, err := getWorldLastTime(world.GetServerLogFile(cluster.ClusterSetting.ClusterName))
			if err != nil {
				utils.Logger.Error("获取日志信息失败", "err", err)
			}

			if world.LastAliveTime == lastAliveTime {
				utils.Logger.Info("发现服务器运行异常，执行重启任务", "集群", cluster.ClusterSetting.ClusterName, "世界", world.Name)
				_ = world.StopGame(cluster.ClusterSetting.ClusterName)
				time.Sleep(3 * time.Second)
				_ = world.StartGame(cluster.ClusterSetting.ClusterName, cluster.Mod, cluster.SysSetting.Bit64)
				break
			} else {
				config, err := utils.ReadConfig()
				if err != nil {
					utils.Logger.Error("配置文件读取失败", "err", err)
					return
				}

				for clusterIndex, willWriteCluster := range config.Clusters {
					if cluster.ClusterSetting.ClusterName == willWriteCluster.ClusterSetting.ClusterName {
						for worldIndex, willWriteWorld := range willWriteCluster.Worlds {
							if world.Name == willWriteWorld.Name {
								config.Clusters[clusterIndex].Worlds[worldIndex].LastAliveTime = lastAliveTime
								err = utils.WriteConfig(config)
								if err != nil {
									utils.Logger.Error("配置文件写入失败", "err", err)
								}
							}
						}
					}
				}
			}
		}
	}
}

func maintainUidMap(config utils.Config) {
	for _, cluster := range config.Clusters {
		uidMap, err := utils.ReadUidMap(cluster)
		if err != nil {
			utils.Logger.Error("读取历史玩家字典失败", "err", err)
			continue
		}
		if len(utils.STATISTICS[cluster.ClusterSetting.ClusterName]) < 2 {
			continue
		}
		currentPlaylist := utils.STATISTICS[cluster.ClusterSetting.ClusterName][len(utils.STATISTICS[cluster.ClusterSetting.ClusterName])-1].Players
		for _, i := range currentPlaylist {
			uid := i.UID
			nickname := i.NickName
			value, exists := uidMap[uid]
			if exists {
				if value != nickname {
					uidMap[uid] = nickname
				}
			} else {
				uidMap[uid] = nickname
			}
		}
		err = utils.WriteUidMap(uidMap, cluster)
		if err != nil {
			utils.Logger.Error("写入历史玩家字典失败", "err", err)
		}
	}
}

func getSysMetrics() {
	cpu, err := utils.CpuUsage()
	if err != nil {
		return
	}
	mem, err := utils.MemoryUsage()
	if err != nil {
		return
	}
	netUplink, netDownlink, err := utils.NetStatus()
	if err != nil {
		return
	}
	currentTime := utils.GetTimestamp()

	var metrics utils.SysMetrics
	metrics.Timestamp = currentTime
	metrics.Cpu = cpu
	metrics.Memory = mem
	metrics.NetUplink = netUplink
	metrics.NetDownlink = netDownlink

	metricsLength := len(utils.SYSMETRICS)

	if metricsLength > 720 {
		utils.SYSMETRICS = append(utils.SYSMETRICS[:0], utils.SYSMETRICS[1:]...)
	}
	utils.SYSMETRICS = append(utils.SYSMETRICS, metrics)
}

func modUpdate(cluster utils.Cluster, check bool) {
	if utils.UpdateModProcessing {
		return
	}

	var (
		chatLogLines []string
		err          error
		screenName   string
		cmd          string
	)
	for _, world := range cluster.Worlds {
		if world.GetStatus() {
			filePath := fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s/server_chat_log.txt", utils.HomeDir, cluster.ClusterSetting.ClusterName, world.Name)
			chatLogLines, err = utils.GetFileLastNLines(filePath, 100)
			if err == nil {
				screenName = world.ScreenName
				break
			}
		}
	}

	if screenName == "" {
		return
	}

	if check {
		// LKGX 立刻更新
		pattern := `([A-Z0-9]{7})-LKGX`
		re := regexp.MustCompile(pattern)
		for _, line := range chatLogLines {
			subMatches := re.FindStringSubmatch(line)
			if len(subMatches) > 1 {
				if utils.VerifyUpdateModID(subMatches[1]) {
					utils.UpdateModProcessing = true
					cmd = "c_announce('模组更新命令校验成功 (Updating mods command check success)')"
					_ = utils.ScreenCMD(cmd, screenName)
					time.Sleep(500 * time.Millisecond)
					cmd = "c_announce('将在1分钟后自动重启服务器并更新模组 (The server will automatically restart in 1 minute to update mods)')"
					_ = utils.ScreenCMD(cmd, screenName)
					time.Sleep(1 * time.Minute)
					_ = utils.StopClusterAllWorlds(cluster)
					time.Sleep(3 * time.Second)
					_ = utils.StartClusterAllWorlds(cluster)
					time.Sleep(10 * time.Minute)
					utils.UpdateModProcessing = false
					return
				}
			}
		}
	} else {
		for _, line := range chatLogLines {
			if strings.Contains(line, "模组需要更新啦") {
				updateModID := utils.GenerateUpdateModID()
				if len(updateModID) == 0 {
					return
				}
				cmd = fmt.Sprintf("c_announce('饥荒管理平台检测到模组需要更新，本次更新ID为%s，请输入ID-LKGX进行模组更新 (DMP found mods need to be updated, update ID is %s, input ID-LKGX to update)')", updateModID, updateModID)
				_ = utils.ScreenCMD(cmd, screenName)
				return
			}
		}
	}
}

func ReloadScheduler() {
	Scheduler.Stop()
	Scheduler.Clear()
	InitTasks()
	go Scheduler.StartAsync()
}
