package dst

import (
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	lua "github.com/yuin/gopher-lua"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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

	return nil
}

func (g *Game) getClusterIni() string {
	var (
		gameMode string
		lang     string
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

	contents := `[GAMEPLAY]
game_mode = ` + gameMode + `
max_players = ` + strconv.Itoa(g.room.MaxPlayer) + `
pvp = ` + strconv.FormatBool(g.room.Pvp) + `
pause_when_empty = ` + strconv.FormatBool(g.room.PauseEmpty) + `
vote_enabled = ` + strconv.FormatBool(g.room.Vote) + `
vote_kick_enabled = ` + strconv.FormatBool(g.room.Vote) + `

[NETWORK]
cluster_description = ` + g.room.Description + `
whitelist_slots = ` + strconv.Itoa(len(g.adminlist)) + `
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
cluster_key = ` + g.room.ClusterKey + `
`

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
