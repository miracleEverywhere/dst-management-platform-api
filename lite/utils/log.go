package utils

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
)

const (
	// worldLogFileName 世界游戏日志文件名
	worldLogFileName = "server_log.txt"
	// worldChatLogFileName 世界聊天日志文件名
	worldChatLogFileName = "server_chat_log.txt"

	// logTailChunkSize 从文件尾部反向读取时单次读取的字节数
	logTailChunkSize = 32 * 1024
	// logTailMaxBytes 单次读取日志的最大字节数，异常超长行也不会让 RSS 失控
	logTailMaxBytes = 1024 * 1024
)

// WorldLogPath 返回世界游戏日志（server_log.txt）的路径。
// 世界名不合法（含目录成分）时返回空字符串。
func WorldLogPath(worldName string) string {
	return worldLogPath(worldName, worldLogFileName)
}

// WorldChatLogPath 返回世界聊天日志（server_chat_log.txt）的路径。
// 世界名不合法（含目录成分）时返回空字符串。
func WorldChatLogPath(worldName string) string {
	return worldLogPath(worldName, worldChatLogFileName)
}

// worldLogPath 拼接集群目录下某个世界的日志文件路径。
// 世界名会用于拼路径，必须限制为单级名称，防止读到集群目录外的文件。
func worldLogPath(worldName, fileName string) string {
	if worldName == "" || worldName == "." || worldName == ".." || filepath.Base(worldName) != worldName {
		return ""
	}

	return filepath.Join(clusterPath, worldName, fileName)
}

// ReadLastLines 从文件尾部反向按块读取最后 maxLines 行。
// 只读取文件尾部，且单次读取量不超过 logTailMaxBytes，避免日志变大时推高内存。
func ReadLastLines(filePath string, maxLines int) ([]string, error) {
	if maxLines <= 0 || filePath == "" {
		return []string{}, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		_ = file.Close()
	}(file)

	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if fileInfo.Size() == 0 {
		return []string{}, nil
	}

	var (
		chunks     [][]byte
		pos        = fileInfo.Size()
		totalSize  int64
		newlineCnt int
	)

	for pos > 0 {
		readSize := int64(logTailChunkSize)
		if pos < readSize {
			readSize = pos
		}
		pos -= readSize

		chunk := make([]byte, readSize)
		readN, readErr := file.ReadAt(chunk, pos)
		if readErr != nil && readErr != io.EOF {
			return nil, readErr
		}
		chunk = chunk[:readN]
		chunks = append(chunks, chunk)
		totalSize += int64(readN)
		for _, b := range chunk {
			if b == '\n' {
				newlineCnt++
			}
		}

		// 多读一个换行，保证最后 maxLines 行完整，缓冲区开头的半行会被裁掉
		if newlineCnt > maxLines || totalSize >= logTailMaxBytes {
			break
		}
	}

	// chunks 按文件尾到文件头收集，这里反转拼回连续字节
	buf := make([]byte, 0, totalSize)
	for i := len(chunks) - 1; i >= 0; i-- {
		buf = append(buf, chunks[i]...)
	}

	lines := bytes.Split(buf, []byte{'\n'})
	// 文件以换行结尾时末尾会多出一个空元素
	if len(lines) > 0 && len(lines[len(lines)-1]) == 0 {
		lines = lines[:len(lines)-1]
	}
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	} else if pos > 0 && len(lines) > 0 {
		// 读取量达到上限被截断时，首行可能是半行，丢弃
		lines = lines[1:]
	}

	result := make([]string, len(lines))
	for i, line := range lines {
		if n := len(line); n > 0 && line[n-1] == '\r' {
			line = line[:n-1]
		}
		result[i] = string(line)
	}

	return result, nil
}
