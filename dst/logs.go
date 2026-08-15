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

func (g *Game) tailGameLog(ctx context.Context, logFileName string, lines int, output chan<- string) error {
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

	logPath := filepath.Join(world.worldPath, logFileName)
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("创建%s日志监听器失败: %w", logFileName, err)
	}
	defer watcher.Close()

	if err = watcher.Add(filepath.Dir(logPath)); err != nil {
		return fmt.Errorf("监听%s日志目录失败: %w", logFileName, err)
	}

	state, initialLines, err := openLogTail(logPath, lines, true)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	defer func() {
		if state != nil {
			state.close()
		}
	}()
	for _, line := range initialLines {
		if err = sendLogLine(ctx, output, line); err != nil {
			return err
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("%s日志监听器已关闭", logFileName)
			}
			if filepath.Clean(event.Name) != filepath.Clean(logPath) {
				continue
			}
			if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) == 0 {
				continue
			}
			if err = refreshLogTail(ctx, logPath, &state, output); err != nil {
				return err
			}
		case watchErr, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("%s日志监听器错误通道已关闭", logFileName)
			}
			logger.Logger.Warnf("%s日志监听事件异常，尝试重新同步文件: %v", logFileName, watchErr)
			if err = refreshLogTail(ctx, logPath, &state, output); err != nil {
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

func openLogTail(logPath string, lines int, initial bool) (*chatLogTailState, []string, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, nil, err
	}

	state := &chatLogTailState{file: file}
	info, err := file.Stat()
	if err != nil {
		state.close()
		return nil, nil, fmt.Errorf("获取日志信息失败: %w", err)
	}
	state.info = info

	if !initial {
		return state, nil, nil
	}
	if lines == 0 {
		state.offset, err = file.Seek(0, io.SeekEnd)
		if err != nil {
			state.close()
			return nil, nil, fmt.Errorf("定位日志末尾失败: %w", err)
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
		return nil, nil, fmt.Errorf("读取日志最后几行失败: %w", err)
	}

	// 显式将指针和 offset 强制对齐到物理文件末尾
	// 此时 lastLines 包含了截至 SeekEnd 前的最后 N 行，后续 drain 绝对只读 SeekEnd 之后的新数据
	state.offset, err = file.Seek(0, io.SeekEnd)
	if err != nil {
		state.close()
		return nil, nil, fmt.Errorf("获取日志读取位置失败: %w", err)
	}
	return state, lastLines, nil
}

// 强制刷盘：在文件轮转/关闭时，将 pending 中遗留的半行数据作为最后一行强行发送
func (s *chatLogTailState) flushPending(ctx context.Context, output chan<- string) error {
	if len(s.pending) == 0 {
		return nil
	}
	line := strings.TrimSuffix(string(s.pending), "\r")
	s.pending = nil // 清空缓存
	return sendLogLine(ctx, output, line)
}

func refreshLogTail(ctx context.Context, logPath string, state **chatLogTailState, output chan<- string) error {
	pathInfo, err := os.Stat(logPath)
	if os.IsNotExist(err) {
		if *state != nil {
			// 文件被删：先读完新追加的，再强行刷出残留的半行，最后关闭
			if err := drainLogTail(ctx, *state, output); err != nil {
				fmt.Printf("[Warn] 移除旧文件前 drain 失败, path: %s, err: %v", logPath, err)
			}
			if err := (*state).flushPending(ctx, output); err != nil {
				fmt.Printf("[Warn] 移除旧文件前 flush 失败, path: %s, err: %v", logPath, err)
			}
			(*state).close()
			*state = nil
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("获取日志信息失败: %w", err)
	}

	// 检测到文件轮转（Inode 改变）或者第一次初始化
	if *state == nil || !os.SameFile((*state).info, pathInfo) {
		if *state != nil {
			// 1. 旧文件收尾：如果在 drain 或 flush 时遇到 ctx 取消，应当直接返回 error 终止轮转
			if err := drainLogTail(ctx, *state, output); err != nil {
				fmt.Printf("[Error] 日志轮转读取旧文件剩余内容失败: %v", err)
				if ctx.Err() != nil {
					return ctx.Err() // 如果是 context 取消导致，直接中断
				}
			}
			if err := (*state).flushPending(ctx, output); err != nil {
				fmt.Printf("[Error] 日志轮转刷新旧文件末尾半行失败: %v", err)
				if ctx.Err() != nil {
					return ctx.Err()
				}
			}

			// 2. 正常关闭旧文件
			(*state).close()
		}

		// 3. 打开新文件
		*state, _, err = openLogTail(logPath, 0, false)
		if err != nil {
			return fmt.Errorf("重新打开日志失败: %w", err)
		}
	} else if pathInfo.Size() < (*state).offset {
		// 文件被截断 (Truncated)
		if _, err = (*state).file.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("重置日志读取位置失败: %w", err)
		}
		(*state).offset = 0
		(*state).pending = nil
	}

	(*state).info = pathInfo
	return drainLogTail(ctx, *state, output)
}

func drainLogTail(ctx context.Context, state *chatLogTailState, output chan<- string) error {
	// 只有在初始时确保指针对齐一次即可，循环内部绝不 Seek
	buffer := make([]byte, 64*1024)
	for {
		n, err := state.file.Read(buffer) // Read 会自动、连续地推进文件指针
		if n > 0 {
			state.offset += int64(n) // 内存记录仅作状态标记
			if emitErr := emitLogLines(ctx, state, buffer[:n], output); emitErr != nil {
				return emitErr
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return fmt.Errorf("读取新增日志失败: %w", err)
		}
	}
}

func emitLogLines(ctx context.Context, state *chatLogTailState, data []byte, output chan<- string) error {
	state.pending = append(state.pending, data...)

	start := 0
	for i, b := range state.pending {
		if b != '\n' {
			continue
		}
		line := strings.TrimSuffix(string(state.pending[start:i]), "\r")
		if err := sendLogLine(ctx, output, line); err != nil {
			// ✅ 回滚：丢弃已发送的行，保留未发送的部分（含当前失败行）
			n := copy(state.pending, state.pending[start:])
			state.pending = state.pending[:n]
			return err
		}
		start = i + 1
	}

	// 提取完所有完整行后，平移剩余的半行数据 (零内存分配)
	if start > 0 {
		n := copy(state.pending, state.pending[start:])
		state.pending = state.pending[:n]
	}

	// 修正：只有在【处理完完整行后】，剩下的未完成单行超过限制才报错
	if len(state.pending) > maxChatLogLineSize {
		return fmt.Errorf("日志单行内容超过限制")
	}

	return nil
}

func sendLogLine(ctx context.Context, output chan<- string, line string) error {
	select {
	case output <- line:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
