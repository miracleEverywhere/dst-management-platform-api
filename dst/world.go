package dst

import (
	"bufio"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type worldSaveData struct {
	worldPath             string
	serverIniPath         string
	savePath              string
	sessionPath           string
	levelDataOverridePath string
	modOverridesPath      string
	startCmd              string
	screenName            string
	models.World
}

func (g *Game) createWorlds() error {
	g.worldMutex.Lock()
	defer g.worldMutex.Unlock()

	var (
		err        error
		worldsName []string
	)

	// 保存文件
	for _, world := range g.worldSaveData {

		err = utils.EnsureDirExists(world.worldPath)
		if err != nil {
			return err
		}

		err = utils.TruncAndWriteFile(world.serverIniPath, getServerIni(&world.World))
		if err != nil {
			return err
		}

		err = utils.TruncAndWriteFile(world.levelDataOverridePath, world.LevelData)
		if err != nil {
			return err
		}

		modData := world.ModData
		if g.room.ModInOne {
			modData = g.room.ModData
		}
		err = utils.TruncAndWriteFile(world.modOverridesPath, normalizeModOverridesContent(modData))
		if err != nil {
			return err
		}

		worldsName = append(worldsName, world.WorldName)
	}

	// 清理删除的世界
	fileSystemWorlds, err := utils.GetDirs(g.clusterPath, false)
	if err != nil {
		logger.Logger.Warnf("获取世界目录列表失败: %v", err)
		return nil
	}
	for _, fileSystemWorld := range fileSystemWorlds {
		if !utils.Contains(worldsName, fileSystemWorld) {
			// 清理文件
			err = utils.RemoveDir(fmt.Sprintf("%s/%s", g.clusterPath, fileSystemWorld))
			if err != nil {
				logger.Logger.Warnf("清理世界失败，删除文件失败: %v", err)
			}
			// 清理运行中的旧世界
			err = g.cleanupRuntimeName(fmt.Sprintf("DMP_Cluster_%d_%s", g.room.ID, fileSystemWorld))
			if err != nil {
				logger.Logger.Warnf("清理世界失败，清理运行时失败: %v", err)
			}
		}
	}

	return nil
}

func (g *Game) worldUpStatus(id int) bool {
	var (
		stat  bool
		err   error
		world *worldSaveData
	)

	world, err = g.getWorldByID(id)
	if err != nil {
		return false
	}

	stat = g.isWorldRunning(world)

	return stat
}

type PerformanceStatus struct {
	CPU     float64 `json:"cpu"`
	Mem     float64 `json:"mem"`
	MemSize float64 `json:"memSize"`
	Disk    int64   `json:"disk"`
}

func (g *Game) worldPerformanceStatus(id int) PerformanceStatus {
	var performanceStatus PerformanceStatus

	world, err := g.getWorldByID(id)
	if err != nil {
		return performanceStatus
	}

	diskUsed, err := utils.GetDirSize(world.worldPath)
	if err != nil {
		logger.Logger.Warnf("获取世界磁盘使用量失败: %v, 世界id: %d", err, world.ID)
		diskUsed = 0
	}

	performanceStatus.Disk = diskUsed

	if !g.worldUpStatus(id) {
		return performanceStatus
	}

	p, err := g.worldProcess(world)
	if err != nil {
		logger.Logger.Warnf("获取世界进程失败, world: %v, err: %v", world.ID, err)
		return performanceStatus
	}

	cpu, err := p.Percent(time.Millisecond * 100)
	if err != nil {
		logger.Logger.Warnf("获取世界CPU失败, world: %v, err: %v", world.ID, err)
		return performanceStatus
	}

	performanceStatus.CPU = cpu

	mem, err := p.MemoryPercent()
	if err != nil {
		logger.Logger.Warnf("获取世界内存使用率失败, world: %v, err: %v", world.ID, err)
		return performanceStatus
	}

	performanceStatus.Mem = float64(mem)

	memSize, err := p.MemoryInfo()
	if err != nil {
		logger.Logger.Warnf("获取世界内存使用量失败, world: %v, err: %v", world.ID, err)
		return performanceStatus
	}

	performanceStatus.MemSize = float64(memSize.RSS / 1024 / 1024)

	logger.Logger.Debug(utils.StructToFlatString(performanceStatus))

	return performanceStatus
}

func (g *Game) startWorld(id int) error {
	g.cleanupRuntime()

	// 启动游戏后，删除mod临时下载目录
	g.acfMutex.Lock()
	defer g.acfMutex.Unlock()
	defer func() {
		err := utils.RemoveDir(fmt.Sprintf("%s/mods/ugc/%s", utils.DmpFiles, g.clusterName))
		if err != nil {
			logger.Logger.Warnf("删除临时模组失败, err: %v", err)
		}
	}()

	g.prepareRuntimeFiles()

	var (
		err   error
		world *worldSaveData
	)

	// 如果正在运行，则跳过
	if g.worldUpStatus(id) {
		logger.Logger.Infof("当前世界正在运行中，跳过，世界ID：%d", id)
		return nil
	}

	world, err = g.getWorldByID(id)
	if err != nil {
		return err
	}

	err = g.dsModsSetup()
	if err != nil {
		return err
	}

	logger.Logger.Debug(world.startCmd)
	err = g.startWorldProcess(world)

	return err
}

func (g *Game) startAllWorld() error {
	g.cleanupRuntime()

	var err error

	g.prepareRuntimeFiles()

	err = g.dsModsSetup()
	if err != nil {
		return err
	}

	for _, world := range g.worldSaveData {
		// 如果正在运行，则跳过
		if g.worldUpStatus(world.ID) {
			logger.Logger.Infof("当前世界正在运行中，跳过，世界ID：%d", world.ID)
			continue
		}

		logger.Logger.Debug(world.startCmd)
		err = g.startWorldProcess(&world)
		if err != nil {
			return err
		}
	}

	webhook.Snd.Send(webhook.EventGameStart, g.room.ID, map[string]interface{}{
		"gameID":   g.room.ID,
		"gameName": g.room.GameName,
	})

	return nil
}

func (g *Game) stopWorld(id int) error {
	world, err := g.getWorldByID(id)
	if err != nil {
		return err
	}

	err = g.stopWorldProcess(world)
	if err != nil {
		logger.Logger.Infof("结束进程失败，可能是未运行: %v", err)
	}

	return nil
}

func (g *Game) stopAllWorld() error {
	for _, world := range g.worldSaveData {
		err := g.stopWorld(world.ID)
		if err != nil {
			return err
		}
	}

	webhook.Snd.Send(webhook.EventGameStop, g.room.ID, map[string]interface{}{
		"gameID":   g.room.ID,
		"gameName": g.room.GameName,
	})

	return nil
}

func (g *Game) deleteWorld(id int) error {
	_ = g.stopWorld(id)
	world, err := g.getWorldByID(id)
	if err != nil {
		return err
	}
	return utils.RemoveDir(world.savePath)
}

func (g *Game) consoleCmd(cmd string, id int) error {
	world, err := g.getWorldByID(id)
	if err != nil {
		return err
	}
	s := strings.ReplaceAll(cmd, "\"", "'")

	return g.sendConsoleCommand(world, s)
}

func (g *Game) getWorldByID(id int) (*worldSaveData, error) {
	for i := range g.worldSaveData {
		if g.worldSaveData[i].ID == id {
			return &g.worldSaveData[i], nil
		}
	}

	return nil, fmt.Errorf("世界不存在: %d", id)
}

func getServerIni(world *models.World) string {
	contents := `[NETWORK]
server_port = ` + strconv.Itoa(world.ServerPort) + `

[SHARD]
id = ` + strconv.Itoa(world.GameID) + `
is_master = ` + strconv.FormatBool(world.IsMaster) + `
name = ` + world.WorldName + `

[STEAM]
master_server_port = ` + strconv.Itoa(world.MasterServerPort) + `
authentication_port = ` + strconv.Itoa(world.AuthenticationPort) + `

[ACCOUNT]
encode_user_path = ` + strconv.FormatBool(world.EncodeUserPath)
	return contents
}

func (g *Game) getOnlinePlayerList(id int) ([]string, error) {
	world, err := g.getWorldByID(id)
	if err != nil {
		return []string{}, err
	}

	listCmd := `for i, v in ipairs(TheNet:GetClientTable()) do print(string.format("playerlist %s [%d] %s <-@dmp@-> %s <-@dmp@-> %s", 99999999, i-1, v.userid, v.name, v.prefab )) end`
	err = g.sendConsoleCommand(world, listCmd)
	if err != nil {
		return []string{}, err
	}

	// 等待命令执行完毕
	time.Sleep(time.Second * 2)

	// 获取日志文件中的list
	logPath := fmt.Sprintf("%s/server_log.txt", world.worldPath)

	// 使用反向读取，只读取最后几KB
	return readPlayerListFromEnd(logPath)
}

var (
	playerListPattern = regexp.MustCompile(`playerlist 99999999 \[[0-9]+\] (KU_.+) <-@dmp@-> (.*) <-@dmp@-> (.+)?`)
	hostPattern       = regexp.MustCompile(`\[Host]`)
)

func readPlayerListFromEnd(logPath string) ([]string, error) {
	const bufferSize = 1024 * 4 // 4KB buffer

	// 打开文件
	file, err := os.Open(logPath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logger.Logger.Errorf("文件关闭失败, err: %v", err)
		}
	}(file)

	// 获取文件大小
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := fileInfo.Size()

	// 计算从哪里开始读取
	startPos := fileSize - bufferSize
	if startPos < 0 {
		startPos = 0
	}

	// 移动到起始位置
	_, err = file.Seek(startPos, 0)
	if err != nil {
		return nil, err
	}

	// 读取缓冲区内容
	buffer := make([]byte, bufferSize)
	n, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}

	content := string(buffer[:n])

	// 分割成行
	lines := strings.Split(content, "\n")

	// 从后往前查找
	var linesAfterKeyword []string
	keyword := "playerlist 99999999 [0]"
	var foundKeyword bool

	// 从末尾开始遍历
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]
		linesAfterKeyword = append(linesAfterKeyword, line)

		if strings.Contains(line, keyword) {
			foundKeyword = true
			break
		}
	}

	if !foundKeyword {
		return nil, fmt.Errorf("keyword not found in the file")
	}

	var players []string

	// 查找匹配的行并提取所需字段
	for _, line := range linesAfterKeyword {
		if matches := playerListPattern.FindStringSubmatch(line); matches != nil {
			// 检查是否包含 [Host]
			if !hostPattern.MatchString(line) {
				uid := strings.ReplaceAll(matches[1], "\t", "")
				nickName := strings.ReplaceAll(matches[2], "\t", "")
				prefab := strings.ReplaceAll(matches[3], "\t", "")
				player := uid + "<-@dmp@->" + nickName + "<-@dmp@->" + prefab
				players = append(players, player)
			}
		}
	}

	players = uniqueSliceKeepOrderString(players)

	return players, nil
}

func (g *Game) getLastAliveTime(id int) (string, error) {
	world, err := g.getWorldByID(id)
	if err != nil {
		return "", err
	}

	_ = g.sendConsoleCommand(world, "print('DMP Keepalive')")
	time.Sleep(1 * time.Second)

	return getWorldLastTime(fmt.Sprintf("%s/server_log.txt", world.worldPath))
}

func getWorldLastTime(logfile string) (string, error) {
	// 获取日志文件中的list
	file, err := os.Open(logfile)
	if err != nil {
		logger.Logger.Errorf("打开文件失败, err: %v, file: %v", err, logfile)
		return "", err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logger.Logger.Errorf("关闭文件失败, err: %v, file: %v", err, logfile)
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
		logger.Logger.Errorf("文件scan失败, err: %v", err)
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
