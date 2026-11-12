package room

import (
	"bufio"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// WorldPath 存档中世界的名称与解压目录
type WorldPath struct {
	name string
	path string
}

// UploadExtraInfo 存档导入过程中提取的附加信息
type UploadExtraInfo struct {
	adminlist string
	blocklist string
	whitelist string
	worldPath []WorldPath
}

// ImportNewRoomFromZip 从已经保存到磁盘的 zip 导入一个新房间。
// savePath 是 zip 文件路径，unzipPath 是解压目录（需要以路径分隔符结尾）。
// 返回 room、失败时的业务错误码（0 表示未指定）和原始错误。
func (s *Service) ImportNewRoomFromZip(actor service.Actor, savePath, unzipPath string) (*models.Room, int, error) {
	return s.importRoomFromZip(actor, savePath, unzipPath, 0, true)
}

func (s *Service) importRoomFromZip(actor service.Actor, savePath, unzipPath string, roomID int, newRoom bool) (*models.Room, int, error) {

	var (
		room            models.Room
		worlds          []models.World
		roomSetting     models.RoomSetting
		uploadExtraInfo UploadExtraInfo
	)

	code, err := handleUpload(savePath, unzipPath, &room, &worlds, &roomSetting, &uploadExtraInfo)
	if err != nil {
		logger.Logger.Errorf("处理上传文件失败, err: %v", err)
		return nil, code, err
	}

	if len(uploadExtraInfo.worldPath) == 0 {
		return nil, constant.CodeRoomImportNoWorlds, fmt.Errorf("存档中没有可用世界")
	}

	// 设置所有的port和roomSetting
	if newRoom {
		room.Status = true
		roomCount, errCount := s.roomDao.Count(nil)
		if errCount != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", errCount)
			return nil, constant.CodeRoomDBError, errCount
		}

		worldCount, errCount := s.worldDao.Count(nil)
		if errCount != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", errCount)
			return nil, constant.CodeRoomDBError, errCount
		}

		room.MasterPort = 21000 + int(roomCount) + 1
		for index := range worlds {
			worlds[index].ServerPort = 11000 + int(worldCount) + index + 1
			logger.Logger.Debugf("正在设置ServerPort, world.ServerPort: %v", worlds[index].ServerPort)
			worlds[index].MasterServerPort = 31000 + int(worldCount) + index + 1
			logger.Logger.Debugf("正在设置MasterServerPort, world.MasterServerPort: %v", worlds[index].MasterServerPort)
			worlds[index].AuthenticationPort = 41000 + int(worldCount) + index + 1
			logger.Logger.Debugf("正在设置AuthenticationPort, world.AuthenticationPort: %v", worlds[index].AuthenticationPort)
		}

		// 新房间的默认设置，与 roomPost 保持一致
		roomSetting.BackupEnable = true
		roomSetting.BackupSetting = "[{\"time\":\"06:00:00\"}]"
		roomSetting.BackupCleanEnable = false
		roomSetting.BackupCleanSetting = 30
		roomSetting.RestartEnable = false
		roomSetting.RestartSetting = "06:30:00"
		roomSetting.KeepaliveEnable = false
		roomSetting.KeepaliveSetting = 30
		roomSetting.ScheduledStartStopEnable = false
		roomSetting.ScheduledStartStopSetting = "{\"start\":\"07:00:00\",\"stop\":\"01:00:00\"}"
		roomSetting.AnnounceSetting = "[]"
		roomSetting.StartType = "32-bit"
	} else {
		dbRoom, errRoom := s.roomDao.GetRoomByID(roomID)
		if errRoom != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", errRoom)
			return nil, constant.CodeRoomDBError, errRoom
		}
		room.MasterPort = dbRoom.MasterPort
		room.ID = roomID
		// Status 是运行时状态，Lan/Offline/CustomGameMode 当前不由 handleUpload
		// 从存档解析；保留数据库中的值，避免 UpdateRoom 的 Save 用零值覆盖。
		room.Status = dbRoom.Status
		room.Lan = dbRoom.Lan
		room.Offline = dbRoom.Offline
		room.CustomGameMode = dbRoom.CustomGameMode

		dbWorlds, errWorlds := s.worldDao.GetWorldsByRoomID(roomID)
		if errWorlds != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", errWorlds)
			return nil, constant.CodeRoomDBError, errWorlds
		}
		if len(worlds) != len(*dbWorlds) {
			return nil, constant.CodeRoomImportWorldCountMismatch, fmt.Errorf("上传存档世界个数与当前房间不一致")
		}
		for index := range worlds {
			worlds[index].ServerPort = (*dbWorlds)[index].ServerPort
			worlds[index].MasterServerPort = (*dbWorlds)[index].MasterServerPort
			worlds[index].AuthenticationPort = (*dbWorlds)[index].AuthenticationPort
			worlds[index].RoomID = roomID
		}
	}

	// 判断是否为统一模组
	if len(worlds) >= 2 {
		if worlds[0].ModData == worlds[1].ModData {
			room.ModInOne = true
			room.ModData = worlds[0].ModData
			for index := range worlds {
				worlds[index].ModData = ""
			}
		} else {
			room.ModInOne = false
		}
	} else {
		room.ModInOne = true
	}

	// 写入数据库
	if newRoom {
		if _, err = s.roomDao.CreateRoom(&room); err != nil {
			logger.Logger.Errorf("创建房间失败, err: %v", err)
			return nil, constant.CodeRoomDBError, err
		}
		for index := range worlds {
			worlds[index].RoomID = room.ID
			if errCreateWorld := s.worldDao.Create(&worlds[index]); errCreateWorld != nil {
				logger.Logger.Errorf("创建房间失败, err: %v", errCreateWorld)
				return nil, constant.CodeRoomDBError, errCreateWorld
			}
		}

		roomSetting.RoomID = room.ID
		if errCreate := s.roomSettingDao.Create(&roomSetting); errCreate != nil {
			logger.Logger.Errorf("创建房间失败, err: %v", errCreate)
			return nil, constant.CodeRoomDBError, errCreate
		}
	} else {
		if err = s.roomDao.UpdateRoom(&room); err != nil {
			logger.Logger.Errorf("更新房间失败, err: %v", err)
			return nil, constant.CodeRoomDBError, err
		}
		if err = s.worldDao.UpdateWorlds(&worlds); err != nil {
			logger.Logger.Errorf("更新房间失败, err: %v", err)
			return nil, constant.CodeRoomDBError, err
		}
		// 不更新 roomSetting
	}

	game := dst.NewGameController(&room, &worlds, &roomSetting, actor.Lang)
	_ = game.StopAllWorld()
	if err = game.SaveAll(); err != nil {
		logger.Logger.Errorf("配置写入磁盘失败, err: %v", err)
		return nil, constant.CodeRoomWriteFileFail, err
	}

	clusterPath := fmt.Sprintf("%s/Cluster_%d", utils.ClusterPath, room.ID)

	// 设置三个名单
	if err = utils.TruncAndWriteFile(fmt.Sprintf("%s/adminlist.txt", clusterPath), uploadExtraInfo.adminlist); err != nil {
		logger.Logger.Errorf("设置管理员失败, err: %v", err)
	}
	if err = utils.TruncAndWriteFile(fmt.Sprintf("%s/blocklist.txt", clusterPath), uploadExtraInfo.blocklist); err != nil {
		logger.Logger.Errorf("设置黑名单失败, err: %v", err)
	}
	if err = utils.TruncAndWriteFile(fmt.Sprintf("%s/whitelist.txt", clusterPath), uploadExtraInfo.whitelist); err != nil {
		logger.Logger.Errorf("设置预留位失败, err: %v", err)
	}

	// 覆盖 save 目录
	for _, world := range uploadExtraInfo.worldPath {
		if err = utils.RemoveDir(fmt.Sprintf("%s/%s/save", clusterPath, world.name)); err != nil {
			logger.Logger.Errorf("删除旧存档数据失败, err: %v", err)
			continue
		}
		savePath := utils.ShellQuote(fmt.Sprintf("%s/save", world.path))
		worldPath := utils.ShellQuote(fmt.Sprintf("%s/%s/", clusterPath, world.name))
		cmd := fmt.Sprintf("cp -r %s %s", savePath, worldPath)
		logger.Logger.Debug(cmd)
		if err = utils.BashCMD(cmd); err != nil {
			logger.Logger.Errorf("复制存档数据失败, err: %v", err)
		}
	}

	if newRoom {
		// 新房间需要注册默认备份/重启等定时任务
		processJobs(game, room.ID, roomSetting)

		// 非管理员创建者需要把房间加入自己的房间权限列表，否则房间创建后不可见
		if !actor.IsAdmin() {
			if err = s.addRoomToUser(actor.Username, strconv.Itoa(room.ID)); err != nil {
				logger.Logger.Errorf("更新用户房间权限失败, err: %v", err)
				return nil, constant.CodeRoomDBError, err
			}
		}

		webhook.Snd.Send(webhook.EventRoomCreated, room.ID, XRoomTotalInfo{
			RoomData:        room,
			WorldData:       worlds,
			RoomSettingData: roomSetting,
		})
	} else {
		if err = s.aiManager.Reload(room.ID); err != nil {
			logger.Logger.Errorf("重载房间 AI 对话失败, roomID: %d, err: %v", room.ID, err)
		}
	}

	return &room, 0, nil
}

func handleUpload(savePath, unzipPath string, room *models.Room, worlds *[]models.World, roomSetting *models.RoomSetting, uploadExtraInfo *UploadExtraInfo) (int, error) {
	// 1. 解压上传的zip压缩包
	err := utils.Unzip(savePath, unzipPath)
	if err != nil {
		return constant.CodeRoomUnzipFail, err
	}

	// 2. 查找存档的home路径
	clusterDir, err := findClusterDir(unzipPath)
	if err != nil {
		return constant.CodeRoomClusterDirNotFound, err
	}

	// 3. 获取token
	clusterToken, err := utils.GetFileAllContent(fmt.Sprintf("%s/cluster_token.txt", clusterDir))
	if err != nil || clusterToken == "" {
		logger.Logger.Info("未发现饥荒令牌文件，使用默认令牌")
		room.Token = utils.GetDstToken()
	} else {
		room.Token = clusterToken
	}

	// 4. 读取adminlist.txt blocklist.txt whitelist.txt的路径
	adminlistPath := fmt.Sprintf("%s/adminlist.txt", clusterDir)
	// whitelist_slots 会在dst.save时设置
	if utils.FileDirectoryExists(adminlistPath) {
		adminlist, err := utils.GetFileAllContent(adminlistPath)
		if err == nil {
			uploadExtraInfo.adminlist = adminlist
		}
	}
	blocklistPath := fmt.Sprintf("%s/blocklist.txt", clusterDir)
	if utils.FileDirectoryExists(blocklistPath) {
		blocklist, err := utils.GetFileAllContent(blocklistPath)
		if err == nil {
			uploadExtraInfo.blocklist = blocklist
		}
	}
	whitelistPath := fmt.Sprintf("%s/whitelist.txt", clusterDir)
	if utils.FileDirectoryExists(whitelistPath) {
		whitelist, err := utils.GetFileAllContent(whitelistPath)
		if err == nil {
			uploadExtraInfo.whitelist = whitelist
		}
	}

	// 5. 读取cluster.ini
	clusterIniPath := fmt.Sprintf("%s/cluster.ini", clusterDir)
	if !utils.FileDirectoryExists(clusterIniPath) {
		return constant.CodeRoomClusterIniNotFound, fmt.Errorf("未发现 cluster.ini: %s", clusterIniPath)
	}
	clusterIni, err := parseIniToMap(clusterIniPath)
	if err != nil {
		return constant.CodeRoomClusterIniReadFail, err
	}
	if clusterIni["cluster_name"] == "" {
		return constant.CodeRoomClusterNameNotFound, fmt.Errorf("未发现房间名")
	}
	room.GameName = clusterIni["cluster_name"]
	room.Description = clusterIni["cluster_description"]
	if clusterIni["game_mode"] == "" {
		return constant.CodeRoomGameModeNotFound, fmt.Errorf("未发现游戏模式")
	}
	if !utils.IsValidGameMode(clusterIni["game_mode"]) {
		return constant.CodeRoomGameModeInvalid, fmt.Errorf("无效的游戏模式: %s", clusterIni["game_mode"])
	}
	room.GameMode = clusterIni["game_mode"]
	maxPlayer, err := strconv.Atoi(clusterIni["max_players"])
	if err != nil {
		logger.Logger.Info("玩家个数获取异常，设置为默认值6")
		room.MaxPlayer = 6
	} else {
		room.MaxPlayer = maxPlayer
	}
	if clusterIni["pvp"] == "" {
		logger.Logger.Info("玩家对战获取异常，设置为默认值关闭")
		room.Pvp = false
	} else {
		pvp, err := strconv.ParseBool(clusterIni["pvp"])
		if err != nil {
			logger.Logger.Info("玩家对战获取异常，设置为默认值关闭")
			room.Pvp = false
		} else {
			room.Pvp = pvp
		}
	}
	if clusterIni["vote_enabled"] == "" {
		logger.Logger.Info("玩家投票获取异常，设置为默认值关闭")
		room.Vote = false
	} else {
		vote, err := strconv.ParseBool(clusterIni["vote_enabled"])
		if err != nil {
			logger.Logger.Info("玩家投票获取异常，设置为默认值关闭")
			room.Vote = false
		} else {
			room.Vote = vote
		}
	}
	if clusterIni["pause_when_empty"] == "" {
		logger.Logger.Info("自动暂停获取异常，设置为默认值开启")
		room.PauseEmpty = true
	} else {
		pauseEmpty, err := strconv.ParseBool(clusterIni["pause_when_empty"])
		if err != nil {
			logger.Logger.Info("自动暂停获取异常，设置为默认值开启")
			room.PauseEmpty = false
		} else {
			room.PauseEmpty = pauseEmpty
		}
	}
	maxRollBack, err := strconv.Atoi(clusterIni["max_snapshots"])
	if err != nil {
		logger.Logger.Info("回档天数获取异常，设置为默认值10")
		room.MaxRollBack = 10
	} else {
		room.MaxRollBack = maxRollBack
	}
	room.Password = clusterIni["cluster_password"]
	if clusterIni["master_ip"] == "" {
		logger.Logger.Info("主世界IP获取异常，设置为默认值127.0.0.1")
		room.MasterIP = "127.0.0.1"
	} else {
		room.MasterIP = clusterIni["master_ip"]
	}
	if clusterIni["cluster_key"] == "" {
		logger.Logger.Info("世界认证密码获取异常，设置随机密码")
		room.ClusterKey = utils.RandomString(14)
	} else {
		room.ClusterKey = clusterIni["cluster_key"]
	}
	tickRate, err := strconv.Atoi(clusterIni["tick_rate"])
	if err != nil {
		logger.Logger.Info("tick rate获取异常，设置为默认值15")
		roomSetting.TickRate = 15
	} else {
		roomSetting.TickRate = tickRate
	}
	// steam 群组设置
	room.SteamGroupID = clusterIni["steam_group_id"]
	room.SteamGroupAdmins, err = strconv.ParseBool(clusterIni["steam_group_admins"])
	if err != nil {
		room.SteamGroupAdmins = false
	}
	room.SteamGroupOnly, err = strconv.ParseBool(clusterIni["steam_group_only"])
	if err != nil {
		room.SteamGroupOnly = false
	}

	// 6. 读取世界目录
	allWorldsPath, err := utils.GetDirs(clusterDir, false)
	if err != nil {
		return constant.CodeRoomWorldsDirInvalid, err
	}
	utils.ReverseSlice(allWorldsPath) // 让Master在Caves前面
	for _, i := range allWorldsPath {
		// 判断是否含有奇奇怪怪的目录，MacOS真是狗屎啊
		if strings.HasPrefix(i, "__") {
			continue
		}
		// worldPath会放入bash命令中，需要校验worldName
		if !isSafeWorldDir(i) {
			return constant.CodeRoomWorldsDirInvalid, fmt.Errorf("非法的世界目录名")
		}

		var (
			world     models.World
			worldPath WorldPath
		)
		worldPath.path = fmt.Sprintf("%s/%s", clusterDir, i)
		// 读取server.ini
		serverIniPath := fmt.Sprintf("%s/%s/server.ini", clusterDir, i)
		if !utils.FileDirectoryExists(serverIniPath) {
			return constant.CodeRoomServerIniNotFound, fmt.Errorf("未发现 server.ini: %s", serverIniPath)
		}
		serverIni, err := parseIniToMap(serverIniPath)
		if err != nil {
			return constant.CodeRoomServerIniReadFail, err
		}
		worldID, err := strconv.Atoi(serverIni["id"])
		if err != nil {
			logger.Logger.Info("世界ID获取异常，设置为默认值101")
			world.GameID = 101
		} else {
			world.GameID = worldID
		}
		if serverIni["is_master"] == "" {
			return constant.CodeRoomIsMasterNotFound, fmt.Errorf("未发现是否为主节点")
		}
		isMaster, err := strconv.ParseBool(serverIni["is_master"])
		if err != nil {
			return constant.CodeRoomIsMasterReadFail, err
		}
		world.IsMaster = isMaster
		// 世界名会被拼接进bash命令和文件路径，必须使用校验后的值
		worldName := serverIni["name"]
		if worldName == "" {
			// 兼容缺失name字段的存档，按主/从节点设置默认名
			if isMaster {
				logger.Logger.Info("世界名获取异常，设置为默认值Master")
				worldName = "Master"
			} else {
				logger.Logger.Info("世界名获取异常，设置为默认值Caves")
				worldName = "Caves"
			}
		}
		world.WorldName = worldName
		// worldPath.name用于拼接复制目标目录，而目标目录由world.WorldName创建，两者必须一致，
		// 否则存档会被复制到错误的目录，甚至覆盖其他世界的存档
		worldPath.name = worldName
		encodeUserPath, err := strconv.ParseBool(serverIni["encode_user_path"])
		if err != nil {
			logger.Logger.Info("获取encode_user_path失败，设置为默认值true")
			world.EncodeUserPath = true
		} else {
			world.EncodeUserPath = encodeUserPath
		}

		// 读取世界配置 leveldataoverride.lua(worldgenoverride.lua)
		levelDataPath := fmt.Sprintf("%s/%s/leveldataoverride.lua", clusterDir, i)
		levelData, err := utils.GetFileAllContent(levelDataPath)
		if err != nil {
			levelDataPath = fmt.Sprintf("%s/%s/worldgenoverride.lua", clusterDir, i)
			levelData, err = utils.GetFileAllContent(levelDataPath)
			if err != nil {
				return constant.CodeRoomLevelDataNotFound, fmt.Errorf("未发现世界配置")
			}
		}
		// 归一化世界配置，剥离 BOM / Klei 引擎头等非法前缀，避免前端世界配置页解析失败
		world.LevelData = utils.NormalizeLuaConfig(levelData)
		if world.LevelData != levelData {
			logger.Logger.Infof("世界配置含非法前缀(BOM/Klei头)，已自动归一化，世界目录: %s", i)
		}

		// 读取mod配置 modoverrides.lua
		modDataPath := fmt.Sprintf("%s/%s/modoverrides.lua", clusterDir, i)
		modData, err := utils.GetFileAllContent(modDataPath)
		if err == nil {
			world.ModData = utils.NormalizeLuaConfig(modData)
		}

		uploadExtraInfo.worldPath = append(uploadExtraInfo.worldPath, worldPath)
		*worlds = append(*worlds, world)
	}

	// 7. 校验存档中的世界名，非法或重复时拒绝导入，避免命令注入和存档互相覆盖
	if worldName, code := validateWorldNames(*worlds); code != 0 {
		logger.Logger.Warnf("存档世界名校验失败，拒绝导入, world: %s, code: %d", worldName, code)
		if code == constant.CodeRoomDuplicateWorldName {
			return constant.CodeRoomDuplicateWorldNameInSave, fmt.Errorf("存档中存在重复的世界名: %s", worldName)
		}
		return constant.CodeRoomInvalidWorldNameInSave, fmt.Errorf("存档中存在非法的世界名: %s", worldName)
	}

	return 0, nil
}

// 查找包含 cluster.ini 文件的目录
func findClusterDir(path string) (string, error) {
	// 检查当前目录是否包含 cluster.ini
	clusterFile := filepath.Join(path, "cluster.ini")
	if _, err := os.Stat(clusterFile); err == nil {
		return path, nil
	}

	// 读取当前目录
	entries, err := os.ReadDir(path)
	if err != nil {
		return "", fmt.Errorf("读取目录失败: %v", err)
	}

	// 遍历子目录
	for _, entry := range entries {
		logger.Logger.Debug(entry.Name())
		if entry.IsDir() {
			subPath := filepath.Join(path, entry.Name())
			// 递归查找
			if result, err := findClusterDir(subPath); err == nil {
				return result, nil
			}
		}
	}

	return "", fmt.Errorf("未找到包含 cluster.ini 的目录")
}

func isSafeWorldDir(s string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9/]+$`, s)

	return matched
}

// 将ini文件读取为map
func parseIniToMap(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	configMap := make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// 检查是否是节标题
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			continue
		}

		// 解析键值对
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			configMap[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return configMap, nil
}
