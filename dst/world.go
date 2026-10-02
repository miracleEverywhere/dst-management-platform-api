package dst

import (
	"bufio"
	"dst-management-platform-api/cache"
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

	"github.com/shirou/gopsutil/v4/process"
)

// 启停时序常量
const (
	// stopGraceTimeout 等待分片优雅退出的最长时间。
	// 游戏收到 c_shutdown() 后需要完成断开玩家、向Klei大厅注销房间(/lobby/delete)、
	// 序列化存档等收尾工作，实测约 26~60 秒；提前强杀会导致房间行残留在大厅数据库中，
	// 下次启动报 E_ROWID_EXIST 且整个运行期无法在服务器列表中被搜到
	stopGraceTimeout = 150 * time.Second
	// stopPollInterval 优雅退出状态轮询间隔
	stopPollInterval = 2 * time.Second
	// startSpawnTimeout 等待分片进程出现(screen会话已建立)的最长时间
	startSpawnTimeout = 30 * time.Second
	// startReadyTimeout 等待分片完成加载的最长时间。
	// 以各分片日志出现"本局加载完成"标志为准（主世界: Starting master server；
	// 从属世界: Sending secondary shard information to master），
	// 带大量模组时加载需 60~120 秒，进程活着不代表已加载完成
	startReadyTimeout = 240 * time.Second
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
			// 清理screen
			cmd := fmt.Sprintf("screen -X -S DMP_Cluster_%d_%s quit", g.room.ID, fileSystemWorld)
			err = utils.BashCMD(cmd)
			if err != nil {
				logger.Logger.Warnf("清理世界失败，清理SCREEN失败: %v", err)
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
		logger.Logger.Warnf("获取世界磁盘使用量失败: %v, 世界id: %d", err, world.ID)
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
		logger.Logger.Warnf("获取世界PID失败, 世界id: %d", world.ID)
		return performanceStatus
	}

	pid, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		logger.Logger.Warnf("获取世界PID失败, id: %d, err: %v", world.ID, err)
		return performanceStatus
	}

	p, err := process.NewProcess(int32(pid))
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
	_ = utils.BashCMD("screen -wipe")

	// 启动游戏后，删除mod临时下载目录
	g.acfMutex.Lock()
	defer g.acfMutex.Unlock()
	defer func() {
		err := utils.RemoveDir(fmt.Sprintf("%s/mods/ugc/%s", utils.DmpFiles, g.clusterName))
		if err != nil {
			logger.Logger.Warnf("删除临时模组失败, err: %v", err)
		}
	}()

	var (
		err   error
		world *worldSaveData
	)

	// 给klei擦钩子，检查so文件
	if cache.OsType != "darwin" {
		if !utils.CompareFileSHA256("dst/bin/lib32/steamclient.so", "steamcmd/linux32/steamclient.so") {
			logger.Logger.Debug("发现so文件异常，开始替换")
			replaceDSTSOFile()
		}
	}

	// 如果正在运行，则跳过
	if g.worldUpStatus(id) {
		logger.Logger.Infof("当前世界正在运行中，跳过，世界ID：%d", id)
		return nil
	}

	err = g.dsModsSetup()
	if err != nil {
		return err
	}

	world, err = g.getWorldByID(id)
	if err != nil {
		return err
	}

	logger.Logger.Debug(world.startCmd)
	if !utils.IsSafeString(world.WorldName) {
		logger.Logger.Warnf("世界名 %s 可能存在注入风险", world.WorldName)
		return fmt.Errorf("世界名 %s 可能存在注入风险", world.WorldName)
	}
	// 记录启动前日志修改时间，用于区分本次运行的新日志
	baseline := time.Time{}
	if fi, err := os.Stat(fmt.Sprintf("%s/server_log.txt", world.worldPath)); err == nil {
		baseline = fi.ModTime()
	}
	if err := utils.BashCMD(world.startCmd); err != nil {
		return fmt.Errorf("世界 %s 启动失败: %w", world.WorldName, err)
	}

	// 确认世界完成加载后才视为启动成功
	return g.waitWorldsUp([]launchedWorld{{id: id, baseline: baseline}})
}

func (g *Game) startAllWorld() error {
	_ = utils.BashCMD("screen -wipe")

	var err error

	if cache.OsType != "darwin" {
		// 给klei擦钩子，检查so文件
		if !utils.CompareFileSHA256("dst/bin/lib32/steamclient.so", "steamcmd/linux32/steamclient.so") {
			logger.Logger.Debug("发现so文件异常，开始替换")
			replaceDSTSOFile()
		}
	}

	err = g.dsModsSetup()
	if err != nil {
		return err
	}

	launched := make([]launchedWorld, 0, len(g.worldSaveData))
	for _, world := range g.worldSaveData {
		// 如果正在运行，则跳过
		if g.worldUpStatus(world.ID) {
			logger.Logger.Infof("当前世界正在运行中，跳过，世界ID：%d", world.ID)
			continue
		}

		logger.Logger.Debug(world.startCmd)
		if !utils.IsSafeString(world.WorldName) {
			logger.Logger.Warnf("世界名 %s 可能存在注入风险", world.WorldName)
			return fmt.Errorf("世界名 %s 可能存在注入风险", world.WorldName)
		}
		// 记录启动前日志修改时间：游戏启动会截断重写 server_log.txt，
		// 以此区分本次运行的新日志与上一次运行残留的内容
		baseline := time.Time{}
		if fi, err := os.Stat(fmt.Sprintf("%s/server_log.txt", world.worldPath)); err == nil {
			baseline = fi.ModTime()
		}
		if err := utils.BashCMD(world.startCmd); err != nil {
			return fmt.Errorf("世界 %s 启动失败: %w", world.WorldName, err)
		}
		launched = append(launched, launchedWorld{id: world.ID, baseline: baseline})
	}

	// 确认所有新启动的世界完成加载后才视为启动成功
	if err := g.waitWorldsUp(launched); err != nil {
		return err
	}

	webhook.Snd.Send(webhook.EventGameStart, g.room.ID, map[string]interface{}{
		"gameID":   g.room.ID,
		"gameName": g.room.GameName,
	})

	return nil
}

// launchedWorld 记录本次启动的世界及其日志基线时间
type launchedWorld struct {
	id       int
	baseline time.Time
}

// waitWorldsUp 等待指定的世界完成启动：
// 1. 进程真实出现；2. 各分片日志出现本局"加载完成"标志。
// 全部就绪才返回 nil；否则返回带具体世界名的错误
func (g *Game) waitWorldsUp(launched []launchedWorld) error {
	if len(launched) == 0 {
		return nil
	}

	worldName := func(id int) string {
		world, err := g.getWorldByID(id)
		if err != nil {
			return fmt.Sprintf("世界#%d", id)
		}
		return world.WorldName
	}

	// 1. 等待进程出现
	deadline := time.Now().Add(startSpawnTimeout)
	for {
		allUp := true
		for _, lw := range launched {
			if !g.worldUpStatus(lw.id) {
				allUp = false
				break
			}
		}
		if allUp {
			break
		}
		if time.Now().After(deadline) {
			var down []string
			for _, lw := range launched {
				if !g.worldUpStatus(lw.id) {
					down = append(down, worldName(lw.id))
				}
			}
			return fmt.Errorf("世界 %s 未能启动，请查看游戏日志", strings.Join(down, "、"))
		}
		time.Sleep(stopPollInterval)
	}

	// 2. 等待各分片完成加载（日志出现本局就绪标志）。
	//    进程活着不代表加载完成，带大量模组时加载需要 60~120 秒；
	//    等待期间任何进程退出都判定为启动失败
	deadline = time.Now().Add(startReadyTimeout)
	for {
		var down, notReady []string
		for _, lw := range launched {
			if !g.worldUpStatus(lw.id) {
				world, err := g.getWorldByID(lw.id)
				if err != nil {
					down = append(down, fmt.Sprintf("世界#%d", lw.id))
				} else {
					down = append(down, world.WorldName)
				}
				continue
			}
			world, err := g.getWorldByID(lw.id)
			if err != nil {
				continue
			}
			if !g.worldLogReady(world, lw.baseline) {
				notReady = append(notReady, world.WorldName)
			}
		}
		if len(down) > 0 {
			return fmt.Errorf("世界 %s 启动后异常退出，请查看游戏日志", strings.Join(down, "、"))
		}
		if len(notReady) == 0 {
			break
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("世界 %s 迟迟未完成加载(超过%d秒)，请查看游戏日志", strings.Join(notReady, "、"), int(startReadyTimeout.Seconds()))
		}
		time.Sleep(stopPollInterval)
	}

	logger.Logger.Info("所有世界已启动并完成加载")
	return nil
}

// worldLogReady 检查世界日志是否出现本局"加载完成"标志行。
// 主世界: [Shard] Starting master server
// 从属世界: [Shard] Sending secondary shard information to master
// baseline 为启动前日志文件的修改时间：游戏启动会截断重写 server_log.txt，
// 文件修改时间晚于 baseline 才说明读到的是本次运行的日志而非上次残留
func (g *Game) worldLogReady(world *worldSaveData, baseline time.Time) bool {
	logPath := fmt.Sprintf("%s/server_log.txt", world.worldPath)
	fi, err := os.Stat(logPath)
	if err != nil || !fi.ModTime().After(baseline) {
		return false
	}

	f, err := os.Open(logPath)
	if err != nil {
		return false
	}
	defer f.Close()

	const tailSize = 256 * 1024
	if fi.Size() > tailSize {
		if _, err := f.Seek(-tailSize, io.SeekEnd); err != nil {
			return false
		}
	}
	data, err := io.ReadAll(f)
	if err != nil {
		return false
	}

	if world.IsMaster {
		return strings.Contains(string(data), "Starting master server")
	}
	return strings.Contains(string(data), "Sending secondary shard information to master")
}

func (g *Game) stopWorld(id int) error {
	world, err := g.getWorldByID(id)
	if err != nil {
		return err
	}

	err = utils.ScreenCMD("c_shutdown()", world.screenName)
	if err != nil {
		logger.Logger.Infof("执行ScreenCMD失败，可能是未运行，忽略: %v, cmd: c_shutdown()", err)
	}

	// 等待游戏优雅退出（完成存档并向Klei大厅注销房间），超时才强制结束
	deadline := time.Now().Add(stopGraceTimeout)
	for g.worldUpStatus(id) {
		if time.Now().After(deadline) {
			logger.Logger.Warnf("世界 %s 优雅退出超时(%s)，强制结束", world.WorldName, stopGraceTimeout)
			killCMD := fmt.Sprintf("screen -S %s -X quit", world.screenName)
			if err := utils.BashCMD(killCMD); err != nil {
				logger.Logger.Infof("结束进程失败，可能是未运行，忽略: %v", err)
			}
			time.Sleep(2 * time.Second)
			// 强制退出后复查，仍存活则如实报告失败
			if g.worldUpStatus(id) {
				return fmt.Errorf("世界 %s 未能停止，请手动检查 screen 会话", world.WorldName)
			}
			break
		}
		time.Sleep(stopPollInterval)
	}

	return nil
}

func (g *Game) stopAllWorld() error {
	// 1. 同时向所有分片发送优雅关机指令（不等任何一个分片退出）
	for _, world := range g.worldSaveData {
		if err := utils.ScreenCMD("c_shutdown()", world.screenName); err != nil {
			logger.Logger.Infof("执行ScreenCMD失败，可能是未运行，忽略: %v, 世界: %s, cmd: c_shutdown()", err, world.WorldName)
		}
	}

	// 2. 轮询等待所有分片自行退出。
	//    游戏需要时间完成存档并向Klei大厅注销房间，提前强杀会导致房间行
	//    残留在大厅数据库中，下次启动报 E_ROWID_EXIST 且整个运行期无法被搜到
	deadline := time.Now().Add(stopGraceTimeout)
	for {
		up := g.worldsUp()
		if len(up) == 0 {
			logger.Logger.Info("所有世界已优雅退出")
			break
		}
		if time.Now().After(deadline) {
			// 3. 超时兜底：对仍未退出的世界强制结束
			logger.Logger.Warnf("部分世界优雅退出超时(%s)，强制结束: %s", stopGraceTimeout, strings.Join(up, "、"))
			for _, world := range g.worldSaveData {
				if g.worldUpStatus(world.ID) {
					killCMD := fmt.Sprintf("screen -S %s -X quit", world.screenName)
					if err := utils.BashCMD(killCMD); err != nil {
						logger.Logger.Infof("结束进程失败，可能是未运行，忽略: %v", err)
					}
				}
			}
			// 给强制退出生效的时间，然后做最终确认
			time.Sleep(3 * time.Second)
			if up := g.worldsUp(); len(up) > 0 {
				return fmt.Errorf("世界 %s 未能停止，请手动检查 screen 会话", strings.Join(up, "、"))
			}
			break
		}
		time.Sleep(stopPollInterval)
	}

	webhook.Snd.Send(webhook.EventGameStop, g.room.ID, map[string]interface{}{
		"gameID":   g.room.ID,
		"gameName": g.room.GameName,
	})

	return nil
}

// worldsUp 返回当前仍在运行的世界名列表
func (g *Game) worldsUp() []string {
	var up []string
	for _, world := range g.worldSaveData {
		if g.worldUpStatus(world.ID) {
			up = append(up, world.WorldName)
		}
	}
	return up
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

	_ = utils.ScreenCMD("print('DMP Keepalive')", world.screenName)
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
