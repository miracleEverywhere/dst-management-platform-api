package utils

import (
	"fmt"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	// onlinePlayersConsoleCmd 注入游戏控制台的 Lua：只打印玩家昵称。
	// 首尾各打印一个标记，便于从日志尾部定位最近一次完整输出；
	// userid 为 nil 的是服务器自身，name 含 [Host] 的是房主，都不算在线玩家。
	// 整条命令不写分号，避免 screen 命令行解析把分号当成自己的分隔符。
	onlinePlayersConsoleCmd = `print("dmp_lite_playerlist_begin") for _, v in ipairs(TheNet:GetClientTable()) do if v.userid ~= nil and v.name ~= nil and string.find(tostring(v.name), "%[Host%]") == nil then print("dmp_lite_playerlist " .. tostring(v.name)) end end print("dmp_lite_playerlist_end")`

	// onlinePlayersTag 玩家列表输出中所有标记行的公共前缀
	onlinePlayersTag = "dmp_lite_playerlist"
	// onlinePlayersBeginMarker 一次玩家列表输出的起始标记
	onlinePlayersBeginMarker = onlinePlayersTag + "_begin"
	// onlinePlayersEndMarker 一次玩家列表输出的结束标记
	onlinePlayersEndMarker = onlinePlayersTag + "_end"
	// onlinePlayersLinePrefix 每条玩家昵称行的前缀
	onlinePlayersLinePrefix = onlinePlayersTag + " "
	// onlinePlayersWait 注入命令后等待游戏把结果写入日志的时间
	onlinePlayersWait = 2 * time.Second
	// onlinePlayersMinTailLines 尾读日志的最小行数，要能覆盖一次完整的玩家列表输出
	onlinePlayersMinTailLines = 128
	// onlinePlayersMaxTailLines 尾读日志的最大行数，避免日志异常时读取过多内容
	onlinePlayersMaxTailLines = 512
)

var (
	// onlinePlayersMu 保护 onlinePlayers，定时任务写、HTTP handler 读。
	onlinePlayersMu sync.RWMutex
	// onlinePlayers 后台任务最近一次成功获取到的在线玩家昵称。
	onlinePlayers = []string{}
)

// GetOnlinePlayers 返回后台任务缓存的地面/洞穴在线玩家昵称。
// 未成功获取过时返回空切片；返回副本，定时任务替换缓存不会影响调用方。
func GetOnlinePlayers() []string {
	onlinePlayersMu.RLock()
	defer onlinePlayersMu.RUnlock()

	players := make([]string, len(onlinePlayers))
	copy(players, onlinePlayers)

	return players
}

func setOnlinePlayers(players []string) {
	if players == nil {
		players = []string{}
	}

	onlinePlayersMu.Lock()
	defer onlinePlayersMu.Unlock()

	onlinePlayers = players
}

// onlinePlayersRefresh 按“地面优先、洞穴兜底”的顺序刷新在线玩家昵称。
// 两个世界都未运行时视为无玩家；世界在运行但获取失败时保留上一次缓存。
func onlinePlayersRefresh() {
	DBMu.RLock()
	groundName := DB.Ground.Name
	caveName := DB.Cave.Name
	maxPlayers := DB.Room.MaxPlayer
	DBMu.RUnlock()

	groundAlive := worldScreenAlive(groundName)
	caveAlive := worldScreenAlive(caveName)

	if !groundAlive && !caveAlive {
		setOnlinePlayers(nil)
		return
	}

	if groundAlive {
		players, err := fetchOnlinePlayers(groundName, maxPlayers)
		if err == nil {
			setOnlinePlayers(players)
			return
		}
		Logger.Error("获取地面世界在线玩家失败: " + err.Error())
		if caveAlive {
			Logger.Info("改为获取洞穴世界在线玩家")
		}
	}

	if caveAlive {
		players, err := fetchOnlinePlayers(caveName, maxPlayers)
		if err == nil {
			setOnlinePlayers(players)
			return
		}
		Logger.Error("获取洞穴世界在线玩家失败: " + err.Error())
	}

	if groundAlive || caveAlive {
		Logger.Error("在线玩家获取失败，保留上一次缓存")
	}
}

// fetchOnlinePlayers 向指定世界注入命令，等待输出后尾读该世界的日志并解析昵称。
func fetchOnlinePlayers(worldName string, maxPlayers int) ([]string, error) {
	// 先校验世界名，非法名称不注入命令，也不会拼出集群目录外的日志路径
	logPath := WorldLogPath(worldName)
	if logPath == "" {
		return nil, fmt.Errorf("世界名不合法: %q", worldName)
	}

	screenName := worldScreenPrefix + worldName
	if err := injectConsoleCommand(screenName, onlinePlayersConsoleCmd); err != nil {
		return nil, err
	}

	time.Sleep(onlinePlayersWait)

	lines, err := ReadLastLines(logPath, onlinePlayersTailLines(maxPlayers))
	if err != nil {
		return nil, err
	}

	return parseOnlinePlayers(lines)
}

// onlinePlayersTailLines 按房间最大人数推算尾读行数，保证一次完整输出能被覆盖。
func onlinePlayersTailLines(maxPlayers int) int {
	lines := maxPlayers*2 + 64
	if lines < onlinePlayersMinTailLines {
		lines = onlinePlayersMinTailLines
	}
	if lines > onlinePlayersMaxTailLines {
		lines = onlinePlayersMaxTailLines
	}

	return lines
}

// injectConsoleCommand 通过 screen 向游戏控制台发送一行 Lua 命令并回车。
// 使用 exec.Command 的 argv 形式，命令内容不经过 shell，避免转义问题。
func injectConsoleCommand(screenName, cmd string) error {
	out, err := exec.Command("screen", "-S", screenName, "-p", "0", "-X", "stuff", cmd+"\r").CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w, 输出=%s", err, strings.TrimSpace(string(out)))
	}

	return nil
}

// worldScreenAlive 判断世界对应的 screen 会话是否在运行，世界名为空时视为未运行。
func worldScreenAlive(worldName string) bool {
	if worldName == "" {
		return false
	}

	return ScreenAlive(worldScreenPrefix + worldName)
}

// parseOnlinePlayers 从日志尾部解析最近一次完整输出的玩家昵称。
// 找不到成对的起止标记时返回错误，以便上层保留上一次缓存；成功但无玩家时返回空切片。
func parseOnlinePlayers(lines []string) ([]string, error) {
	endIdx := -1
	for i := len(lines) - 1; i >= 0; i-- {
		if isOnlinePlayersMarker(lines[i], onlinePlayersEndMarker) {
			endIdx = i
			break
		}
	}
	if endIdx < 0 {
		return nil, fmt.Errorf("日志中未找到玩家列表结束标记")
	}

	beginIdx := -1
	for i := endIdx - 1; i >= 0; i-- {
		if isOnlinePlayersMarker(lines[i], onlinePlayersBeginMarker) {
			beginIdx = i
			break
		}
	}
	if beginIdx < 0 {
		return nil, fmt.Errorf("日志中未找到玩家列表开始标记")
	}

	players := make([]string, 0, endIdx-beginIdx-1)
	for _, line := range lines[beginIdx+1 : endIdx] {
		if name, ok := onlinePlayerName(line); ok {
			players = append(players, name)
		}
	}

	return players, nil
}

// onlinePlayersLineSuffix 从日志行中截取标记开始的部分。
// DST 会在 print 输出前加时间戳，所以按行内第一个标记定位。
func onlinePlayersLineSuffix(line string) (string, bool) {
	idx := strings.Index(line, onlinePlayersTag)
	if idx < 0 {
		return "", false
	}

	return line[idx:], true
}

// isOnlinePlayersMarker 判断日志行是否为指定标记行。
// DST 的日志行尾可能带 tab 等空白字符，比较前去掉首尾空白。
func isOnlinePlayersMarker(line, marker string) bool {
	suffix, ok := onlinePlayersLineSuffix(line)
	if !ok {
		return false
	}

	return strings.TrimSpace(suffix) == marker
}

// onlinePlayerName 从一条玩家行中取出昵称，非玩家行返回 false。
func onlinePlayerName(line string) (string, bool) {
	suffix, ok := onlinePlayersLineSuffix(line)
	if !ok || !strings.HasPrefix(suffix, onlinePlayersLinePrefix) {
		return "", false
	}

	name := strings.TrimSpace(suffix[len(onlinePlayersLinePrefix):])
	if name == "" {
		return "", false
	}

	return name, true
}
