package room

import (
	"bufio"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	roomDao          *dao.RoomDAO
	userDao          *dao.UserDAO
	worldDao         *dao.WorldDAO
	roomSettingDao   *dao.RoomSettingDAO
	globalSettingDao *dao.GlobalSettingDAO
	uidMapDao        *dao.UidMapDAO
}

func NewHandler(userDao *dao.UserDAO, roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO, globalSettingDao *dao.GlobalSettingDAO, uidMapDao *dao.UidMapDAO) *Handler {
	return &Handler{
		roomDao:          roomDao,
		userDao:          userDao,
		worldDao:         worldDao,
		roomSettingDao:   roomSettingDao,
		globalSettingDao: globalSettingDao,
		uidMapDao:        uidMapDao,
	}
}

type Partition struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"pageSize" form:"pageSize"`
}

type XRoomWorld struct {
	models.Room
	Worlds  []models.World `json:"worlds"`
	Players []db.Players   `json:"players"`
}

type XRoomTotalInfo struct {
	RoomData        models.Room        `json:"roomData"`
	WorldData       []models.World     `json:"worldData"`
	RoomSettingData models.RoomSetting `json:"roomSettingData"`
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
		logger.Logger.Error("获取房间备份设置失败", "err", err)
	}
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
					logger.Logger.Error("备份定时任务处理失败", "err", err)
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
						logger.Logger.Error("备份定时任务处理失败", "err", err)
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
			logger.Logger.Error("备份清理定时任务处理失败", "err", err)
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
			logger.Logger.Error("重启定时任务处理失败", "err", err)
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-Restart", roomID))
	}
	// 自动开启关闭游戏
	if roomSetting.ScheduledStartStopEnable {
		type ScheduledStartStopSetting struct {
			Start string `json:"start"`
			Stop  string `json:"stop"`
		}
		var scheduledStartStopSetting ScheduledStartStopSetting
		if err := json.Unmarshal([]byte(roomSetting.ScheduledStartStopSetting), &scheduledStartStopSetting); err != nil {
			logger.Logger.Error("获取自动开启关闭游戏设置失败", "err", err)
		}
		err := scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     fmt.Sprintf("%d-ScheduledStart", roomID),
			Func:     scheduler.ScheduledStart,
			Args:     []any{game},
			TimeType: scheduler.DayType,
			Interval: 0,
			DayAt:    scheduledStartStopSetting.Start,
		})
		if err != nil {
			logger.Logger.Error("自动开启游戏任务处理失败", "err", err)
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
			logger.Logger.Error("自动关闭游戏任务处理失败", "err", err)
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
			logger.Logger.Error("自动保活定时任务处理失败", "err", err)
		}
	} else {
		scheduler.DeleteJob(fmt.Sprintf("%d-Keepalive", roomID))
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
		if serverIni["name"] == "" {
			if isMaster {
				logger.Logger.Info("世界名获取异常，设置为默认值Master")
				world.WorldName = "Master"
				worldPath.name = "Master"
			} else {
				logger.Logger.Info("世界名获取异常，设置为默认值Caves")
				world.WorldName = "Caves"
				worldPath.name = "Caves"
			}
		} else {
			world.WorldName = serverIni["name"]
			worldPath.name = serverIni["name"]
		}
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
		world.LevelData = levelData

		// 读取mod配置 modoverrides.lua
		modDataPath := fmt.Sprintf("%s/%s/modoverrides.lua", clusterDir, i)
		modData, err := utils.GetFileAllContent(modDataPath)
		if err == nil {
			world.ModData = modData
		}

		uploadExtraInfo.worldPath = append(uploadExtraInfo.worldPath, worldPath)
		*worlds = append(*worlds, world)
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
