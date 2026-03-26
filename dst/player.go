package dst

import (
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type playerSaveData struct {
	whitelist     []string
	blocklist     []string
	adminlist     []string
	whitelistPath string
	blocklistPath string
	adminlistPath string
}

func getPlayerList(filepath string) []string {
	// 预留位 黑名单 管理员
	err := utils.EnsureFileExists(filepath)
	if err != nil {
		logger.Logger.Error("创建文件失败", "err", err, "file", filepath)
		return []string{}
	}
	al, err := utils.ReadLinesToSlice(filepath)
	if err != nil {
		logger.Logger.Error("读取文件失败", "err", err, "file", filepath)
		return []string{}
	}
	var uidList []string
	for _, uid := range al {
		logger.Logger.Debug(uid)
		if uid == "" || strings.HasPrefix(uid, " ") {
			continue
		}

		uidList = append(uidList, uid)
	}

	return uidList
}

func (g *Game) savePlayerList() error {
	// 先去重
	adminlist := utils.RemoveDuplicates(g.adminlist)
	whitelist := utils.RemoveDuplicates(g.whitelist)
	blocklist := utils.RemoveDuplicates(g.blocklist)

	var err error
	err = utils.WriteLinesFromSlice(g.adminlistPath, adminlist)
	if err != nil {
		return err
	}
	err = utils.WriteLinesFromSlice(g.blocklistPath, blocklist)
	if err != nil {
		return err
	}
	err = utils.WriteLinesFromSlice(g.whitelistPath, whitelist)
	if err != nil {
		return err
	}

	return nil
}

func (g *Game) addPlayerList(uids []string, listType string) error {
	switch listType {
	case "adminlist":
		g.playerSaveData.adminlist = append(g.playerSaveData.adminlist, uids...)
		return g.savePlayerList()
	case "blocklist":
		g.playerSaveData.blocklist = append(g.playerSaveData.blocklist, uids...)
		return g.savePlayerList()
	case "whitelist":
		g.playerSaveData.whitelist = append(g.playerSaveData.whitelist, uids...)
		err := g.savePlayerList()
		if err != nil {
			return err
		}
		return g.createRoom() // 不统一处理，提升性能
	}

	return fmt.Errorf("类型错误")
}

func (g *Game) removePlayerList(uid, listType string) error {
	switch listType {
	case "adminlist":
		if !utils.Contains(g.playerSaveData.adminlist, uid) {
			return nil
		}
		g.playerSaveData.adminlist = utils.RemoveItem(g.playerSaveData.adminlist, uid)
		return g.savePlayerList()
	case "blocklist":
		if !utils.Contains(g.playerSaveData.blocklist, uid) {
			return nil
		}
		g.playerSaveData.blocklist = utils.RemoveItem(g.playerSaveData.blocklist, uid)
		return g.savePlayerList()
	case "whitelist":
		if !utils.Contains(g.playerSaveData.whitelist, uid) {
			return nil
		}
		g.playerSaveData.whitelist = utils.RemoveItem(g.playerSaveData.whitelist, uid)
		err := g.savePlayerList()
		if err != nil {
			return err
		}
		return g.createRoom() // 不统一处理，提升性能
	}

	return fmt.Errorf("类型错误")
}

type ChatMessage struct {
	Time        int64  `json:"time"`
	TimeSeconds int64  `json:"timeSeconds"`
	UID         string `json:"uid"`
	Nickname    string `json:"nickname"`
	Message     string `json:"message"`
	Type        string `json:"type"`
}

type ChatLogLine struct {
	TimeSeconds int64  `json:"timeSeconds"`
	Type        string `json:"type"`
	Message     string `json:"message"`
}

func getDstStartTime(filepath string) (time.Time, error) {
	lines := utils.GetFileFirstNLines(filepath, 10)

	timeFormat := "Mon Jan 2 15:04:05 2006"

	// 使用正则表达式匹配 Current time: 后面的时间字符串
	re := regexp.MustCompile(`Current time:\s*(.+)`)

	for _, line := range lines {
		matches := re.FindStringSubmatch(line)
		if len(matches) > 1 {
			timeStr := strings.TrimSpace(matches[1])
			parsedTime, err := time.ParseInLocation(timeFormat, timeStr, time.Local)
			if err == nil {
				logger.Logger.DebugF("解析到的时间为：%s", parsedTime.Format(time.RFC3339))
				return parsedTime, nil
			}
			logger.Logger.Warn("解析时间失败", "line", line, "error", err)
		}
	}

	return time.Time{}, fmt.Errorf("未找到有效的时间信息")
}

func parseChatLogLine(line string) (*ChatLogLine, error) {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil, fmt.Errorf("空行")
	}

	// 正则表达式匹配：[时间]: [类型] 消息
	// 匹配模式：\[([^\]]+)\]:\s*\[([^\]]+)\]\s*(.*)
	re := regexp.MustCompile(`\[([^\]]+)\]:\s*\[([^\]]+)\]\s*(.*)`)
	matches := re.FindStringSubmatch(line)

	if len(matches) != 4 {
		return nil, fmt.Errorf("无法解析的行：%s", line)
	}

	timeStr := matches[1] // HH:MM:SS
	typeStr := matches[2] // Join Announcement, Leave Announcement 等
	message := matches[3] // 具体的消息内容

	// 将时间转换为秒数
	timeSeconds, err := timeToSeconds(timeStr)
	if err != nil {
		return nil, fmt.Errorf("时间转换失败：%v", err)
	}

	return &ChatLogLine{
		TimeSeconds: timeSeconds,
		Type:        typeStr,
		Message:     message,
	}, nil
}

func timeToSeconds(timeStr string) (int64, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("无效的时间格式：%s", timeStr)
	}

	var hours, minutes, seconds int
	_, err := fmt.Sscanf(parts[0], "%d", &hours)
	if err != nil {
		return 0, err
	}
	_, err = fmt.Sscanf(parts[1], "%d", &minutes)
	if err != nil {
		return 0, err
	}
	_, err = fmt.Sscanf(parts[2], "%d", &seconds)
	if err != nil {
		return 0, err
	}

	totalSeconds := hours*3600 + minutes*60 + seconds

	return int64(totalSeconds), nil
}

func (g *Game) chatMessages(lines int, needTime bool) ([]ChatMessage, error) {
	var (
		chatMessages    []ChatMessage
		serverStartTime time.Time
		err             error
		day             int64 // 游戏启动的第几天，如果期间聊天日志超过24小时未刷新，则会出现时间异常
	)

	rePlayerChat := regexp.MustCompile(`\(([^)]+)\)\s+([^:]+):\s*(.+)`)

	world := g.worldSaveData[0]

	chatLogPath := fmt.Sprintf("%s/server_chat_log.txt", world.worldPath)

	serverLogPath := fmt.Sprintf("%s/server_log.txt", world.worldPath)

	if needTime {
		serverStartTime, err = getDstStartTime(serverLogPath)
		if err != nil {
			return chatMessages, err
		}
	}

	chatLog, err := utils.ReadLinesToSlice(chatLogPath)
	if err != nil {
		return chatMessages, err
	}

	for _, line := range chatLog {
		parsed, err := parseChatLogLine(line)
		if err != nil {
			continue
		}

		if needTime {
			if len(chatMessages) > 0 {
				if chatMessages[len(chatMessages)-1].TimeSeconds > parsed.TimeSeconds {
					day++
				}
			}
		}

		chatMessage := ChatMessage{
			Time:        serverStartTime.Unix() + int64(day*24*3600) + parsed.TimeSeconds,
			TimeSeconds: parsed.TimeSeconds,
			Type:        parsed.Type,
		}

		switch parsed.Type {
		case "Say":
			matches := rePlayerChat.FindStringSubmatch(parsed.Message)
			if matches != nil && len(matches) >= 4 {
				chatMessage.UID = matches[1]
				chatMessage.Nickname = matches[2]
				chatMessage.Message = matches[3]
			}
		case "Skin Announcement":
			parts := strings.Split(parsed.Message, " ")
			if len(parts) > 0 {
				if len(parts) == 2 {
					chatMessage.Nickname = parts[0]
					chatMessage.Message = parts[1]
				}
				if len(parts) > 2 {
					// 处理玩家游戏昵称中含有空格的情况
					skinName := parts[len(parts)-1]
					nickname := parts[:len(parts)-1]

					chatMessage.Nickname = strings.Join(nickname, " ")
					chatMessage.Message = skinName
				}
			}

		default:
			chatMessage.Message = parsed.Message
			chatMessage.Nickname = "DST"
		}

		chatMessage.Type = strings.ReplaceAll(chatMessage.Type, " ", "")

		chatMessages = append(chatMessages, chatMessage)
	}

	chatMessagesLength := len(chatMessages)
	if chatMessagesLength > lines {
		return chatMessages[chatMessagesLength-lines:], nil
	}

	return chatMessages, nil
}
