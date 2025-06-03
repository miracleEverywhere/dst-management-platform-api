package setting

import (
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/utils"
	"fmt"
	"math"
	"strconv"
	"strings"
)

type System struct {
	SysSetting       utils.SysSetting       `json:"sysSetting"`
	SchedulerSetting utils.SchedulerSetting `json:"schedulerSetting"`
}

func clusterTemplate(cluster utils.Cluster) string {
	var (
		masterIP   string
		masterPort int
		clusterKey string
		hasMaster  bool
	)

	for _, world := range cluster.Worlds {
		if world.IsMaster {
			masterIP = world.ShardMasterIp
			masterPort = world.ShardMasterPort
			clusterKey = world.ClusterKey
			hasMaster = true
		}
	}

	if !hasMaster {
		masterIP = cluster.Worlds[0].ShardMasterIp
		masterPort = cluster.Worlds[0].ShardMasterPort
		clusterKey = cluster.Worlds[0].ClusterKey
	}

	contents := `
[GAMEPLAY]
game_mode = ` + cluster.ClusterSetting.GameMode + `
max_players = ` + strconv.Itoa(cluster.ClusterSetting.PlayerNum) + `
pvp = ` + strconv.FormatBool(cluster.ClusterSetting.PVP) + `
pause_when_empty = true
vote_enabled = ` + strconv.FormatBool(cluster.ClusterSetting.Vote) + `
vote_kick_enabled = ` + strconv.FormatBool(cluster.ClusterSetting.Vote) + `

[NETWORK]
cluster_description = ` + cluster.ClusterSetting.Description + `
whitelist_slots = ` + strconv.Itoa(cluster.GetWhiteListSlot()) + `
cluster_name = ` + cluster.ClusterSetting.Name + `
cluster_password = ` + cluster.ClusterSetting.Password + `
cluster_language = zh
tick_rate = ` + strconv.Itoa(cluster.SysSetting.TickRate) + `

[MISC]
console_enabled = true
max_snapshots = ` + strconv.Itoa(cluster.ClusterSetting.BackDays) + `

[SHARD]
shard_enabled = true
bind_ip = 0.0.0.0
master_ip = ` + masterIP + `
master_port = ` + strconv.Itoa(masterPort) + `
cluster_key = ` + clusterKey + `
`
	return contents
}

func worldTemplate(world utils.World) string {
	content := `
[NETWORK]
server_port = ` + strconv.Itoa(world.ServerPort) + `

[SHARD]
id = ` + strconv.Itoa(world.ID) + `
is_master = ` + strconv.FormatBool(world.IsMaster) + `
name = ` + world.Name + `

[STEAM]
master_server_port = ` + strconv.Itoa(world.SteamMasterPort) + `
authentication_port = ` + strconv.Itoa(world.SteamAuthenticationPort) + `

[ACCOUNT]
encode_user_path = ` + strconv.FormatBool(world.EncodeUserPath) + `
`
	return content
}

func SaveSetting(reqCluster utils.Cluster) error {
	defer func() {
		scheduler.ReloadScheduler()
	}()
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		return err
	}

	var clusterIndex = -1

	for i, dbCluster := range config.Clusters {
		if dbCluster.ClusterSetting.ClusterName == reqCluster.ClusterSetting.ClusterName {
			clusterIndex = i
			reqCluster.SysSetting = dbCluster.SysSetting
			break
		}
	}

	if clusterIndex == -1 {
		return fmt.Errorf("集群不存在")
	}

	// 对world进行排序
	var formattedCluster utils.Cluster

	for _, world := range reqCluster.Worlds {
		world.ScreenName = fmt.Sprintf("DST_%s_%d", reqCluster.ClusterSetting.ClusterName, world.ID)
		formattedCluster.Worlds = append(formattedCluster.Worlds, world)
	}
	reqCluster.Worlds = formattedCluster.Worlds

	// 写入文件
	err = utils.EnsureDirExists(utils.DstPath)
	if err != nil {
		utils.Logger.Error("创建饥荒目录失败", "err", err)
		return err
	}

	err = utils.EnsureDirExists(reqCluster.GetMainPath())
	if err != nil {
		utils.Logger.Error("创建集群目录失败", "err", err)
		return err
	}

	//cluster.ini
	err = utils.EnsureFileExists(reqCluster.GetIniFile())
	if err != nil {
		utils.Logger.Error("创建cluster.ini失败", "err", err)
		return err
	}

	clusterIniFileContent := clusterTemplate(reqCluster)
	err = utils.TruncAndWriteFile(reqCluster.GetIniFile(), clusterIniFileContent)
	if err != nil {
		utils.Logger.Error("写入cluster.ini失败", "err", err)
		return err
	}

	//cluster_token.txt
	err = utils.EnsureFileExists(reqCluster.GetTokenFile())
	if err != nil {
		utils.Logger.Error("创建cluster_token.txt失败", "err", err)
		return err
	}
	err = utils.TruncAndWriteFile(reqCluster.GetTokenFile(), reqCluster.ClusterSetting.Token)
	if err != nil {
		utils.Logger.Error("写入cluster_token.txt失败", "err", err)
		return err
	}

	for _, world := range reqCluster.Worlds {
		err = utils.EnsureDirExists(world.GetMainPath(reqCluster.ClusterSetting.ClusterName))
		if err != nil {
			utils.Logger.Error("创建世界目录失败", "err", err)
			return err
		}

		// leveldataoverride.lua
		err = utils.EnsureFileExists(world.GetLevelDataFile(reqCluster.ClusterSetting.ClusterName))
		if err != nil {
			utils.Logger.Error("创建leveldataoverride.lua失败", "err", err)
			return err
		}
		err = utils.TruncAndWriteFile(world.GetLevelDataFile(reqCluster.ClusterSetting.ClusterName), world.LevelData)
		if err != nil {
			utils.Logger.Error("写入leveldataoverride.lua失败", "err", err)
			return err
		}

		// modoverrides.lua
		err = utils.EnsureFileExists(world.GetModFile(reqCluster.ClusterSetting.ClusterName))
		if err != nil {
			utils.Logger.Error("创建modoverrides.lua失败", "err", err)
			return err
		}
		err = utils.TruncAndWriteFile(world.GetModFile(reqCluster.ClusterSetting.ClusterName), reqCluster.Mod)
		if err != nil {
			utils.Logger.Error("写入modoverrides.lua失败", "err", err)
			return err
		}

		// server.ini
		err = utils.EnsureFileExists(world.GetIniFile(reqCluster.ClusterSetting.ClusterName))
		if err != nil {
			utils.Logger.Error("创建server.ini失败", "err", err)
			return err
		}
		worldIniContent := worldTemplate(world)
		err = utils.TruncAndWriteFile(world.GetIniFile(reqCluster.ClusterSetting.ClusterName), worldIniContent)
		if err != nil {
			utils.Logger.Error("写入server.ini失败", "err", err)
			return err
		}
	}

	config.Clusters[clusterIndex] = reqCluster
	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("写入配置文件失败", "err", err)
		return err
	}

	return nil
}

func DoImport(filename string, cluster utils.Cluster, langStr string) (bool, string, utils.Cluster, map[string][]string, map[string][]string) {
	var (
		result    bool
		errMsgKey string
	)

	filePath := utils.ImportFileUploadPath + filename
	err := utils.EnsureDirExists(utils.ImportFileUnzipPath)
	if err != nil {
		errMsgKey = "createUnzipDir"
		utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
		return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
	}
	err = utils.BashCMD("unzip -qo " + filePath + " -d " + utils.ImportFileUnzipPath)
	if err != nil {
		errMsgKey = "unzipProcess"
		utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
		return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
	}

	/* ======== cluster.ini ======== */
	clusterIniFilePath := utils.ImportFileUnzipPath + "cluster.ini"
	result, err = utils.FileDirectoryExists(clusterIniFilePath)
	if !result || err != nil {
		errMsgKey = "clusterIniNotFound"
		utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
		return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
	}

	clusterIni, err := utils.ParseIniToMap(clusterIniFilePath)
	if err != nil {
		errMsgKey = "clusterIniReadFail"
		utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
		return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
	}

	if clusterIni["cluster_name"] == "" {
		errMsgKey = "cluster_name_NotSet"
		utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
		return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
	}
	cluster.ClusterSetting.Name = clusterIni["cluster_name"]

	cluster.ClusterSetting.Description = clusterIni["cluster_description"]

	if clusterIni["game_mode"] == "" {
		errMsgKey = "game_mode_NotSet"
		utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
		return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
	}
	cluster.ClusterSetting.GameMode = clusterIni["game_mode"]

	if clusterIni["pvp"] == "" {
		cluster.ClusterSetting.PVP = false
	} else {
		pvp, err := strconv.ParseBool(clusterIni["pvp"])
		if err != nil {
			cluster.ClusterSetting.PVP = false
		} else {
			cluster.ClusterSetting.PVP = pvp
		}
	}

	if clusterIni["max_players"] == "" {
		errMsgKey = "max_players_NotSet"
		utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
		return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
	}
	playnum, err := strconv.Atoi(clusterIni["max_players"])
	if err != nil {
		utils.Logger.Warn("最大玩家数获取异常，设置为默认值6")
		cluster.ClusterSetting.PlayerNum = 6
	} else {
		cluster.ClusterSetting.PlayerNum = playnum
	}

	if clusterIni["max_snapshots"] == "" {
		utils.Logger.Info("未获取到最大回档天数，设置为默认值10")
		cluster.ClusterSetting.BackDays = 10
	} else {
		backdays, err := strconv.Atoi(clusterIni["max_snapshots"])
		if err != nil {
			utils.Logger.Warn("最大回档天数获取异常，设置为默认值10")
			cluster.ClusterSetting.BackDays = 10
		} else {
			cluster.ClusterSetting.BackDays = backdays
		}
	}

	if clusterIni["vote_enabled"] == "" {
		utils.Logger.Info("未获取到是否开启玩家投票，设置为默认值关闭")
		cluster.ClusterSetting.Vote = false
	} else {
		vote, err := strconv.ParseBool(clusterIni["vote_enabled"])
		if err != nil {
			utils.Logger.Warn("是否开启玩家投票获取异常，设置为默认值关闭")
			cluster.ClusterSetting.Vote = false
		} else {
			cluster.ClusterSetting.Vote = vote
		}
	}

	cluster.ClusterSetting.Password = clusterIni["cluster_password"]

	if clusterIni["tick_rate"] == "" {
		utils.Logger.Info("未获取到tick_rate，设置为默认值15")
		cluster.SysSetting.TickRate = 15
	} else {
		tickRate, err := strconv.Atoi(clusterIni["tick_rate"])
		if err != nil {
			utils.Logger.Warn("tick_rate获取异常，设置为默认值15")
			cluster.SysSetting.TickRate = 15
		} else {
			cluster.SysSetting.TickRate = tickRate
		}
	}

	clusterKey := "supersecretkey"
	if clusterIni["cluster_key"] == "" {
		utils.Logger.Info("未获取到cluster_key，设置为默认值'supersecretkey'")
	} else {
		clusterKey = clusterIni["cluster_key"]
	}

	masterIp := "127.0.0.1"
	if clusterIni["master_ip"] == "" {
		errMsgKey = "master_ip_NotSet"
		utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
		return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
	} else {
		masterIp = clusterIni["master_ip"]
	}

	/* ======== cluster_token.txt ======== */
	clusterTokenFilePath := utils.ImportFileUnzipPath + "cluster_token.txt"
	result, err = utils.FileDirectoryExists(clusterTokenFilePath)
	if !result || err != nil {
		errMsgKey = "clusterTokenNotFound"
		utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
		return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
	}
	clusterToken, err := utils.GetFileAllContent(clusterTokenFilePath)
	if err != nil {
		errMsgKey = "clusterTokenReadFail"
		utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
		return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
	}
	cluster.ClusterSetting.Token = clusterToken

	/* ======== whitelist.txt blocklist.txt adminlist.txt ======== */
	lists := make(map[string][]string)
	whiteListFilePath := utils.ImportFileUnzipPath + "whitelist.txt"
	result, err = utils.FileDirectoryExists(whiteListFilePath)
	if !result || err != nil {
		utils.Logger.Warn("未发现白名单文件，跳过")
	} else {
		whiteList, err := utils.ReadLinesToSlice(whiteListFilePath)
		if err != nil {
			utils.Logger.Warn("读取白名单文件失败，跳过")
		} else {
			lists["whitelist.txt"] = whiteList
		}
	}

	blockListFilePath := utils.ImportFileUnzipPath + "blocklist.txt"
	result, err = utils.FileDirectoryExists(blockListFilePath)
	if !result || err != nil {
		utils.Logger.Warn("未发现黑名单文件，跳过")
	} else {
		blockList, err := utils.ReadLinesToSlice(blockListFilePath)
		if err != nil {
			utils.Logger.Warn("读取黑名单文件失败，跳过")
		} else {
			lists["blocklist.txt"] = blockList
		}
	}

	adminListFilePath := utils.ImportFileUnzipPath + "adminlist.txt"
	result, err = utils.FileDirectoryExists(adminListFilePath)
	if !result || err != nil {
		utils.Logger.Warn("未发现管理员名单文件，跳过")
	} else {
		adminList, err := utils.ReadLinesToSlice(adminListFilePath)
		if err != nil {
			utils.Logger.Warn("读取管理员名单文件失败，跳过")
		} else {
			lists["adminlist.txt"] = adminList
		}
	}

	/* ======== Master/ Caves/ ======== */
	fuckWorldsPath, err := utils.GetDirs(utils.ImportFileUnzipPath, true)
	var worldsPath []string
	for _, i := range fuckWorldsPath {
		// 判断是否含有奇奇怪怪的目录，MacOS真是狗屎啊
		lastDir := utils.GetLastDir(i)
		if !strings.HasPrefix(lastDir, "__") {
			worldsPath = append(worldsPath, i)
		}
	}

	if err != nil || len(worldsPath) == 0 {
		errMsgKey = "world_file_path_GetFail"
		utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
		return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
	}

	worldPortFactor, err := utils.GetWorldPortFactor(cluster.ClusterSetting.ClusterName)
	if err != nil {
		errMsgKey = "port_factor_GetFail"
		utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
		return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
	}

	dstFiles := make(map[string][]string)

	for index, worldPath := range worldsPath {
		var world utils.World
		world.ID = 100 + worldPortFactor + index + 1
		world.Name = fmt.Sprintf("World%d", world.ID)

		/* ======== server.ini ======== */
		result, err = utils.FileDirectoryExists(worldPath + "/server.ini")
		if !result || err != nil {
			errMsgKey = "serverIniNotFound"
			utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
			return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
		}

		serverIni, err := utils.ParseIniToMap(worldPath + "/server.ini")
		if err != nil {
			errMsgKey = "clusterIniReadFail"
			utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
			return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
		}

		world.ServerPort = 11000 + worldPortFactor + index + 1

		if serverIni["is_master"] == "" {
			errMsgKey = "is_master_NotSet"
			utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
			return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
		}
		isMaster, err := strconv.ParseBool(serverIni["is_master"])
		if err != nil {
			errMsgKey = "is_master_ValueError"
			utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
			return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
		}
		world.IsMaster = isMaster

		world.ClusterKey = clusterKey

		world.ShardMasterIp = masterIp

		world.ShardMasterPort = 10887 + worldPortFactor + index + 1

		world.SteamMasterPort = 27017 + worldPortFactor + index + 1

		world.SteamAuthenticationPort = 8767 + worldPortFactor + index + 1

		if serverIni["encode_user_path"] == "" {
			world.EncodeUserPath = true
		} else {
			encodeUserPath, err := strconv.ParseBool(serverIni["encode_user_path"])
			if err != nil {
				utils.Logger.Warn("encode_user_path值异常，设置为默认值true")
				world.EncodeUserPath = true
			} else {
				world.EncodeUserPath = encodeUserPath
			}
		}

		/* ======== leveldataoverride.lua(worldgenoverride.lua) ======== */
		// 兼容两种文件名leveldataoverride.lua和worldgenoverride.lua
		levelDataPath := worldPath + "/leveldataoverride.lua"
		result, err = utils.FileDirectoryExists(levelDataPath)
		if !result || err != nil {
			levelDataPath = worldPath + "/worldgenoverride.lua"
			result, err := utils.FileDirectoryExists(levelDataPath)
			if !result || err != nil {
				errMsgKey = "levelDataNotFound"
				utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
				return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
			}
		}
		levelData, err := utils.GetFileAllContent(levelDataPath)
		if err != nil {
			errMsgKey = "levelDataReadFail"
			utils.Logger.Error(responseImportError(errMsgKey, langStr), "err", err)
			return false, errMsgKey, utils.Cluster{}, map[string][]string{}, map[string][]string{}
		}
		world.LevelData = levelData

		/* ======== modoverrides.lua ======== */
		result, err = utils.FileDirectoryExists(worldPath + "/modoverrides.lua")
		if !result || err != nil {
			utils.Logger.Warn("未发现modoverrides.lua文件，跳过")
		}
		mod, err := utils.GetFileAllContent(worldPath + "/modoverrides.lua")
		if err != nil {
			utils.Logger.Warn("读取modoverrides.lua文件失败，跳过")
		} else {
			cluster.Mod = mod
		}

		cluster.Worlds = append(cluster.Worlds, world)

		/* ======== save/ backup/ ======== */
		result, err = utils.FileDirectoryExists(worldPath + "/save")
		if !result || err != nil {
			utils.Logger.Warn("未发现save目录，跳过(注意：没有该目录，游戏启动后会生成新世界)")
		} else {
			dstFiles[fmt.Sprintf("%s", world.Name)] = append(dstFiles[fmt.Sprintf("%s", world.Name)], worldPath+"/save")
		}
		result, err = utils.FileDirectoryExists(worldPath + "/backup")
		if !result || err != nil {
			utils.Logger.Warn("未发现backup目录，跳过")
		} else {
			dstFiles[fmt.Sprintf("%s", world.Name)] = append(dstFiles[fmt.Sprintf("%s", world.Name)], worldPath+"/backup")
		}
	}

	return true, "", cluster, lists, dstFiles
}

func ClearFiles() {
	err := utils.BashCMD("rm -rf " + utils.ImportFileUploadPath + "*")
	if err != nil {
		utils.Logger.Error("清理导入的压缩文件失败", "err", err)
	}
}

func getList(filepath string) []string {
	// 预留位 黑名单 管理员
	al, err := utils.ReadLinesToSlice(filepath)
	if err != nil {
		utils.Logger.Warn("读取文件失败", "err", err, "file", filepath)
		return []string{}
	}
	var uidList []string
	for _, uid := range al {
		if !(uid == "" || strings.HasPrefix(uid, " ")) {
			uidList = append(uidList, uid)
		}
	}

	return uidList
}

func GetPlayerAgePrefab(uid string, cluster utils.Cluster) (int, string, error) {
	var (
		path      string
		cmdAge    string
		cmdPrefab string
		world     utils.World
		hasMaster bool
	)

	for _, i := range cluster.Worlds {
		if i.IsMaster {
			world = i
			hasMaster = true
			break
		}
	}
	if !hasMaster {
		world = cluster.Worlds[0]
	}

	if world.EncodeUserPath {
		sessionFileCmd := "TheNet:GetUserSessionFile(ShardGameIndex:GetSession(), '" + uid + "')"
		userSessionFile, err := utils.ScreenCMDOutput(sessionFileCmd, uid+"UserSessionFile", world.ScreenName, world.GetServerLogFile(cluster.ClusterSetting.ClusterName))
		if err != nil {
			return 0, "", err
		}

		path = world.GetSavePath(cluster.ClusterSetting.ClusterName) + "/" + userSessionFile

		ok, _ := utils.FileDirectoryExists(path)
		if !ok {
			return 0, "", err
		}

	} else {
		cmd := fmt.Sprintf("find %s/session/*/%s_/ -name \"*.meta\" -type f -printf \"%%T@ %%p\\n\" | sort -n | tail -n 1 | cut -d' ' -f2", world.GetSavePath(cluster.ClusterSetting.ClusterName), uid)
		stdout, _, err := utils.BashCMDOutput(cmd)
		if err != nil || stdout == "" {
			utils.Logger.Warn("Bash命令执行失败", "err", err, "cmd", cmd)
			return 0, "", err
		}
		path = stdout[:len(stdout)-6]
	}

	if utils.Platform == "darwin" {
		cmdAge = "ggrep -aoP 'age=\\d+\\.\\d+' " + path + " | awk -F'=' '{print $2}'"
	} else {
		cmdAge = "grep -aoP 'age=\\d+\\.\\d+' " + path + " | awk -F'=' '{print $2}'"
	}

	stdout, _, err := utils.BashCMDOutput(cmdAge)
	if err != nil || stdout == "" {
		utils.Logger.Error("Bash命令执行失败", "err", err, "cmd", cmdAge)
		return 0, "", err
	}

	stdout = strings.TrimSpace(stdout)
	age, err := strconv.ParseFloat(stdout, 64)
	if err != nil {
		utils.Logger.Error("玩家游戏时长转换失败", "err", err)
		age = 0
	}
	age = age / 480
	ageInt := int(math.Round(age))

	if utils.Platform == "darwin" {
		cmdPrefab = "ggrep -aoP '},age=\\d+,prefab=\"(.+)\"}' " + path + " | awk -F'[\"]' '{print $2}'"
	} else {
		cmdPrefab = "grep -aoP '},age=\\d+,prefab=\"(.+)\"}' " + path + " | awk -F'[\"]' '{print $2}'"
	}

	stdout, _, err = utils.BashCMDOutput(cmdPrefab)
	if err != nil || stdout == "" {
		utils.Logger.Error("Bash命令执行失败", "err", err, "cmd", cmdPrefab)
		return ageInt, "", nil
	}
	prefab := strings.TrimSpace(stdout)

	return ageInt, prefab, nil
}

func KillInvalidScreen(oldWorlds, newWorlds []utils.World) {
	var (
		oldWorldScreenNames []string
		newWorldScreenNames []string
	)

	for _, i := range oldWorlds {
		oldWorldScreenNames = append(oldWorldScreenNames, i.ScreenName)
	}
	for _, i := range newWorlds {
		newWorldScreenNames = append(newWorldScreenNames, i.ScreenName)
	}

	for _, oldWorld := range oldWorldScreenNames {
		if !utils.Contains(newWorldScreenNames, oldWorld) {
			killCMD := fmt.Sprintf("ps -ef | grep %s | grep -v grep | awk '{print $2}' | xargs kill -9", oldWorld)
			err := utils.BashCMD(killCMD)
			if err != nil {
				utils.Logger.Info("执行Bash命令失败", "msg", err, "cmd", killCMD)
			}
		}
	}

	_ = utils.BashCMD("screen -wipe")
}
