package room

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"strconv"
	"strings"
)

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
