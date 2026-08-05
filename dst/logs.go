package dst

import (
	"bufio"
	"context"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

func (g *Game) getLogContent(logType string, id, lines int) []string {
	var logPath string

	switch logType {
	case "game":
		world, err := g.getWorldByID(id)
		if err != nil {
			return []string{}
		}
		logPath = fmt.Sprintf("%s/server_log.txt", world.worldPath)
		logger.Logger.Debug(logPath)
	case "chat":
		for _, world := range g.worldSaveData {
			if g.worldUpStatus(world.ID) {
				logPath = fmt.Sprintf("%s/server_chat_log.txt", world.worldPath)
				break
			}
		}
	default:
		return []string{}
	}

	logger.Logger.Debug(logPath)
	if logPath == "" {
		return []string{}
	}

	return utils.GetFileLastNLines(logPath, lines)
}

func (g *Game) historyFileList(logType string, id int) []string {
	var logPath string

	switch logType {
	case "game":
		world, err := g.getWorldByID(id)
		if err != nil {
			return []string{}
		}
		logPath = fmt.Sprintf("%s/backup/server_log", world.worldPath)
		logger.Logger.Debug(logPath)
	case "chat":
		for _, world := range g.worldSaveData {
			if g.worldUpStatus(world.ID) {
				logPath = fmt.Sprintf("%s/backup/server_chat_log", world.worldPath)
				break
			}
		}
	default:
		return []string{}
	}

	files, err := utils.GetFiles(logPath)
	if err != nil {
		return []string{}
	}

	return files
}

func (g *Game) historyFileContent(logType, logfileName string, id int) string {
	var logPath string

	// 防止路径穿越攻击：仅使用文件名部分，去除任何目录组件
	safeFileName := filepath.Base(logfileName)

	switch logType {
	case "game":
		world, err := g.getWorldByID(id)
		if err != nil {
			return ""
		}
		logPath = fmt.Sprintf("%s/backup/server_log/%s", world.worldPath, safeFileName)
		logger.Logger.Debug(logPath)
	case "chat":
		for _, world := range g.worldSaveData {
			if g.worldUpStatus(world.ID) {
				logPath = fmt.Sprintf("%s/backup/server_chat_log/%s", world.worldPath, safeFileName)
				break
			}
		}
	default:
		return ""
	}

	content, err := utils.GetFileAllContent(logPath)
	if err != nil {
		return ""
	}

	return content
}

type LogInfo struct {
	Game    int64 `json:"game"`
	Chat    int64 `json:"chat"`
	Steam   int64 `json:"steam"`
	Access  int64 `json:"access"`
	Runtime int64 `json:"runtime"`
}

func (g *Game) logsInfo() LogInfo {
	var logInfo LogInfo
	for _, world := range g.worldSaveData {
		size, err := utils.GetDirSize(fmt.Sprintf("%s/backup/server_log", world.worldPath))
		if err == nil {
			logInfo.Game = logInfo.Game + size
		}
		size, err = utils.GetDirSize(fmt.Sprintf("%s/backup/server_chat_log", world.worldPath))
		if err == nil {
			logInfo.Chat = logInfo.Chat + size
		}
	}
	steamSize, err := utils.GetFileSize("Steam/logs/bootstrap_log.txt")
	if err == nil {
		logInfo.Steam = logInfo.Steam + steamSize
	}
	accessSize, err := utils.GetFileSize("logs/access.log")
	if err == nil {
		logInfo.Access = logInfo.Access + accessSize
	}
	runtimeSize, err := utils.GetFileSize("logs/runtime.log")
	if err == nil {
		logInfo.Runtime = logInfo.Runtime + runtimeSize
	}

	return logInfo
}

type CleanLogs struct {
	Game    bool `json:"game"`
	Chat    bool `json:"chat"`
	Steam   bool `json:"steam"`
	Access  bool `json:"access"`
	Runtime bool `json:"runtime"`
}

func (g *Game) logsClean(cleanLogs *CleanLogs) bool {
	allSuccess := true

	if cleanLogs.Game {
		for _, world := range g.worldSaveData {
			err := utils.RemoveDir(fmt.Sprintf("%s/backup/server_log", world.worldPath))
			if err != nil {
				allSuccess = false
				logger.Logger.Errorf("删除游戏日志失败, err: %v", err)
			}
		}
	}
	if cleanLogs.Chat {
		for _, world := range g.worldSaveData {
			err := utils.RemoveDir(fmt.Sprintf("%s/backup/server_chat_log", world.worldPath))
			if err != nil {
				allSuccess = false
				logger.Logger.Errorf("删除聊天日志失败, err: %v", err)
			}
		}
	}
	if cleanLogs.Steam {
		err := utils.TruncAndWriteFile("Steam/logs/bootstrap_log.txt", "")
		if err != nil {
			allSuccess = false
			logger.Logger.Errorf("删除Steam日志失败, err: %v", err)
		}
	}
	if cleanLogs.Access {
		err := utils.TruncAndWriteFile("logs/access.log", "")
		if err != nil {
			allSuccess = false
			logger.Logger.Errorf("删除请求日志失败, err: %v", err)
		}
	}
	if cleanLogs.Runtime {
		err := utils.TruncAndWriteFile("logs/runtime.log", "")
		if err != nil {
			allSuccess = false
			logger.Logger.Errorf("删除运行日志失败, err: %v", err)
		}
	}

	return allSuccess
}

func (g *Game) logsList(admin bool) []string {
	var files []string

	for _, world := range g.worldSaveData {
		files = append(files, fmt.Sprintf("%s/server_log.txt", world.worldPath))
	}

	if admin {
		files = append(files, "logs/access.log", "logs/runtime.log")
	}

	return files
}

func (g *Game) tailChatLog(ctx context.Context, lines int, output chan<- string) error {
	if ctx == nil {
		return fmt.Errorf("context 不能为空")
	}
	if lines < 0 {
		return fmt.Errorf("日志行数不能小于 0")
	}
	if output == nil {
		return fmt.Errorf("日志输出通道不能为空")
	}
	if len(g.worldSaveData) == 0 {
		return fmt.Errorf("房间中不存在世界")
	}

	world := &g.worldSaveData[0]
	for i := range g.worldSaveData {
		if g.worldSaveData[i].IsMaster {
			world = &g.worldSaveData[i]
			break
		}
	}

	logPath := filepath.Join(world.worldPath, "server_chat_log.txt")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("创建聊天日志监听器失败: %w", err)
	}
	defer watcher.Close()

	if err = watcher.Add(filepath.Dir(logPath)); err != nil {
		return fmt.Errorf("监听聊天日志目录失败: %w", err)
	}

	state, initialLines, err := openChatLogTail(logPath, lines, true)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	defer func() {
		if state != nil {
			state.close()
		}
	}()
	for _, line := range initialLines {
		if err = sendChatLogLine(ctx, output, line); err != nil {
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("聊天日志监听器已关闭")
			}
			if filepath.Clean(event.Name) != filepath.Clean(logPath) {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
				continue
			}
			if err = refreshChatLogTail(ctx, logPath, &state, output); err != nil {
				return err
			}
		case watchErr, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("聊天日志监听器错误通道已关闭")
			}
			logger.Logger.Warnf("聊天日志监听事件异常，尝试重新同步文件: %v", watchErr)
			if err = refreshChatLogTail(ctx, logPath, &state, output); err != nil {
				return err
			}
		}
	}
}

const maxChatLogLineSize = 1024 * 1024

type chatLogTailState struct {
	file    *os.File
	info    os.FileInfo
	offset  int64
	pending []byte
}

func (s *chatLogTailState) close() {
	if s != nil && s.file != nil {
		_ = s.file.Close()
	}
}

func openChatLogTail(logPath string, lines int, initial bool) (*chatLogTailState, []string, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, nil, err
	}

	state := &chatLogTailState{file: file}
	info, err := file.Stat()
	if err != nil {
		state.close()
		return nil, nil, fmt.Errorf("获取聊天日志信息失败: %w", err)
	}
	state.info = info

	if !initial {
		return state, nil, nil
	}
	if lines == 0 {
		state.offset, err = file.Seek(0, io.SeekEnd)
		if err != nil {
			state.close()
			return nil, nil, fmt.Errorf("定位聊天日志末尾失败: %w", err)
		}
		return state, nil, nil
	}

	lastLines := make([]string, 0, lines)
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), maxChatLogLineSize)
	for scanner.Scan() {
		if len(lastLines) >= lines {
			lastLines = lastLines[1:]
		}
		lastLines = append(lastLines, scanner.Text())
	}
	if err = scanner.Err(); err != nil {
		state.close()
		return nil, nil, fmt.Errorf("读取聊天日志最后几行失败: %w", err)
	}

	state.offset, err = file.Seek(0, io.SeekCurrent)
	if err != nil {
		state.close()
		return nil, nil, fmt.Errorf("获取聊天日志读取位置失败: %w", err)
	}
	return state, lastLines, nil
}

func refreshChatLogTail(ctx context.Context, logPath string, state **chatLogTailState, output chan<- string) error {
	pathInfo, err := os.Stat(logPath)
	if os.IsNotExist(err) {
		if *state != nil {
			(*state).close()
			*state = nil
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("获取聊天日志信息失败: %w", err)
	}

	if *state == nil || !os.SameFile((*state).info, pathInfo) {
		if *state != nil {
			(*state).close()
		}
		*state, _, err = openChatLogTail(logPath, 0, false)
		if err != nil {
			return fmt.Errorf("重新打开聊天日志失败: %w", err)
		}
	} else if pathInfo.Size() < (*state).offset {
		if _, err = (*state).file.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("重置聊天日志读取位置失败: %w", err)
		}
		(*state).offset = 0
		(*state).pending = nil
	}

	(*state).info = pathInfo
	return drainChatLogTail(ctx, *state, output)
}

func drainChatLogTail(ctx context.Context, state *chatLogTailState, output chan<- string) error {
	if _, err := state.file.Seek(state.offset, io.SeekStart); err != nil {
		return fmt.Errorf("设置聊天日志读取位置失败: %w", err)
	}

	buffer := make([]byte, 64*1024)
	for {
		n, err := state.file.Read(buffer)
		if n > 0 {
			state.offset += int64(n)
			if emitErr := emitChatLogLines(ctx, state, buffer[:n], output); emitErr != nil {
				return emitErr
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("读取新增聊天日志失败: %w", err)
		}
	}
}

func emitChatLogLines(ctx context.Context, state *chatLogTailState, data []byte, output chan<- string) error {
	state.pending = append(state.pending, data...)
	if len(state.pending) > maxChatLogLineSize {
		return fmt.Errorf("聊天日志单行内容超过限制")
	}

	start := 0
	for i, b := range state.pending {
		if b != '\n' {
			continue
		}
		line := strings.TrimSuffix(string(state.pending[start:i]), "\r")
		if err := sendChatLogLine(ctx, output, line); err != nil {
			return err
		}
		start = i + 1
	}

	if start > 0 {
		state.pending = append([]byte(nil), state.pending[start:]...)
	}
	return nil
}

func sendChatLogLine(ctx context.Context, output chan<- string, line string) error {
	select {
	case output <- line:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
