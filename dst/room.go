package dst

import (
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	lua "github.com/yuin/gopher-lua"
)

type roomSaveData struct {
	// dir
	clusterName string
	clusterPath string
	// file
	clusterIniPath      string
	clusterTokenTxtPath string
}

type SeasonLength struct {
	Summer int `json:"summer"`
	Autumn int `json:"autumn"`
	Spring int `json:"spring"`
	Winter int `json:"winter"`
}

type RoomSessionInfo struct {
	Cycles       int          `json:"cycles"`
	Phase        string       `json:"phase"`
	Season       string       `json:"season"`
	ElapsedDays  int          `json:"elapsedDays"`
	SeasonLength SeasonLength `json:"seasonLength"`
}

func (g *Game) createRoom() error {
	g.roomMutex.Lock()
	defer g.roomMutex.Unlock()

	var err error

	err = utils.EnsureDirExists(g.clusterPath)
	if err != nil {
		return err
	}

	err = utils.TruncAndWriteFile(g.clusterIniPath, g.getClusterIni())
	if err != nil {
		return err
	}

	err = utils.TruncAndWriteFile(g.clusterTokenTxtPath, g.room.Token)
	if err != nil {
		return err
	}

	// 创建备份目录
	_ = utils.EnsureDirExists(fmt.Sprintf("%s/backup/%d", utils.DmpFiles, g.room.ID))

	return nil
}

func (g *Game) getClusterIni() string {
	var (
		gameMode          string
		lang              string
		steamGroupSetting string
	)

	switch g.room.GameMode {
	case "relaxed":
		gameMode = "survival"
	case "wilderness":
		gameMode = "survival"
	case "lightsOut":
		gameMode = "survival"
	case "custom":
		gameMode = g.room.CustomGameMode
	default:
		gameMode = g.room.GameMode
	}

	switch g.lang {
	case "zh":
		lang = "zh"
	case "en":
		lang = "en"
	default:
		lang = "zh"
	}

	if g.room.SteamGroupID != "" {
		steamGroupSetting = `

[STEAM]
steam_group_admins = ` + strconv.FormatBool(g.room.SteamGroupAdmins) + `
steam_group_id = ` + g.room.SteamGroupID + `
steam_group_only = ` + strconv.FormatBool(g.room.SteamGroupOnly) + `
`
	}

	contents := `[GAMEPLAY]
game_mode = ` + gameMode + `
max_players = ` + strconv.Itoa(g.room.MaxPlayer) + `
pvp = ` + strconv.FormatBool(g.room.Pvp) + `
pause_when_empty = ` + strconv.FormatBool(g.room.PauseEmpty) + `
vote_enabled = ` + strconv.FormatBool(g.room.Vote) + `
vote_kick_enabled = ` + strconv.FormatBool(g.room.Vote) + `

[NETWORK]
lan_only_cluster = ` + strconv.FormatBool(g.room.Lan) + `
offline_cluster = ` + strconv.FormatBool(g.room.Offline) + `
cluster_description = ` + g.room.Description + `
whitelist_slots = ` + strconv.Itoa(len(g.whitelist)) + `
cluster_name = ` + g.room.GameName + `
cluster_password = ` + g.room.Password + `
cluster_language = ` + lang + `
tick_rate = ` + strconv.Itoa(g.setting.TickRate) + `

[MISC]
console_enabled = true
max_snapshots = ` + strconv.Itoa(g.room.MaxRollBack) + `

[SHARD]
shard_enabled = true
bind_ip = 0.0.0.0
master_ip = ` + g.room.MasterIP + `
master_port = ` + strconv.Itoa(g.room.MasterPort) + `
cluster_key = ` + g.room.ClusterKey + steamGroupSetting

	logger.Logger.Debug(contents)

	return contents
}

func (g *Game) reset(force bool) error {
	if force {
		defer func() {
			_ = g.startAllWorld()
		}()

		err := g.stopAllWorld()
		if err != nil {
			return err
		}

		allSuccess := true

		for _, world := range g.worldSaveData {
			err = utils.RemoveDir(world.savePath)
			if err != nil {
				allSuccess = false
				logger.Logger.Error("删除存档文件失败", "err", err)
			}
		}

		if allSuccess {
			return nil
		} else {
			return fmt.Errorf("删除存档文件失败")
		}

	} else {
		resetCmd := fmt.Sprintf("c_regenerateworld()")
		return utils.ScreenCMD(resetCmd, g.worldSaveData[0].screenName)
	}
}

func (g *Game) announce(message string) error {
	s := strings.ReplaceAll(message, "'", "")
	s = strings.ReplaceAll(s, "\"", "")
	cmd := fmt.Sprintf("c_announce('%s')", s)
	for _, world := range g.worldSaveData {
		err := utils.ScreenCMD(cmd, world.screenName)
		if err == nil {
			return err
		}
	}

	return fmt.Errorf("执行失败")
}

func (g *Game) systemMsg(message string) error {
	s := strings.ReplaceAll(message, "'", "")
	s = strings.ReplaceAll(s, "\"", "")
	cmd := fmt.Sprintf("TheNet:SystemMessage('%s')", s)
	for _, world := range g.worldSaveData {
		err := utils.ScreenCMD(cmd, world.screenName)
		if err == nil {
			return err
		}
	}

	return fmt.Errorf("执行失败")
}

func (g *Game) sessionInfo() *RoomSessionInfo {
	roomSessionInfo := RoomSessionInfo{
		Season: "error",
		Cycles: -1,
		Phase:  "error",
	}

	var (
		sessionPath string
		sessionErr  error
	)

	for _, world := range g.worldSaveData {
		sessionPath, sessionErr = findLatestMetaFile(world.sessionPath)
		if sessionErr == nil {
			break
		}
	}

	if sessionPath == "" {
		return &roomSessionInfo
	}

	// 读取二进制文件
	data, err := os.ReadFile(sessionPath)
	if err != nil {
		return &roomSessionInfo
	}

	// 创建 Lua 虚拟机
	L := lua.NewState()
	defer L.Close()

	// 将文件内容作为 Lua 代码执行
	content := string(data)
	content = content[:len(content)-1]

	err = L.DoString(content)
	if err != nil {
		return &roomSessionInfo
	}
	// 获取 Lua 脚本的返回值
	lv := L.Get(-1)
	if tbl, ok := lv.(*lua.LTable); ok {
		// 获取 clock 表
		clockTable := tbl.RawGet(lua.LString("clock"))
		if clock, ok := clockTable.(*lua.LTable); ok {
			// 获取 cycles 字段
			cycles := clock.RawGet(lua.LString("cycles"))
			if cyclesValue, ok := cycles.(lua.LNumber); ok {
				roomSessionInfo.Cycles = int(cyclesValue)
			}
			// 获取 phase 字段
			phase := clock.RawGet(lua.LString("phase"))
			if phaseValue, ok := phase.(lua.LString); ok {
				roomSessionInfo.Phase = string(phaseValue)
			}
		}
		// 获取 seasons 表
		seasonsTable := tbl.RawGet(lua.LString("seasons"))
		if seasons, ok := seasonsTable.(*lua.LTable); ok {
			// 获取 season 字段
			season := seasons.RawGet(lua.LString("season"))
			if seasonValue, ok := season.(lua.LString); ok {
				roomSessionInfo.Season = string(seasonValue)
			}
			// 获取 elapseddaysinseason 字段
			elapsedDays := seasons.RawGet(lua.LString("elapseddaysinseason"))
			if elapsedDaysValue, ok := elapsedDays.(lua.LNumber); ok {
				roomSessionInfo.ElapsedDays = int(elapsedDaysValue)
			}
			//获取季节长度
			lengthsTable := seasons.RawGet(lua.LString("lengths"))
			if lengths, ok := lengthsTable.(*lua.LTable); ok {
				summer := lengths.RawGet(lua.LString("summer"))
				if summerValue, ok := summer.(lua.LNumber); ok {
					roomSessionInfo.SeasonLength.Summer = int(summerValue)
				}
				autumn := lengths.RawGet(lua.LString("autumn"))
				if autumnValue, ok := autumn.(lua.LNumber); ok {
					roomSessionInfo.SeasonLength.Autumn = int(autumnValue)
				}
				spring := lengths.RawGet(lua.LString("spring"))
				if springValue, ok := spring.(lua.LNumber); ok {
					roomSessionInfo.SeasonLength.Spring = int(springValue)
				}
				winter := lengths.RawGet(lua.LString("winter"))
				if winterValue, ok := winter.(lua.LNumber); ok {
					roomSessionInfo.SeasonLength.Winter = int(winterValue)
				}

			}
		}
	}

	return &roomSessionInfo
}

type SaveJson struct {
	Room        models.Room        `json:"room"`
	Worlds      []models.World     `json:"worlds"`
	RoomSetting models.RoomSetting `json:"roomSetting"`
}

func (g *Game) backup() error {
	// 生成房间信息
	saveJson := SaveJson{
		Room:        *g.room,
		Worlds:      *g.worlds,
		RoomSetting: *g.setting,
	}
	// 房间信息写入文件
	err := utils.StructToJsonFile(fmt.Sprintf("%s/dmp.json", g.clusterPath), saveJson)
	if err != nil {
		return err
	}

	// 生成压缩文件
	cycle := g.sessionInfo().Cycles
	ts := utils.GetTimestamp()
	fileName := fmt.Sprintf("%s<-@dmp@->%d<-@dmp@->%d", g.room.GameName, cycle, ts)
	fileNameEncode := utils.Base64Encode(fileName) + ".zip"

	zipPath := fmt.Sprintf("%s/backup/%d", utils.DmpFiles, g.room.ID)
	err = utils.EnsureDirExists(zipPath)
	if err != nil {
		return err
	}

	zipFilePath := fmt.Sprintf("%s/%s", zipPath, fileNameEncode)

	err = utils.Zip(g.clusterPath, zipFilePath)
	if err != nil {
		return err
	}

	return nil
}

func (g *Game) restore(filename string) (*SaveJson, error) {
	zipPath := fmt.Sprintf("%s/backup/%d", utils.DmpFiles, g.room.ID)
	filePath := fmt.Sprintf("%s/%s", zipPath, filename)
	err := utils.Unzip(filePath, zipPath)
	if err != nil {
		return nil, err
	}

	saveJson := SaveJson{
		Room:        *g.room,
		Worlds:      *g.worlds,
		RoomSetting: *g.setting,
	}

	dmpJsonPath := fmt.Sprintf("%s/Cluster_%d/dmp.json", zipPath, g.room.ID)
	logger.Logger.Debug(dmpJsonPath)
	err = utils.JsonFileToStruct(dmpJsonPath, &saveJson)
	if err != nil {
		return nil, err
	}

	_ = g.stopAllWorld()

	cmd := fmt.Sprintf("rm -rf %s && cp -r %s/Cluster_%d %s", g.clusterPath, zipPath, g.room.ID, utils.ClusterPath)
	logger.Logger.Debug(cmd)
	err = utils.BashCMD(cmd)
	if err != nil {
		return nil, err
	}

	err = utils.RemoveDir(fmt.Sprintf("%s/Cluster_%d", zipPath, g.room.ID))
	if err != nil {
		return nil, err
	}

	return &saveJson, nil
}

type BackupFile struct {
	GameName  string `json:"gameName"`
	Cycles    string `json:"cycles"`
	TimeStamp int    `json:"timestamp"`
	Size      int64  `json:"size"`
	FileName  string `json:"fileName"`
}

func (g *Game) getBackups() ([]BackupFile, error) {
	zipPath := fmt.Sprintf("%s/backup/%d", utils.DmpFiles, g.room.ID)
	zipFiles, err := utils.GetFiles(zipPath)
	if err != nil {
		return []BackupFile{}, err
	}

	var backupFile []BackupFile

	for _, filename := range zipFiles {
		filenameParts := strings.Split(filename, ".")
		if len(filenameParts) != 2 {
			logger.Logger.Debug(filename)
			continue
		}

		decodeFilename, err := utils.Base64Decode(filenameParts[0])
		if err != nil {
			logger.Logger.Debug(filename)
			logger.Logger.Debug(err.Error())
			continue
		}
		decodeFilenameParts := strings.Split(decodeFilename, "<-@dmp@->")
		if len(decodeFilenameParts) != 3 {
			logger.Logger.Debug(decodeFilename)
			continue
		}

		ts, err := strconv.Atoi(decodeFilenameParts[2])
		if err != nil {
			logger.Logger.Debug(decodeFilename)
			continue
		}

		size, err := utils.GetFileSize(fmt.Sprintf("%s/%s", zipPath, filename))
		if err != nil {
			logger.Logger.Warn("获取备份文件大小失败", "err", err)
		}

		backupFile = append(backupFile, BackupFile{
			GameName:  decodeFilenameParts[0],
			Cycles:    decodeFilenameParts[1],
			TimeStamp: ts,
			Size:      size,
			FileName:  filename,
		})
	}

	if len(backupFile) == 0 {
		return []BackupFile{}, nil
	}

	return backupFile, nil
}

func (g *Game) deleteBackups(filenames []string) int {
	s := 0
	for _, filename := range filenames {
		filePath := fmt.Sprintf("%s/backup/%d/%s", utils.DmpFiles, g.room.ID, filename)
		err := utils.RemoveFile(filePath)
		if err != nil {
			logger.Logger.Error("删除备份文件失败", "err", err)
		}
		s++
	}

	return s
}

func findLatestMetaFile(directory string) (string, error) {
	// 检查指定目录是否存在
	_, err := os.Stat(directory)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("目录不存在：%s", directory)
	}

	// 获取指定目录下的所有子目录
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", fmt.Errorf("读取目录失败：%s", err)
	}

	// 用于存储最新的.meta文件路径和其修改时间
	var latestMetaFile string
	var latestMetaFileTime time.Time

	for _, entry := range entries {
		// 检查是否是目录
		if entry.IsDir() {
			subDirPath := filepath.Join(directory, entry.Name())

			// 获取子目录下的所有文件
			files, err := os.ReadDir(subDirPath)
			if err != nil {
				return "", fmt.Errorf("读取子目录失败：%s", err)
			}

			for _, file := range files {
				// 检查文件是否是.meta文件
				if !file.IsDir() && filepath.Ext(file.Name()) == ".meta" {
					// 获取文件的完整路径
					fullPath := filepath.Join(subDirPath, file.Name())

					// 获取文件的修改时间
					info, err := file.Info()
					if err != nil {
						return "", fmt.Errorf("获取文件信息失败：%s", err)
					}
					modifiedTime := info.ModTime()

					// 如果找到的文件的修改时间比当前最新的.meta文件的修改时间更晚，则更新最新的.meta文件路径和修改时间
					if modifiedTime.After(latestMetaFileTime) {
						latestMetaFile = fullPath
						latestMetaFileTime = modifiedTime
					}
				}
			}
		}
	}

	if latestMetaFile == "" {
		return "", fmt.Errorf("未找到.meta文件")
	}

	return latestMetaFile, nil
}

func (g *Game) runningScreen() ([]string, error) {
	cmd := fmt.Sprintf("ps -ef | grep DMP_Cluster_%d | grep dontstarve_dedicated_server_nullrenderer | grep -v grep | awk '{print $14}'", g.room.ID)
	out, _, _ := utils.BashCMDOutput(cmd)
	screenNamesStr := strings.TrimSpace(out)

	return strings.Split(screenNamesStr, "\n"), nil
}

func (g *Game) deleteRoom() error {
	// 关闭游戏
	_ = g.stopAllWorld()
	// 删除配置文件
	err := utils.RemoveDir(g.clusterPath)
	if err != nil {
		return err
	}
	// 删除mod
	err = utils.RemoveDir(g.ugcPath)
	if err != nil {
		return err
	}
	// 删除备份
	err = utils.RemoveDir(fmt.Sprintf("%s/backup/%d", utils.DmpFiles, g.room.ID))
	if err != nil {
		return err
	}

	return nil
}

func (g *Game) getSnapshot() ([]SnapshotFile, error) {
	sessionID, err := getSessionID(g.worldSaveData[0].savePath)
	if err != nil {
		return []SnapshotFile{}, err
	}

	snapshotPath := fmt.Sprintf("%s/%s", g.worldSaveData[0].sessionPath, sessionID)

	return getSnapshotFiles(snapshotPath)
}

func (g *Game) deleteSnapshot(filename string) error {
	for _, world := range g.worldSaveData {
		sessionID, err := getSessionID(world.savePath)
		if err != nil {
			return err
		}

		sessionFile := fmt.Sprintf("%s/%s/%s", world.sessionPath, sessionID, filename)
		err = utils.RemoveFile(sessionFile)
		if err != nil {
			return err
		}

		sessionFileMeta := fmt.Sprintf("%s/%s/%s.meta", world.sessionPath, sessionID, filename)
		err = utils.RemoveFile(sessionFileMeta)
		if err != nil {
			return err
		}
	}

	return nil
}
