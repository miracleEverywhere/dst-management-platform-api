package utils

import (
	"archive/zip"
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

var StartTime = time.Now()

// EnsureDirExists 检查目录是否存在，如果不存在则创建
func EnsureDirExists(dirPath string) error {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		err = os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("无法创建目录: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("检查目录时出错: %w", err)
	}

	return nil
}

// EnsureFileExists 检查文件是否存在，如果不存在则创建空文件
func EnsureFileExists(filePath string) error {
	// 检查文件是否存在
	_, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		// 文件不存在，创建一个空文件
		file, err := os.Create(filePath)
		if err != nil {
			return err
		}
		err = file.Close()
		if err != nil {
			return err
		}
	} else if err != nil {
		// 其他错误
		return err
	}

	return nil
}

// FileDirectoryExists 检查文件或目录是否存在
func FileDirectoryExists(filePath string) bool {
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	} else if err != nil {
		return false
	} else {
		return true
	}
}

// TruncAndWriteFile 将指定内容完整写入文件，如果文件已存在会清空原有内容，如果文件不存在会创建新文件
func TruncAndWriteFile(fileName string, fileContent string) error {
	fileContentByte := []byte(fileContent)
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("打开或创建文件时出错: %w", err)
	}
	defer file.Close()

	// 写入新数据
	_, err = file.Write(fileContentByte)
	if err != nil {
		return fmt.Errorf("写入数据时出错: %w", err)
	}

	return nil
}

// RemoveDir 删除目录
func RemoveDir(dirPath string) error {
	// 调用 os.RemoveAll 删除目录及其所有内容
	err := os.RemoveAll(dirPath)
	if err != nil {
		return fmt.Errorf("删除目录失败: %w", err)
	}
	return nil
}

// RemoveFile 删除文件
func RemoveFile(filename string) error {
	err := os.Remove(filename)
	if err != nil {
		return fmt.Errorf("删除文件失败: %v", err)
	}
	return nil
}

// RemoveFilesOlderThan 删除指定目录下，修改时间大于days的文件
func RemoveFilesOlderThan(dirPath string, days int) (int, error) {
	// 计算截止时间（当前时间减去N天）
	cutoffTime := time.Now().AddDate(0, 0, -days)
	deletedFileCount := 0

	// 遍历目录
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 跳过目录本身
		if path == dirPath {
			return nil
		}
		// 只处理普通文件
		if !info.Mode().IsRegular() {
			return nil
		}
		// 获取文件修改时间
		fileTime := info.ModTime()
		// 检查文件是否早于截止时间
		if fileTime.Before(cutoffTime) {

			err := os.Remove(path)
			if err != nil {
				return fmt.Errorf("删除 %s: %v文件失败", path, err)
			} else {
				deletedFileCount++
			}
		}

		return nil
	})

	return deletedFileCount, err
}

// ReadLinesToSlice 文件内容按行读取到切片中
func ReadLinesToSlice(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// WriteLinesFromSlice 将切片内容按元素+\n写回文件
func WriteLinesFromSlice(filePath string, lines []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, _ = writer.WriteString(line + "\n")
	}
	return writer.Flush()
}

// GetFileAllContent 读取文件内容
func GetFileAllContent(filePath string) (string, error) {
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close() // 确保在函数结束时关闭文件
	// 创建一个Reader，可以使用任何实现了io.Reader接口的类型
	reader := file

	// 读取文件内容到byte切片中
	content, err := io.ReadAll(reader)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// StructToJsonFile 结构体保存到json文件
func StructToJsonFile[T any](filePath string, s T) error {
	data, err := json.MarshalIndent(s, "", "    ") // 格式化输出
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	file, err := os.OpenFile(filePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}
	// 确保数据刷入磁盘
	if err := file.Sync(); err != nil {
		return fmt.Errorf("同步文件到磁盘失败: %w", err)
	}

	return nil
}

// JsonFileToStruct 从JSON文件读取并解析到结构体
func JsonFileToStruct[T any](filePath string, s *T) error {
	// 读取 JSON 文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// 解析 JSON
	return json.Unmarshal(data, s)
}

// BashCMD 执行Linux Bash 命令
func BashCMD(cmd string) error {
	cmdExec := exec.Command("/bin/bash", "-c", cmd)
	err := cmdExec.Run()
	if err != nil {
		return err
	}
	return nil
}

// BashCMDOutput 执行Linux Bash 命令，并返回结果
func BashCMDOutput(cmd string) (string, string, error) {
	// 定义要执行的命令和参数
	cmdExec := exec.Command("/bin/bash", "-c", cmd)

	// 使用 bytes.Buffer 捕获命令的输出
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmdExec.Stdout = &stdout
	cmdExec.Stderr = &stderr

	// 执行命令
	err := cmdExec.Run()
	if err != nil {
		return "", stderr.String(), err
	}

	return stdout.String(), "", nil
}

// ScreenCMD 执行饥荒Console命令
func ScreenCMD(cmd string, screenName string) error {
	totalCMD := "screen -S \"" + screenName + "\" -p 0 -X stuff \"" + cmd + "\\n\""

	cmdExec := exec.Command("/bin/bash", "-c", totalCMD)
	err := cmdExec.Run()
	if err != nil {
		return err
	}
	return nil
}

// ScreenCMDOutput 执行饥荒Console命令，并从日志中获取输出
// 自动添加print命令，cmdIdentifier是该命令在日志中输出的唯一标识符
func ScreenCMDOutput(cmd string, cmdIdentifier string, screenName string, logPath string) (string, error) {
	totalCMD := "screen -S \"" + screenName + "\" -p 0 -X stuff \"print('" + cmdIdentifier + "' .. 'DMPSCREENCMD' .. tostring(" + cmd + "))\\n\""

	cmdExec := exec.Command("/bin/bash", "-c", totalCMD)
	err := cmdExec.Run()
	if err != nil {
		return "", err
	}

	// 等待日志打印
	time.Sleep(50 * time.Millisecond)

	logCmd := "tail -1000 " + logPath
	out, _, err := BashCMDOutput(logCmd)
	if err != nil {
		return "", err
	}

	for _, i := range strings.Split(out, "\n") {
		if strings.Contains(i, cmdIdentifier+"DMPSCREENCMD") {
			result := strings.Split(i, "DMPSCREENCMD")
			return strings.TrimSpace(result[1]), nil
		}
	}

	return "", fmt.Errorf("在日志中未找到对应输出")
}

// GetDirs 获取指定目录下的目录，不包含子目录和文件
func GetDirs(dirPath string, fullPath bool) ([]string, error) {
	var dirs []string
	// 如果路径中包含 ~，则将其替换为用户的 home 目录
	if strings.HasPrefix(dirPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return []string{}, err
		}
		dirPath = strings.Replace(dirPath, "~", homeDir, 1)
	}
	// 打开目录
	dir, err := os.Open(dirPath)
	if err != nil {
		return []string{}, err
	}
	defer dir.Close()

	// 读取目录条目
	entries, err := dir.Readdir(-1)
	if err != nil {
		return []string{}, err
	}

	// 遍历目录条目，只输出目录
	for _, entry := range entries {
		if entry.IsDir() {
			if fullPath {
				lastChar := string([]rune(dirPath)[len([]rune(dirPath))-1])
				if lastChar != "/" {
					dirs = append(dirs, dirPath+"/"+entry.Name())
				} else {
					dirs = append(dirs, dirPath+entry.Name())
				}
			} else {
				dirs = append(dirs, entry.Name())
			}
		}
	}
	return dirs, nil
}

// GetFiles 递归地获取指定目录下的所有文件名
func GetFiles(dirPath string) ([]string, error) {
	var fileNames []string

	err := filepath.WalkDir(dirPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			fileNames = append(fileNames, d.Name())
		}
		return nil
	})

	if err != nil {
		return []string{}, err
	}

	return fileNames, nil
}

// GetDirSize 计算目录大小
func GetDirSize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// GetFileSize 文件大小
func GetFileSize(filePath string) (int64, error) {
	// 使用 os.Stat 获取文件信息
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return 0, err
	}

	// 获取文件大小
	fileSize := fileInfo.Size()

	return fileSize, nil
}

// ChangeFileMode 修改文件权限
func ChangeFileMode(filepath string, mod os.FileMode) error {
	return os.Chmod(filepath, mod)
}

// Zip 压缩文件或目录
func Zip(source, target string) error {
	// 创建目标ZIP文件
	zipFile, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("创建ZIP文件失败: %v", err)
	}
	defer zipFile.Close()

	// 创建ZIP写入器
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 获取源文件信息
	info, err := os.Stat(source)
	if err != nil {
		return fmt.Errorf("获取源文件信息失败: %v", err)
	}

	var baseDir string
	if info.IsDir() {
		baseDir = filepath.Base(source)
	}

	// 遍历文件并添加到ZIP
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 创建ZIP文件头
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// 设置文件头名称
		header.Name, err = filepath.Rel(filepath.Dir(source), path)
		if err != nil {
			return err
		}

		if baseDir != "" {
			header.Name = filepath.Join(baseDir, strings.TrimPrefix(path, source))
		}

		// 如果是目录，需要在名称后加斜杠
		if info.IsDir() {
			header.Name += "/"
		} else {
			// 设置压缩方法
			header.Method = zip.Deflate
		}

		// 创建ZIP文件条目
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// 如果是目录，不需要写入内容
		if info.IsDir() {
			return nil
		}

		// 打开源文件
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// 将文件内容复制到ZIP条目
		_, err = io.Copy(writer, file)
		return err
	})
}

// ZipFiles 压缩多个文件到指定ZIP文件中 files: 要压缩的文件路径列表（只包含文件，不包含目录）target: 压缩后的ZIP文件路径
func ZipFiles(files []string, target string) error {
	// 创建目标ZIP文件
	zipFile, err := os.Create(target)
	if err != nil {
		return fmt.Errorf("创建ZIP文件失败: %v", err)
	}
	defer zipFile.Close()

	// 创建ZIP写入器
	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// 遍历所有文件
	for _, filePath := range files {
		// 打开源文件
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("打开文件失败 %s: %v", filePath, err)
		}

		// 获取文件信息
		info, err := file.Stat()
		if err != nil {
			file.Close()
			return fmt.Errorf("获取文件信息失败 %s: %v", filePath, err)
		}

		// 验证是否是普通文件
		if !info.Mode().IsRegular() {
			file.Close()
			return fmt.Errorf("不是普通文件: %s", filePath)
		}

		// 创建ZIP文件头
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			file.Close()
			return fmt.Errorf("创建文件头失败 %s: %v", filePath, err)
		}

		// 设置文件在ZIP中的名称（只保留文件名）
		header.Name = filepath.Base(filePath)

		// 设置压缩方法
		header.Method = zip.Deflate

		// 创建ZIP文件条目
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			file.Close()
			return fmt.Errorf("创建ZIP条目失败 %s: %v", filePath, err)
		}

		// 将文件内容复制到ZIP条目
		_, err = io.Copy(writer, file)
		if err != nil {
			file.Close()
			return fmt.Errorf("写入文件内容失败 %s: %v", filePath, err)
		}

		file.Close()
	}

	return nil
}

// Unzip 解压ZIP文件
func Unzip(zipFile, dest string) error {
	// 打开ZIP文件
	reader, err := zip.OpenReader(zipFile)
	if err != nil {
		return fmt.Errorf("打开ZIP文件失败: %v", err)
	}
	defer reader.Close()

	// 创建目标目录
	if err := os.MkdirAll(dest, 0755); err != nil {
		return fmt.Errorf("创建目标目录失败: %v", err)
	}

	// 遍历ZIP文件中的每个条目
	for _, file := range reader.File {
		// 关键修复：将Windows风格的反斜杠路径转换为当前系统的路径分隔符
		name := file.Name

		// 替换所有反斜杠为正斜杠
		name = strings.ReplaceAll(name, "\\", "/")

		// 清理路径
		name = filepath.Clean(name)

		// 构建完整路径
		filePath := filepath.Join(dest, name)

		// 安全检查：防止路径遍历攻击
		cleanDest := filepath.Clean(dest)
		cleanFilePath := filepath.Clean(filePath)
		if !strings.HasPrefix(cleanFilePath, cleanDest+string(os.PathSeparator)) &&
			cleanFilePath != cleanDest {
			return fmt.Errorf("无效的文件路径: %s", filePath)
		}

		// 检查是否是目录
		isDir := file.FileInfo().IsDir() || strings.HasSuffix(file.Name, "/")

		if isDir {
			// 创建目录（包括所有父目录）
			if err := os.MkdirAll(filePath, 0755); err != nil {
				return fmt.Errorf("创建目录失败: %v", err)
			}
			continue
		}

		// 确保文件的父目录存在
		parentDir := filepath.Dir(filePath)
		if err := os.MkdirAll(parentDir, 0755); err != nil {
			return fmt.Errorf("创建父目录失败: %v", err)
		}

		// 创建目标文件
		outFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return fmt.Errorf("创建文件失败: %v", err)
		}

		// 打开ZIP中的文件
		rc, err := file.Open()
		if err != nil {
			outFile.Close()
			return fmt.Errorf("打开ZIP内文件失败: %v", err)
		}

		// 复制文件内容
		_, err = io.Copy(outFile, rc)

		// 关闭文件句柄
		outFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("写入文件失败: %v", err)
		}
	}

	return nil
}

// CpuUsage 获取cpu使用率
func CpuUsage() float64 {
	percent, err := cpu.Percent(0, false)
	if err != nil {
		return 0
	}
	return percent[0]
}

// MemoryUsage 获取内存使用率
func MemoryUsage() float64 {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return 0
	}
	return vmStat.UsedPercent
}

// NetStatus 获取网络使用情况
func NetStatus() (float64, float64) {
	// 获取初始的网络统计信息
	initialCounters, err := net.IOCounters(true)
	if err != nil {
		return 0, 0
	}

	// 记录初始时间
	initialTime := time.Now()

	// 等待0.5秒
	time.Sleep(500 * time.Millisecond)

	// 获取新的网络统计信息
	newCounters, err := net.IOCounters(true)
	if err != nil {
		return 0, 0
	}

	// 记录新时间
	newTime := time.Now()

	// 计算时间差（秒）
	timeDiff := newTime.Sub(initialTime).Seconds()

	// 计算所有接口的总数据
	var (
		totalSentBytes float64
		totalRecvBytes float64
	)
	for i, counter := range newCounters {
		if i < len(initialCounters) {
			sentBytes := float64(counter.BytesSent - initialCounters[i].BytesSent)
			recvBytes := float64(counter.BytesRecv - initialCounters[i].BytesRecv)
			totalSentBytes += sentBytes
			totalRecvBytes += recvBytes
		}
	}

	// 计算总数据速率（KB/s）
	totalSentKB := totalSentBytes / 1024.0
	totalUplinkKBps := totalSentKB / timeDiff
	totalRecvKB := totalRecvBytes / 1024.0
	totalDownlinkKBps := totalRecvKB / timeDiff

	return totalUplinkKBps, totalDownlinkKBps
}

// DiskUsage 获取当前分区磁盘使用率
func DiskUsage() float64 {
	// 获取当前目录
	currentDir, err := os.Getwd()
	if err != nil {
		return 0
	}

	// 获取当前目录所在的挂载点
	mountPoint := findMountPoint(currentDir)
	if mountPoint == "" {
		return 0
	}

	// 获取挂载点的磁盘使用情况
	usage, err := disk.Usage(mountPoint)
	if err != nil {
		return 0
	}
	return usage.UsedPercent
}

func findMountPoint(path string) string {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return ""
	}

	for {
		partitions, err := disk.Partitions(false)
		if err != nil {
			return ""
		}

		for _, partition := range partitions {
			if isSubPath(absPath, partition.Mountpoint) {
				return partition.Mountpoint
			}
		}

		// 向上遍历目录
		parent := filepath.Dir(absPath)
		if parent == absPath {
			break
		}
		absPath = parent
	}

	return ""
}

func isSubPath(path, mountpoint string) bool {
	rel, err := filepath.Rel(mountpoint, path)
	if err != nil {
		return false
	}
	return !strings.Contains(rel, "..")
}

// GetFileLastNLines 获取文件的最后N行，返回字符串切片
func GetFileLastNLines(filename string, n int) []string {
	file, err := os.Open(filename)
	if err != nil {
		return []string{}
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:] // 移除前面的行，保持最后 n 行
		}
	}

	if err := scanner.Err(); err != nil {
		return []string{}
	}

	return lines
}
