package dst

import (
	"bufio"
	"bytes"
	"context"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fsnotify/fsnotify"
)

func (g *Game) getLogContent(logType string, id, lines int) []string {
	logPath := g.LogContentPath(logType, id)
	logger.Logger.Debug(logPath)
	if logPath == "" {
		return []string{}
	}

	return utils.GetFileLastNLines(logPath, lines)
}

// LogContentPath 返回当前日志文件路径。game 取指定世界的 server_log.txt；
// chat 取运行中世界的 server_chat_log.txt，没有世界运行时回退到主世界（再回退到第一个世界），
// 便于世界停止后仍能查看上一次的聊天记录。路径不存在时由调用方处理。
func (g *Game) LogContentPath(logType string, id int) string {
	switch logType {
	case "game":
		world, err := g.getWorldByID(id)
		if err != nil {
			return ""
		}
		return filepath.Join(world.worldPath, "server_log.txt")
	case "chat":
		for i := range g.worldSaveData {
			if g.worldUpStatus(g.worldSaveData[i].ID) {
				return filepath.Join(g.worldSaveData[i].worldPath, "server_chat_log.txt")
			}
		}
		for i := range g.worldSaveData {
			if g.worldSaveData[i].IsMaster {
				return filepath.Join(g.worldSaveData[i].worldPath, "server_chat_log.txt")
			}
		}
		if len(g.worldSaveData) > 0 {
			return filepath.Join(g.worldSaveData[0].worldPath, "server_chat_log.txt")
		}
	}

	return ""
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

const maxLogLineSize = 1024 * 1024

// logTailReadChunkSize 尾部定位与追读时单次读取的字节数。
const logTailReadChunkSize = 32 * 1024

type logTailState struct {
	file    *os.File
	info    os.FileInfo
	offset  int64
	pending []byte
}

func (s *logTailState) close() {
	if s != nil && s.file != nil {
		_ = s.file.Close()
	}
}

func openLogTail(logPath string, lines int, initial bool) (*logTailState, []string, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, nil, err
	}

	state := &logTailState{file: file}
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
	scanner.Buffer(make([]byte, 64*1024), maxLogLineSize)
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

func refreshChatLogTail(ctx context.Context, logPath string, state **logTailState, output chan<- string) error {
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
		*state, _, err = openLogTail(logPath, 0, false)
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

func drainChatLogTail(ctx context.Context, state *logTailState, output chan<- string) error {
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

func emitChatLogLines(ctx context.Context, state *logTailState, data []byte, output chan<- string) error {
	state.pending = append(state.pending, data...)
	if len(state.pending) > maxLogLineSize {
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

// logTailPollInterval 日志流检查文件变化的间隔。
// 用轮询而不是 fsnotify，既能在日志文件（甚至目录）尚未创建时等待，
// 也能统一处理文件截断、删除与替换。
const logTailPollInterval = time.Second

// maxLogChunkLines 单条事件最多携带的日志行数，避免一次推送过大。
const maxLogChunkLines = 200

// LogChunk 是一批日志内容。Initial 为 true 表示这是流开始或日志文件重建时的尾部快照，
// 调用方应替换已展示的内容；否则应把 Lines 追加到末尾。
type LogChunk struct {
	Initial bool     `json:"initial"`
	Lines   []string `json:"lines"`
}

// TailLog 先输出文件最后 lines 行，之后持续输出新增的完整行。
// 文件不存在时保持等待，创建后输出尾部快照；文件被截断、删除或替换后会自动重新读取。
// 调用方应在不再需要时取消 ctx，并持续消费 output，避免阻塞日志读取。
func TailLog(ctx context.Context, logPath string, lines int, output chan<- LogChunk) error {
	if ctx == nil {
		return fmt.Errorf("context 不能为空")
	}
	if logPath == "" {
		return fmt.Errorf("日志路径不能为空")
	}
	if lines < 0 {
		return fmt.Errorf("日志行数不能小于 0")
	}
	if output == nil {
		return fmt.Errorf("日志输出通道不能为空")
	}

	state, initial, err := openLogTailAt(logPath, lines)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if err != nil {
		initial = []string{}
	}
	defer func() {
		if state != nil {
			state.close()
		}
	}()

	if err = sendLogChunk(ctx, output, LogChunk{Initial: true, Lines: initial}); err != nil {
		return err
	}

	ticker := time.NewTicker(logTailPollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err = syncLogTail(ctx, logPath, lines, &state, output); err != nil {
				return err
			}
		}
	}
}

// openLogTailAt 打开日志文件，返回最后 lines 行，并把读取位置定位到文件末尾。
func openLogTailAt(logPath string, lines int) (*logTailState, []string, error) {
	file, err := os.Open(logPath)
	if err != nil {
		return nil, nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, nil, fmt.Errorf("获取日志信息失败: %w", err)
	}
	lastLines, offset := utils.GetFileLastNLinesWithOffset(logPath, lines)
	// 文件非空但没拿到读取位置，说明尾部读取失败，避免从文件头重放整份日志。
	if offset == 0 && info.Size() > 0 {
		_ = file.Close()
		return nil, nil, fmt.Errorf("读取日志尾部失败")
	}
	lastLines, offset = trimIncompleteTail(file, offset, lastLines)
	if lastLines == nil {
		lastLines = []string{}
	}

	return &logTailState{file: file, info: info, offset: offset}, lastLines, nil
}

// trimIncompleteTail 在文件末尾没有换行时，把读取位置回退到最后一行开头，并把它从快照中移除。
// 这样半行不会先出现在快照里、换行到达后又多出一个空行，而是在完整行写入后再推送。
func trimIncompleteTail(file *os.File, size int64, lines []string) ([]string, int64) {
	if size == 0 || len(lines) == 0 {
		return lines, size
	}

	last := make([]byte, 1)
	if _, err := file.ReadAt(last, size-1); err != nil || last[0] == '\n' {
		return lines, size
	}

	buffer := make([]byte, logTailReadChunkSize)
	for pos := size; pos > 0; {
		readSize := int64(len(buffer))
		if pos < readSize {
			readSize = pos
		}
		pos -= readSize
		n, err := file.ReadAt(buffer[:readSize], pos)
		if idx := bytes.LastIndexByte(buffer[:n], '\n'); idx >= 0 {
			return lines[:len(lines)-1], pos + int64(idx) + 1
		}
		if err != nil && err != io.EOF {
			return lines, size
		}
	}

	return []string{}, 0
}

// syncLogTail 检查日志文件是否被创建、截断或替换，并推送新增的完整行。
func syncLogTail(ctx context.Context, logPath string, lines int, state **logTailState, output chan<- LogChunk) error {
	pathInfo, err := os.Stat(logPath)
	if os.IsNotExist(err) {
		if *state != nil {
			(*state).close()
			*state = nil
		}
		return nil
	}
	if err != nil {
		return fmt.Errorf("获取日志信息失败: %w", err)
	}

	if *state == nil || !os.SameFile((*state).info, pathInfo) {
		if *state != nil {
			(*state).close()
			*state = nil
		}
		newState, initial, openErr := openLogTailAt(logPath, lines)
		if openErr != nil {
			if os.IsNotExist(openErr) {
				return nil
			}
			return openErr
		}
		*state = newState
		return sendLogChunk(ctx, output, LogChunk{Initial: true, Lines: initial})
	}

	if pathInfo.Size() < (*state).offset {
		// 日志被清空或截断（例如清理日志）：先让前端清屏，再从文件头重新读取。
		if _, err = (*state).file.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("重置日志读取位置失败: %w", err)
		}
		(*state).offset = 0
		(*state).pending = nil
		if err = sendLogChunk(ctx, output, LogChunk{Initial: true, Lines: []string{}}); err != nil {
			return err
		}
	}

	(*state).info = pathInfo
	return drainLogChunk(ctx, *state, output)
}

// drainLogChunk 从当前读取位置输出新增的完整行，按批发送。
func drainLogChunk(ctx context.Context, state *logTailState, output chan<- LogChunk) error {
	if _, err := state.file.Seek(state.offset, io.SeekStart); err != nil {
		return fmt.Errorf("设置日志读取位置失败: %w", err)
	}

	buffer := make([]byte, 64*1024)
	lines := make([]string, 0, maxLogChunkLines)
	flush := func() error {
		if len(lines) == 0 {
			return nil
		}
		chunk := LogChunk{Lines: lines}
		lines = make([]string, 0, maxLogChunkLines)
		return sendLogChunk(ctx, output, chunk)
	}

	for {
		n, err := state.file.Read(buffer)
		if n > 0 {
			state.offset += int64(n)
			state.pending = append(state.pending, buffer[:n]...)
			if len(state.pending) > maxLogLineSize {
				return fmt.Errorf("日志单行内容超过限制")
			}
			start := 0
			for i, b := range state.pending {
				if b != '\n' {
					continue
				}
				lines = append(lines, strings.TrimSuffix(string(state.pending[start:i]), "\r"))
				start = i + 1
				if len(lines) >= maxLogChunkLines {
					if flushErr := flush(); flushErr != nil {
						return flushErr
					}
				}
			}
			if start > 0 {
				state.pending = append([]byte(nil), state.pending[start:]...)
			}
		}
		if err == io.EOF {
			return flush()
		}
		if err != nil {
			return fmt.Errorf("读取新增日志失败: %w", err)
		}
	}
}

func sendLogChunk(ctx context.Context, output chan<- LogChunk, chunk LogChunk) error {
	select {
	case output <- chunk:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
