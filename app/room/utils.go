package room

import (
	"bufio"
	"dst-management-platform-api/aichat"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	roomDao          *dao.RoomDAO
	userDao          *dao.UserDAO
	worldDao         *dao.WorldDAO
	roomSettingDao   *dao.RoomSettingDAO
	favoriteDao      *dao.FavoriteDAO
	globalSettingDao *dao.GlobalSettingDAO
	uidMapDao        *dao.UidMapDAO
	roomAISettingDao *dao.RoomAISettingDAO
	aiManager        *aichat.Manager
}

func NewHandler(userDao *dao.UserDAO, roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO, favoriteDao *dao.FavoriteDAO, globalSettingDao *dao.GlobalSettingDAO, uidMapDao *dao.UidMapDAO, roomAISettingDao *dao.RoomAISettingDAO, aiManager *aichat.Manager) *Handler {
	return &Handler{
		roomDao:          roomDao,
		userDao:          userDao,
		worldDao:         worldDao,
		roomSettingDao:   roomSettingDao,
		favoriteDao:      favoriteDao,
		globalSettingDao: globalSettingDao,
		uidMapDao:        uidMapDao,
		roomAISettingDao: roomAISettingDao,
		aiManager:        aiManager,
	}
}

type Partition struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"pageSize" form:"pageSize"`
}

type XRoomWorld struct {
	models.Room
	Worlds  []models.World  `json:"worlds"`
	Players []cache.Players `json:"players"`
}

type XRoomTotalInfo struct {
	RoomData        models.Room        `json:"roomData"`
	WorldData       []models.World     `json:"worldData"`
	RoomSettingData models.RoomSetting `json:"roomSettingData"`
}

// validateWorldNames 校验世界名是否合法
// 世界名会被拼接进bash命令和文件路径，因此必须限定字符集；
// 同时同一房间内世界名不能重复，否则多个世界会使用同一个目录、互相覆盖存档。
// 返回第一个不合法的世界名和对应提示文案的i18n key，全部合法时返回空字符串
func validateWorldNames(worlds []models.World) (string, string) {
	seen := make(map[string]struct{}, len(worlds))
	for _, world := range worlds {
		name := world.WorldName
		if !utils.IsSafeString(name) {
			return name, "invalid world name"
		}
		if _, ok := seen[name]; ok {
			return name, "duplicate world name"
		}
		seen[name] = struct{}{}
	}

	return "", ""
}

// 是否拥有房间创建权限
func (h *Handler) hasCreatePermission(c *gin.Context) (bool, error) {
	role, _ := c.Get("role")
	username, _ := c.Get("username")
	var (
		has    bool
		err    error
		dbUser *models.User
	)

	// 管理员直接返回true
	if role.(string) == "admin" {
		has = true
	} else {
		dbUser, err = h.userDao.GetUserByUsername(username.(string))
		if err != nil {
			return has, err
		}
		if dbUser.RoomCreation {
			has = true
		}
	}

	return has, err
}

// 是否拥有对应房间权限
func (h *Handler) hasRoomPermission(c *gin.Context, roomID string) bool {
	role, _ := c.Get("role")
	username, _ := c.Get("username")

	// 管理员直接返回true
	if role.(string) == "admin" {
		return true
	} else {
		dbUser, err := h.userDao.GetUserByUsername(username.(string))
		if err != nil {
			logger.Logger.Error("查询数据库失败")
			return false
		}
		roomIDs := strings.Split(dbUser.Rooms, ",")
		for _, id := range roomIDs {
			if id == roomID {
				return true
			}
		}
	}

	return false
}

// 处理定时任务
func processJobs(game *dst.Game, roomID int, roomSetting models.RoomSetting) {
	// 备份 //
	backupNames := scheduler.GetJobsByType(roomID, "Backup")
	type BackupSetting struct {
		Time string `json:"time"`
	}
	var backupSettings []BackupSetting
	if err := json.Unmarshal([]byte(roomSetting.BackupSetting), &backupSettings); err != nil {
		logger.Logger.Errorf("获取房间备份设置失败, err: %v", err)
	} else {
		if roomSetting.BackupEnable {
			if len(backupSettings) >= len(backupNames) {
				// 新设置长度大于旧设置，直接更新
				for i, s := range backupSettings {
					err := scheduler.UpdateJob(&scheduler.JobConfig{
						Name:     fmt.Sprintf("%d-%d-Backup", roomID, i),
						Func:     scheduler.Backup,
						Args:     []any{game},
						TimeType: scheduler.DayType,
						Interval: 0,
						DayAt:    s.Time,
					})
					if err != nil {
						logger.Logger.Errorf("备份定时任务处理失败, err: %v", err)
					}
				}
			} else {
				// 新设置长度小于旧设置，超出的删除
				for i, jobName := range backupNames {
					if i >= len(backupSettings) {
						scheduler.DeleteJob(jobName)
					} else {
						err := scheduler.UpdateJob(&scheduler.JobConfig{
							Name:     fmt.Sprintf("%d-%d-Backup", roomID, i),
							Func:     scheduler.Backup,
							Args:     []any{game},
							TimeType: scheduler.DayType,
							Interval: 0,
							DayAt:    backupSettings[i].Time,
						})
						if err != nil {
							logger.Logger.Errorf("备份定时任务处理失败, err: %v", err)
						}
					}
				}
			}
		} else {
			// 删除所有备份任务
			for _, jobName := range backupNames {
				scheduler.DeleteJob(jobName)
			}
		}
	}

	// 备份清理 //
	if roomSetting.BackupCleanEnable {
		err := scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     fmt.Sprintf("%d-BackupClean", roomID),
			Func:     scheduler.BackupClean,
			Args:     []any{roomID, roomSetting.BackupCleanSetting},
			TimeType: scheduler.DayType,
			Interval: 0,
			DayAt:    "05:16:27",
		})
		if err != nil {
			logger.Logger.Errorf("备份清理定时任务处理失败, err: %v", err)
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-BackupClean", roomID))
	}

	// 重启 //
	if roomSetting.RestartEnable {
		err := scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     fmt.Sprintf("%d-Restart", roomID),
			Func:     scheduler.Restart,
			Args:     []any{game},
			TimeType: scheduler.DayType,
			Interval: 0,
			DayAt:    roomSetting.RestartSetting,
		})
		if err != nil {
			logger.Logger.Errorf("自动重启定时任务处理失败, err: %v", err)
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-Restart", roomID))
	}
	// 重置 //
	if roomSetting.ResetEnable {
		var resetSetting struct {
			Force bool   `json:"force"`
			Time  string `json:"time"`
			Days  int    `json:"days"`
		}
		if err := json.Unmarshal([]byte(roomSetting.ResetSetting), &resetSetting); err != nil {
			logger.Logger.Errorf("获取自动重置设置失败, err: %v", err)
		} else {
			err := scheduler.UpdateJob(&scheduler.JobConfig{
				Name:     fmt.Sprintf("%d-Reset", roomID),
				Func:     scheduler.Reset,
				Args:     []any{game, roomID, resetSetting.Force, resetSetting.Days},
				TimeType: scheduler.DayType,
				Interval: 0,
				DayAt:    resetSetting.Time,
			})
			if err != nil {
				logger.Logger.Errorf("自动重置定时任务处理失败, err: %v", err)
			}
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-Reset", roomID))
	}
	// 自动开启关闭游戏
	if roomSetting.ScheduledStartStopEnable {
		type ScheduledStartStopSetting struct {
			Start string `json:"start"`
			Stop  string `json:"stop"`
		}
		var scheduledStartStopSetting ScheduledStartStopSetting
		if err := json.Unmarshal([]byte(roomSetting.ScheduledStartStopSetting), &scheduledStartStopSetting); err != nil {
			logger.Logger.Errorf("获取自动开启关闭游戏设置失败, err: %v", err)
		} else {
			err := scheduler.UpdateJob(&scheduler.JobConfig{
				Name:     fmt.Sprintf("%d-ScheduledStart", roomID),
				Func:     scheduler.ScheduledStart,
				Args:     []any{game},
				TimeType: scheduler.DayType,
				Interval: 0,
				DayAt:    scheduledStartStopSetting.Start,
			})
			if err != nil {
				logger.Logger.Errorf("自动开启游戏任务处理失败, err: %v", err)
			}
			err = scheduler.UpdateJob(&scheduler.JobConfig{
				Name:     fmt.Sprintf("%d-ScheduledStop", roomID),
				Func:     scheduler.ScheduledStop,
				Args:     []any{game},
				TimeType: scheduler.DayType,
				Interval: 0,
				DayAt:    scheduledStartStopSetting.Stop,
			})
			if err != nil {
				logger.Logger.Errorf("自动关闭游戏任务处理失败, err: %v", err)
			}
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-ScheduledStart", roomID))
		scheduler.DeleteJob(fmt.Sprintf("%d-ScheduledStop", roomID))
	}
	// 自动保活 //
	if roomSetting.KeepaliveEnable {
		err := scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     fmt.Sprintf("%d-Keepalive", roomID),
			Func:     scheduler.Keepalive,
			Args:     []any{game, roomID},
			TimeType: scheduler.MinuteType,
			Interval: roomSetting.KeepaliveSetting,
			DayAt:    "",
		})
		if err != nil {
			logger.Logger.Errorf("自动保活定时任务处理失败, err: %v", err)
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-Keepalive", roomID))
	}
	// 房间资源监控 //
	if roomSetting.RoomMetricsEnable {
		err := scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     fmt.Sprintf("%d-RoomMetrics", roomID),
			Func:     scheduler.RoomMetricsGet,
			Args:     []any{game, roomID},
			TimeType: scheduler.MinuteType,
			Interval: scheduler.RoomMetricsGetInInterval,
			DayAt:    "",
		})
		if err != nil {
			logger.Logger.Errorf("房间资源监控定时任务处理失败, err: %v", err)
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-RoomMetrics", roomID))
	}
	// 玩家更新模组
	if roomSetting.PlayerUpdateModEnable {
		errFalse := scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     fmt.Sprintf("%d-PlayerUpdateModFalse", roomID),
			Func:     scheduler.PlayerUpdateMod,
			Args:     []any{game, false},
			TimeType: scheduler.SecondType,
			Interval: scheduler.PlayerUpdateModFalseInterval,
			DayAt:    "",
		})
		if errFalse != nil {
			logger.Logger.Errorf("玩家更新模组定时任务处理失败, err: %v", errFalse)
		}
		errTrue := scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     fmt.Sprintf("%d-PlayerUpdateModTrue", roomID),
			Func:     scheduler.PlayerUpdateMod,
			Args:     []any{game, true},
			TimeType: scheduler.SecondType,
			Interval: scheduler.PlayerUpdateModTrueInterval,
			DayAt:    "",
		})
		if errTrue != nil {
			logger.Logger.Errorf("玩家更新模组定时任务处理失败, err: %v", errTrue)
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-PlayerUpdateModFalse", roomID))
		scheduler.DeleteJob(fmt.Sprintf("%d-PlayerUpdateModTrue", roomID))
	}
}

// worldIDByName 世界名到世界ID的映射，世界名是房间内世界的唯一标识
func worldIDByName(worlds []models.World) map[string]int {
	worldIDs := make(map[string]int, len(worlds))
	for _, world := range worlds {
		worldIDs[world.WorldName] = world.ID
	}
	return worldIDs
}

// deleteWorldMetrics 删除世界监控数据
func deleteWorldMetrics(worldIDMaps ...map[string]int) {
	cache.WorldMetricsMutex.Lock()
	defer cache.WorldMetricsMutex.Unlock()

	for _, worldIDMap := range worldIDMaps {
		for _, worldID := range worldIDMap {
			delete(cache.WorldMetrics, worldID)
		}
	}
}

// migrateWorldMetrics 房间保存时按世界名迁移世界监控数据
// 房间保存会删除并重建世界（世界ID发生变化），直接按ID清理会丢失已有的监控历史
func migrateWorldMetrics(oldWorldIDMap, newWorldIDMap map[string]int) {
	cache.WorldMetricsMutex.Lock()
	defer cache.WorldMetricsMutex.Unlock()

	// 先取出旧世界的数据，避免新旧世界ID交叉时互相覆盖
	oldMetrics := make(map[string][]cache.WorldMetricsData, len(oldWorldIDMap))
	for worldName, oldWorldID := range oldWorldIDMap {
		if metrics, ok := cache.WorldMetrics[oldWorldID]; ok {
			oldMetrics[worldName] = metrics
		}
	}

	// 清理该房间全部旧世界的数据
	for _, oldWorldID := range oldWorldIDMap {
		delete(cache.WorldMetrics, oldWorldID)
	}

	// 按世界名写回到新的世界ID，已删除的世界不再写入
	for worldName, newWorldID := range newWorldIDMap {
		if metrics, ok := oldMetrics[worldName]; ok {
			cache.WorldMetrics[newWorldID] = metrics
		}
	}
}

func handleUpload(savePath, unzipPath string, room *models.Room, worlds *[]models.World, roomSetting *models.RoomSetting, uploadExtraInfo *UploadExtraInfo) (string, error) {
	// 1. 解压上传的zip压缩包
	err := utils.Unzip(savePath, unzipPath)
	if err != nil {
		return "unzip fail", err
	}

	// 2. 查找存档的home路径
	clusterDir, err := findClusterDir(unzipPath)
	if err != nil {
		return "find cluster home fail", err
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
		return "cluster.ini file not found", err
	}
	clusterIni, err := parseIniToMap(clusterIniPath)
	if err != nil {
		return "read cluster.ini file fail", err
	}
	if clusterIni["cluster_name"] == "" {
		return "cluster.ini cluster_name not found", fmt.Errorf("未发现房间名")
	}
	room.GameName = clusterIni["cluster_name"]
	room.Description = clusterIni["cluster_description"]
	if clusterIni["game_mode"] == "" {
		return "cluster.ini game_mode not found", fmt.Errorf("未发现游戏模式")
	}
	if !utils.IsValidGameMode(clusterIni["game_mode"]) {
		return "cluster.ini game_mode invalid", fmt.Errorf("无效的游戏模式: %s", clusterIni["game_mode"])
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
		return "get worlds path fail", err
	}
	utils.ReverseSlice(allWorldsPath) // 让Master在Caves前面
	for _, i := range allWorldsPath {
		// 判断是否含有奇奇怪怪的目录，MacOS真是狗屎啊
		if strings.HasPrefix(i, "__") {
			continue
		}
		// worldPath会放入bash命令中，需要校验worldName
		if !isSafeWorldDir(i) {
			return "get worlds path fail", fmt.Errorf("非法的世界目录名")
		}

		var (
			world     models.World
			worldPath WorldPath
		)
		worldPath.path = fmt.Sprintf("%s/%s", clusterDir, i)
		// 读取server.ini
		serverIniPath := fmt.Sprintf("%s/%s/server.ini", clusterDir, i)
		if !utils.FileDirectoryExists(serverIniPath) {
			return "server.ini file not found", err
		}
		serverIni, err := parseIniToMap(serverIniPath)
		if err != nil {
			return "read server.ini file fail", err
		}
		worldID, err := strconv.Atoi(serverIni["id"])
		if err != nil {
			logger.Logger.Info("世界ID获取异常，设置为默认值101")
			world.GameID = 101
		} else {
			world.GameID = worldID
		}
		if serverIni["is_master"] == "" {
			return "server.ini is_master not found", fmt.Errorf("未发现是否为主节点")
		}
		isMaster, err := strconv.ParseBool(serverIni["is_master"])
		if err != nil {
			return "read is_master from server.ini fail", err
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
				return "level data not found", fmt.Errorf("未发现世界配置")
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
	if worldName, msgKey := validateWorldNames(*worlds); msgKey != "" {
		logger.Logger.Warnf("存档世界名校验失败，拒绝导入, world: %s, reason: %s", worldName, msgKey)
		if msgKey == "duplicate world name" {
			return "duplicate world name in save", fmt.Errorf("存档中存在重复的世界名: %s", worldName)
		}
		return "invalid world name in save", fmt.Errorf("存档中存在非法的世界名: %s", worldName)
	}

	return "", nil
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

type WorldPath struct {
	name string
	path string
}

type UploadExtraInfo struct {
	adminlist string
	blocklist string
	whitelist string
	worldPath []WorldPath
}

// checkGamePort 检测端口是否冲突，excludeRoomID 为 0 时不排除任何房间
// 先检测入参内部是否重复，再检测与数据库中已有的世界端口是否冲突
// MasterPort（[SHARD] master_port）不参与检测，允许重复
// 返回冲突的端口，0 表示无冲突
func (h *Handler) checkGamePort(ports []int, excludeRoomID int) int {
	// 1. 检测入参端口自身是否重复
	seen := make(map[int]bool)
	for _, port := range ports {
		if seen[port] {
			return port
		}
		seen[port] = true
	}

	// 2. 获取所有世界端口
	worlds, err := h.worldDao.Query(1, 10000, nil)
	if err != nil {
		logger.Logger.Errorf("checkGamePort 查询世界失败, err: %v", err)
		return 0
	}

	// 3. 已使用的端口集合
	usedPorts := make(map[int]bool)
	for _, world := range worlds.Data {
		if world.RoomID != excludeRoomID {
			usedPorts[world.ServerPort] = true
			usedPorts[world.MasterServerPort] = true
			usedPorts[world.AuthenticationPort] = true
		}
	}

	// 4. 检查与已有端口是否冲突
	for _, port := range ports {
		if usedPorts[port] {
			return port
		}
	}

	return 0
}

const FavoriteTypeDST = "dst"

// FavoriteItem 收藏接口返回结构，Data 解析为 JSON 对象
type FavoriteItem struct {
	ID        string         `json:"id"`
	Username  string         `json:"username"`
	Type      string         `json:"type"`
	Data      XRoomTotalInfo `json:"data"`
	Remark    string         `json:"remark"`
	CreatedAt int64          `json:"createdAt"`
	UpdatedAt int64          `json:"updatedAt"`
}

type favoriteCreateReq struct {
	RoomID int    `json:"roomID" binding:"required"`
	Remark string `json:"remark"`
}

type favoriteUpdateReq struct {
	ID     string  `json:"id" binding:"required"`
	Remark *string `json:"remark" binding:"required"`
}

type favoriteDeleteReq struct {
	ID string `json:"id" binding:"required"`
}

type favoriteListReq struct {
	Partition
}

type favoriteApplyReq struct {
	ID       string `json:"id" binding:"required"`
	GameName string `json:"gameName"`
}

// RoomSnapshot 生成 XRoomTotalInfo 房间快照，清空本地关联 ID，保留 token。
// 收藏与云存档元数据共用该结构，保证云端看到的数据形状一致。
func RoomSnapshot(room *models.Room, worlds *[]models.World, setting *models.RoomSetting) XRoomTotalInfo {
	roomData := *room
	roomData.ID = 0

	worldData := make([]models.World, len(*worlds))
	for i, world := range *worlds {
		world.ID = 0
		world.RoomID = 0
		world.LastAliveTime = ""
		worldData[i] = world
	}

	roomSetting := *setting
	roomSetting.RoomID = 0

	return XRoomTotalInfo{
		RoomData:        roomData,
		WorldData:       worldData,
		RoomSettingData: roomSetting,
	}
}

// ValidateFavoriteData 校验收藏数据，返回 i18n key 和格式化参数。
// 云同步下载的快照同样必须经过该函数校验。
func ValidateFavoriteData(data *XRoomTotalInfo) (string, string) {
	if len(data.WorldData) == 0 {
		return "favorite no worlds", ""
	}

	if worldName, msgKey := validateWorldNames(data.WorldData); msgKey != "" {
		return msgKey, worldName
	}

	if !utils.IsValidGameMode(data.RoomData.GameMode) {
		return "favorite invalid game mode", ""
	}
	if data.RoomData.GameMode == "custom" && data.RoomData.CustomGameMode != "" && !utils.IsValidGameMode(data.RoomData.CustomGameMode) {
		return "favorite invalid game mode", ""
	}

	if strings.TrimSpace(data.RoomSettingData.WebhookSetting) != "" {
		var webhooks []webhook.WebhookItem
		if err := json.Unmarshal([]byte(data.RoomSettingData.WebhookSetting), &webhooks); err != nil {
			return "favorite invalid webhook", ""
		}
		for _, item := range webhooks {
			if !utils.IsValidURL(item.URL) {
				return "invalid url", ""
			}
		}
	}

	return "", ""
}

// ClearFavoriteCustomStartupCmd 清空自定义启动命令，返回是否有内容被清空。
func ClearFavoriteCustomStartupCmd(data *XRoomTotalInfo) bool {
	cleared := false
	for i := range data.WorldData {
		if data.WorldData[i].CustomStartupCmd != "" {
			data.WorldData[i].CustomStartupCmd = ""
			cleared = true
		}
	}

	return cleared
}

func favoriteItemFromModel(favorite *models.Favorite) (*FavoriteItem, error) {
	var data XRoomTotalInfo
	if err := json.Unmarshal([]byte(favorite.Data), &data); err != nil {
		return nil, err
	}

	item := favoriteItemFromData(favorite, data)
	return &item, nil
}

func favoriteItemFromData(favorite *models.Favorite, data XRoomTotalInfo) FavoriteItem {
	return FavoriteItem{
		ID:        favorite.ID,
		Username:  favorite.Username,
		Type:      favorite.Type,
		Data:      data,
		Remark:    favorite.Remark,
		CreatedAt: favorite.CreatedAt,
		UpdatedAt: favorite.UpdatedAt,
	}
}

func favoriteMessage(c *gin.Context, msgKey, arg string) string {
	if arg == "" {
		return message.Get(c, msgKey)
	}

	return message.GetF(c, msgKey, arg)
}

// ===== 存档导入 =====

// ImportNewRoomFromZip 从已经保存到磁盘的 zip 导入一个新房间。
// savePath 是 zip 文件路径，unzipPath 是解压目录（需要以路径分隔符结尾）。
// 返回 room、业务响应 code、失败时的 i18n key 和原始错误。
func (h *Handler) ImportNewRoomFromZip(c *gin.Context, savePath, unzipPath string) (*models.Room, int, string, error) {
	return h.importRoomFromZip(c, savePath, unzipPath, 0, true)
}

func (h *Handler) importRoomFromZip(c *gin.Context, savePath, unzipPath string, roomID int, newRoom bool) (*models.Room, int, string, error) {

	var (
		room            models.Room
		worlds          []models.World
		roomSetting     models.RoomSetting
		uploadExtraInfo UploadExtraInfo
	)

	errMsg, err := handleUpload(savePath, unzipPath, &room, &worlds, &roomSetting, &uploadExtraInfo)
	if err != nil {
		logger.Logger.Errorf("处理上传文件失败, err: %v", err)
		return nil, 201, errMsg, err
	}

	if len(uploadExtraInfo.worldPath) == 0 {
		return nil, 201, "no available worlds found", fmt.Errorf("存档中没有可用世界")
	}

	// 设置所有的port和roomSetting
	if newRoom {
		room.Status = true
		roomCount, err := h.roomDao.Count(nil)
		if err != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", err)
			return nil, 500, "database error", err
		}

		worldCount, err := h.worldDao.Count(nil)
		if err != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", err)
			return nil, 500, "database error", err
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
		dbRoom, err := h.roomDao.GetRoomByID(roomID)
		if err != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", err)
			return nil, 500, "database error", err
		}
		room.MasterPort = dbRoom.MasterPort
		room.ID = roomID

		dbWorlds, err := h.worldDao.GetWorldsByRoomID(roomID)
		if err != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", err)
			return nil, 500, "database error", err
		}
		if len(worlds) != len(*dbWorlds) {
			return nil, 201, "number of worlds does not match", fmt.Errorf("上传存档世界个数与当前房间不一致")
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
		if _, err = h.roomDao.CreateRoom(&room); err != nil {
			logger.Logger.Errorf("创建房间失败, err: %v", err)
			return nil, 500, "database error", err
		}
		for index := range worlds {
			worlds[index].RoomID = room.ID
			if errCreateWorld := h.worldDao.Create(&worlds[index]); errCreateWorld != nil {
				logger.Logger.Errorf("创建房间失败, err: %v", errCreateWorld)
				return nil, 500, "database error", errCreateWorld
			}
		}

		roomSetting.RoomID = room.ID
		if errCreate := h.roomSettingDao.Create(&roomSetting); errCreate != nil {
			logger.Logger.Errorf("创建房间失败, err: %v", errCreate)
			return nil, 500, "database error", errCreate
		}
	} else {
		if err = h.roomDao.UpdateRoom(&room); err != nil {
			logger.Logger.Errorf("更新房间失败, err: %v", err)
			return nil, 500, "database error", err
		}
		if err = h.worldDao.UpdateWorlds(&worlds); err != nil {
			logger.Logger.Errorf("更新房间失败, err: %v", err)
			return nil, 500, "database error", err
		}
		// 不更新 roomSetting
	}

	game := dst.NewGameController(&room, &worlds, &roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	_ = game.StopAllWorld()
	if err = game.SaveAll(); err != nil {
		logger.Logger.Errorf("配置写入磁盘失败, err: %v", err)
		return nil, 201, "write file fail", err
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
		if c.GetString("role") != "admin" {
			if err = h.addRoomToUser(c.GetString("username"), strconv.Itoa(room.ID)); err != nil {
				logger.Logger.Errorf("更新用户房间权限失败, err: %v", err)
				return nil, 500, "database error", err
			}
		}

		webhook.Snd.Send(webhook.EventRoomCreated, room.ID, XRoomTotalInfo{
			RoomData:        room,
			WorldData:       worlds,
			RoomSettingData: roomSetting,
		})
	} else {
		if err = h.aiManager.Reload(room.ID); err != nil {
			logger.Logger.Errorf("重载房间 AI 对话失败, roomID: %d, err: %v", room.ID, err)
		}
	}

	return &room, 200, "", nil
}
