package dst

import (
	"bufio"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/process"
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

		if g.room.ModInOne {
			err = utils.TruncAndWriteFile(world.modOverridesPath, g.room.ModData)
			if err != nil {
				return err
			}
		} else {
			err = utils.TruncAndWriteFile(world.modOverridesPath, world.ModData)
			if err != nil {
				return err
			}
		}

		worldsName = append(worldsName, world.WorldName)
	}

	// 清理删除的世界
	fileSystemWorlds, err := utils.GetDirs(g.clusterPath, false)
	for _, fileSystemWorld := range fileSystemWorlds {
		if !utils.Contains(worldsName, fileSystemWorld) {
			// 清理文件
			err = utils.RemoveDir(fmt.Sprintf("%s/%s", g.clusterPath, fileSystemWorld))
			if err != nil {
				logger.Logger.Warn("清理世界失败，删除文件失败", "err", err)
			}
			// 清理screen
			cmd := fmt.Sprintf("screen -X -S DMP_Cluster_%d_%s quit", g.room.ID, fileSystemWorld)
			err = utils.BashCMD(cmd)
			if err != nil {
				logger.Logger.Warn("清理世界失败，清理SCREEN失败", "err", err)
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

	cmd := fmt.Sprintf("ps -ef | grep %s | grep -v grep", world.screenName)
	err = utils.BashCMD(cmd)
	if err != nil {
		stat = false
	} else {
		stat = true
	}

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
		logger.Logger.Warn("获取世界磁盘使用量失败", "world", world.ID, "err", err)
		diskUsed = 0
	}

	performanceStatus.Disk = diskUsed

	if !g.worldUpStatus(id) {
		return performanceStatus
	}

	cmd := fmt.Sprintf("ps -ef | grep dontstarve_dedicated_server_nullrenderer | grep Cluster_%d | grep %s | grep -v luajit | grep -vi screen | awk '{print $2}'", g.room.ID, world.WorldName)
	logger.Logger.Debug(cmd)
	out, _, _ := utils.BashCMDOutput(cmd)
	logger.Logger.Debug(out)

	if len(out) < 2 {
		logger.Logger.Warn("获取世界PID失败", "world", world.ID)
		return performanceStatus
	}

	pid, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		logger.Logger.Warn("获取世界PID失败", "world", world.ID, "err", err)
		return performanceStatus
	}

	p, err := process.NewProcess(int32(pid))
	if err != nil {
		logger.Logger.Warn("获取世界进程失败", "world", world.ID, "err", err)
		return performanceStatus
	}

	cpu, err := p.Percent(time.Millisecond * 100)
	if err != nil {
		logger.Logger.Warn("获取世界CPU失败", "world", world.ID, "err", err)
		return performanceStatus
	}

	performanceStatus.CPU = cpu

	mem, err := p.MemoryPercent()
	if err != nil {
		logger.Logger.Warn("获取世界内存使用率失败", "world", world.ID, "err", err)
		return performanceStatus
	}

	performanceStatus.Mem = float64(mem)

	memSize, err := p.MemoryInfo()
	if err != nil {
		logger.Logger.Warn("获取世界内存使用量失败", "world", world.ID, "err", err)
		return performanceStatus
	}

	performanceStatus.MemSize = float64(memSize.RSS / 1024 / 1024)

	logger.Logger.Debug(utils.StructToFlatString(performanceStatus))

	return performanceStatus
}

func (g *Game) startWorld(id int) error {
	_ = utils.BashCMD("screen -wipe")

	// 启动游戏后，删除mod临时下载目录
	g.acfMutex.Lock()
	defer g.acfMutex.Unlock()
	defer func() {
		err := utils.RemoveDir(fmt.Sprintf("%s/mods/ugc/%s", utils.DmpFiles, g.clusterName))
		if err != nil {
			logger.Logger.Warn("删除临时模组失败", "err", err)
		}
	}()

	// 给klei擦钩子，检查so文件
	if !utils.CompareFileSHA256("dst/bin/lib32/steamclient.so", "steamcmd/linux32/steamclient.so") {
		logger.Logger.Debug("发现so文件异常，开始替换")
		replaceDSTSOFile()
	}

	var (
		err   error
		world *worldSaveData
	)

	// 如果正在运行，则跳过
	if g.worldUpStatus(id) {
		logger.Logger.Info("当前世界正在运行中，跳过", "世界ID", id)
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
	err = utils.BashCMD(world.startCmd)

	return err
}

func (g *Game) startAllWorld() error {
	_ = utils.BashCMD("screen -wipe")

	var err error

	// 给klei擦钩子，检查so文件
	if !utils.CompareFileSHA256("dst/bin/lib32/steamclient.so", "steamcmd/linux32/steamclient.so") {
		logger.Logger.Debug("发现so文件异常，开始替换")
		replaceDSTSOFile()
	}

	err = g.dsModsSetup()
	if err != nil {
		return err
	}

	for _, world := range g.worldSaveData {
		// 如果正在运行，则跳过
		if g.worldUpStatus(world.ID) {
			logger.Logger.Info("当前世界正在运行中，跳过", "世界ID", world.ID)
			continue
		}

		logger.Logger.Debug(world.startCmd)
		err = utils.BashCMD(world.startCmd)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Game) stopWorld(id int) error {
	world, err := g.getWorldByID(id)
	if err != nil {
		return err
	}

	err = utils.ScreenCMD("c_shutdown()", world.screenName)
	if err != nil {
		logger.Logger.Info("执行ScreenCMD失败，可能是未运行", "msg", err, "cmd", "c_shutdown()")
	}

	time.Sleep(1 * time.Second)

	killCMD := fmt.Sprintf("screen -S %s -X quit", world.screenName)
	err = utils.BashCMD(killCMD)
	if err != nil {
		logger.Logger.Info("结束进程失败，可能是未运行", "err", err)
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

	return utils.ScreenCMD(s, world.screenName)
}

func (g *Game) getWorldByID(id int) (*worldSaveData, error) {
	for _, world := range g.worldSaveData {
		if world.ID == id {
			return &world, nil
		}
	}

	return &worldSaveData{}, nil
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

	listScreenCmd := fmt.Sprintf("screen -S \"%s\" -p 0 -X stuff \"for i, v in ipairs(TheNet:GetClientTable()) do  print(string.format(\\\"playerlist %%s [%%d] %%s <-@dmp@-> %%s <-@dmp@-> %%s\\\", 99999999, i-1, v.userid, v.name, v.prefab )) end$(printf \\\\r)\"\n", world.screenName)
	err = utils.BashCMD(listScreenCmd)
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
			logger.Logger.Error("文件关闭失败", "err", err)
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

	_ = utils.ScreenCMD("print('DMP Keepalive')", world.screenName)
	time.Sleep(1 * time.Second)

	return getWorldLastTime(fmt.Sprintf("%s/server_log.txt", world.worldPath))
}

func getWorldLastTime(logfile string) (string, error) {
	// 获取日志文件中的list
	file, err := os.Open(logfile)
	if err != nil {
		logger.Logger.Error("打开文件失败", "err", err, "file", logfile)
		return "", err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logger.Logger.Error("关闭文件失败", "err", err, "file", logfile)
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
		logger.Logger.Error("文件scan失败", "err", err)
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
