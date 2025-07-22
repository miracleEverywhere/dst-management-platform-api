package utils

import (
	"bufio"
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	cRand "crypto/rand"
	"crypto/sha256"
	"encoding/base32"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/yuin/gopher-lua"
	"io"
	"io/fs"
	"math"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	nonceSize   = 1 // 1字节Nonce（8位）
	sigSize     = 4 // 4字节签名（32位）
	maxSigChars = 7 // Base32编码后最多7字符
)

var (
	BindPort      int
	ConsoleOutput bool
	VersionShow   bool
	ConfDir       string
	ConfigMutex   sync.Mutex
)

type Claims struct {
	Username string `json:"username"`
	Nickname string `json:"nickname"`
	Role     string `json:"role"`
	jwt.StandardClaims
}

type OSInfo struct {
	Architecture    string
	OS              string
	CPUModel        string
	CPUCores        int
	MemorySize      uint64
	Platform        string
	PlatformVersion string
	Uptime          uint64
}

type SeasonLength struct {
	Summer int `json:"summer"`
	Autumn int `json:"autumn"`
	Spring int `json:"spring"`
	Winter int `json:"winter"`
}

type SeasonI18N struct {
	En string `json:"en"`
	Zh string `json:"zh"`
}

type MetaInfo struct {
	Cycles       int          `json:"cycles"`
	Phase        SeasonI18N   `json:"phase"`
	Season       SeasonI18N   `json:"season"`
	ElapsedDays  int          `json:"elapsedDays"`
	SeasonLength SeasonLength `json:"seasonLength"`
}

func SetGlobalVariables() {
	config, err := ReadConfig()
	if err != nil {
		Logger.Error("启动检查出现致命错误：获取数据库失败", "err", err)
		panic(err)
	}

	HomeDir, err = os.UserHomeDir()
	if err != nil {
		Logger.Error("无法获取用户HOME目录", "err", err)
		panic("无法获取用户HOME目录")
	}

	osInfo, err := GetOSInfo()
	if err != nil {
		Logger.Error("启动检查出现致命错误：获取系统信息失败", "err", err)
		panic(err)
	}
	Platform = osInfo.Platform

	Registered = config.Registered

	// 设置全局用户缓存
	for _, user := range config.Users {
		UserCache[user.Username] = user
	}

	// 查看是否在容器内
	_, InContainer = os.LookupEnv("DMP_IN_CONTAINER")
}

func GenerateJWTSecret() string {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length := 26
	randomString := make([]byte, length)
	for i := range randomString {
		// 从字符集中随机选择一个字符
		randomString[i] = charset[r.Intn(len(charset))]
	}

	return string(randomString)
}

func GenerateJWT(user User, jwtSecret []byte, expiration int) (string, error) {
	// 定义一个自定义的声明结构

	claims := Claims{
		Username: user.Username,
		Nickname: user.Nickname,
		Role:     user.Role,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(expiration) * time.Hour).Unix(), // 过期时间
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ReadUidMap(cluster Cluster) (map[string]interface{}, error) {
	uidMap := make(map[string]interface{})
	content, err := os.ReadFile(cluster.GetUIDMapFile())
	if err != nil {
		// 如果打开文件失败，则初始化json文件
		err = EnsureDirExists(UidFilePath)
		if err != nil {
			return uidMap, err
		}
		err = EnsureFileExists(cluster.GetUIDMapFile())
		if err != nil {
			return uidMap, err
		}
		err = TruncAndWriteFile(cluster.GetUIDMapFile(), "{}")
		if err != nil {
			return uidMap, err
		}
	}
	jsonData := string(content)
	err = json.Unmarshal([]byte(jsonData), &uidMap)
	if err != nil {
		return uidMap, err
	}
	return uidMap, nil
}

func WriteUidMap(uidMap map[string]interface{}, cluster Cluster) error {
	data, err := json.MarshalIndent(uidMap, "", "    ") // 格式化输出
	if err != nil {
		return fmt.Errorf("Error marshalling JSON:" + err.Error())
	}
	file, err := os.OpenFile(cluster.GetUIDMapFile(), os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("Error opening file:" + err.Error())
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			Logger.Error("关闭文件失败", "err", err)
		}
	}(file) // 在函数结束时关闭文件
	// 写入 JSON 数据到文件
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("Error writing to file:" + err.Error())
	}
	return nil
}

func CreateManualInstallScript() {
	var (
		manualInstallScript string
		err                 error
	)

	if Platform == "darwin" {
		manualInstallScript = ManualInstallMac
	} else {
		manualInstallScript = ManualInstall
	}

	//创建手动安装脚本
	err = TruncAndWriteFile("manual_install.sh", manualInstallScript)
	if err != nil {
		Logger.Error("手动安装脚本创建失败", "err", err)
	}
	err = BashCMD("chmod +x manual_install.sh")
	if err != nil {
		Logger.Error("手动安装脚本添加执行权限失败", "err", err)
	}
}

func CheckDirs() {
	var err error
	// dst config
	err = EnsureDirExists(DstPath)
	if err != nil {
		Logger.Error("目录检查未通过", "path", DstPath)
		panic("目录检查未通过")
	}
	// dmp_files
	err = EnsureDirExists(DmpFilesPath)
	if err != nil {
		Logger.Error("目录检查未通过", "path", DmpFilesPath)
		panic("目录检查未通过")
	}
	err = EnsureDirExists(ImportFileUploadPath)
	if err != nil {
		Logger.Error("目录检查未通过", "path", ImportFileUploadPath)
		panic("目录检查未通过")
	}
	// mod下载目录
	err = EnsureDirExists(ModUgcDownloadPath)
	if err != nil {
		Logger.Error("目录检查未通过", "path", ModUgcDownloadPath)
		panic("目录检查未通过")
	}
	err = EnsureDirExists(ModNoUgcDownloadPath)
	if err != nil {
		Logger.Error("目录检查未通过", "path", ModNoUgcDownloadPath)
		panic("目录检查未通过")
	}
	// 备份目录
	err = EnsureDirExists(BackupPath)
	if err != nil {
		Logger.Error("目录检查未通过", "path", BackupPath)
		panic("目录检查未通过")
	}
}

func BindFlags() {
	flag.IntVar(&BindPort, "l", 80, "监听端口，如： -l 8080 (Listening Port, e.g. -l 8080)")
	flag.StringVar(&ConfDir, "s", "./", "数据库文件目录，如： -s ./conf (Database Directory, e.g. -s ./conf)")
	flag.BoolVar(&ConsoleOutput, "c", false, "开启控制台日志输出，如： -c (Enable console log output, e.g. -c)")
	flag.BoolVar(&VersionShow, "v", false, "查看版本，如： -v (Check version, e.g. -v)")
	flag.Parse()
}

func GetOSInfo() (*OSInfo, error) {
	architecture := runtime.GOARCH

	// 获取CPU信息
	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, err
	}
	cpuModel := cpuInfo[0].ModelName
	cpuCount, _ := cpu.Counts(true)
	cpuCore := cpuCount

	// 获取内存信息
	virtualMemory, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	memorySize := virtualMemory.Total

	// 获取主机信息
	hostInfo, err := host.Info()
	if err != nil {
		return nil, err
	}
	platformVersion := hostInfo.PlatformVersion
	platform := hostInfo.Platform
	uptime := hostInfo.Uptime
	osName := hostInfo.OS
	// 返回系统信息
	return &OSInfo{
		Architecture:    architecture,
		OS:              osName,
		CPUModel:        cpuModel,
		CPUCores:        cpuCore,
		MemorySize:      memorySize,
		Platform:        platform,
		Uptime:          uptime,
		PlatformVersion: platformVersion,
	}, nil
}

func TruncAndWriteFile(fileName string, fileContent string) error {
	fileContentByte := []byte(fileContent)
	file, err := os.OpenFile(fileName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		return fmt.Errorf("打开或创建文件时出错: %w", err)
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			Logger.Error("关闭文件失败", "err", err)
		}
	}(file) // 确保在函数结束时关闭文件

	// 写入新数据
	_, err = file.Write(fileContentByte)
	if err != nil {
		return fmt.Errorf("写入数据时出错: %w", err)
	}

	return nil
}

func DeleteDir(dirPath string) error {
	err := os.RemoveAll(dirPath)
	if err != nil {
		return fmt.Errorf("删除目录失败: %w", err)
	}

	return nil
}

func ReadContainCpuUsage() (uint64, error) {
	file, err := os.Open("/sys/fs/cgroup/cpu.stat")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "usage_usec") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				return strconv.ParseUint(parts[1], 10, 64)
			}
		}
	}

	return 0, fmt.Errorf("未找到 usage_usec 数据")
}

func ReadContainUintFromFile(path string) (uint64, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}

	valueStr := strings.TrimSpace(string(data))
	if valueStr == "max" {
		return math.MaxUint64, nil
	}

	return strconv.ParseUint(valueStr, 10, 64)
}

func CpuUsage() (float64, error) {
	// 获取 CPU 使用率
	if InContainer {
		const samplingInterval = 100 * time.Millisecond // 0.1秒
		// 第一次采样
		usage1, err := ReadContainCpuUsage()
		if err != nil {
			return 0, err
		}
		// 等待 0.1 秒
		time.Sleep(samplingInterval)
		// 第二次采样
		usage2, err := ReadContainCpuUsage()
		if err != nil {
			return 0, err
		}
		// 计算 CPU 使用率百分比
		delta := usage2 - usage1
		intervalMicroseconds := float64(samplingInterval.Microseconds())
		return float64(delta) / intervalMicroseconds * 100, nil
	} else {
		percent, err := cpu.Percent(0, false)
		if err != nil {
			return 0, fmt.Errorf("error getting CPU percent: %w", err)
		}
		return percent[0], nil
	}
}

func MemoryUsage() (float64, error) {
	// 获取内存信息
	if InContainer {
		// 读取内存限制
		data, err := os.ReadFile("/sys/fs/cgroup/memory.max")
		if err != nil {
			return 0, err
		}
		valueStr := strings.TrimSpace(string(data))
		if valueStr == "max" {
			// 没有内存限制
			vmStat, err := mem.VirtualMemory()
			if err != nil {
				return 0, fmt.Errorf("error getting virtual memory info: %w", err)
			}
			return vmStat.UsedPercent, nil
		} else {
			// 存在内存限制
			// 读取当前内存使用量
			memCurrent, err := ReadContainUintFromFile("/sys/fs/cgroup/memory.current")
			if err != nil {
				return 0, err
			}
			// 读取内存限制
			memMax, err := ReadContainUintFromFile("/sys/fs/cgroup/memory.max")
			if err != nil {
				return 0, err
			}
			return float64(memCurrent) / float64(memMax) * 100, nil
		}

	} else {
		vmStat, err := mem.VirtualMemory()
		if err != nil {
			return 0, fmt.Errorf("error getting virtual memory info: %w", err)
		}
		return vmStat.UsedPercent, nil
	}
}

func NetStatus() (float64, float64, error) {
	if InContainer {
		const samplingInterval = 500 * time.Millisecond // 0.5秒
		const interfaceName = "eth0"
		// 第一次采样
		rx1, err := ReadContainUintFromFile(fmt.Sprintf("/sys/class/net/%s/statistics/rx_bytes", interfaceName))
		if err != nil {
			return 0, 0, err
		}
		tx1, err := ReadContainUintFromFile(fmt.Sprintf("/sys/class/net/%s/statistics/tx_bytes", interfaceName))
		if err != nil {
			return 0, 0, err
		}
		// 等待 0.1 秒
		time.Sleep(samplingInterval)
		// 第二次采样
		rx2, err := ReadContainUintFromFile(fmt.Sprintf("/sys/class/net/%s/statistics/rx_bytes", interfaceName))
		if err != nil {
			return 0, 0, err
		}
		tx2, err := ReadContainUintFromFile(fmt.Sprintf("/sys/class/net/%s/statistics/tx_bytes", interfaceName))
		if err != nil {
			return 0, 0, err
		}
		// 计算流量速率 (KB/s)
		intervalSeconds := samplingInterval.Seconds()
		rxRate := float64(rx2-rx1) / 1024 / intervalSeconds
		txRate := float64(tx2-tx1) / 1024 / intervalSeconds

		return txRate, rxRate, nil
	} else {
		// 获取初始的网络统计信息
		initialCounters, err := net.IOCounters(true)
		if err != nil {
			return 0, 0, fmt.Errorf("error getting initial network counters: %v", err)
		}

		// 记录初始时间
		initialTime := time.Now()

		// 等待0.5秒
		time.Sleep(500 * time.Millisecond)

		// 获取新的网络统计信息
		newCounters, err := net.IOCounters(true)
		if err != nil {
			return 0, 0, fmt.Errorf("error getting new network counters: %v", err)
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

		return totalUplinkKBps, totalDownlinkKBps, nil
	}
}

func DiskUsage() (float64, error) {
	// 获取当前目录
	currentDir, err := os.Getwd()
	if err != nil {
		return 0, fmt.Errorf("error getting current directory: %w", err)
	}

	// 获取当前目录所在的挂载点
	mountPoint := findMountPoint(currentDir)
	if mountPoint == "" {
		return 0, fmt.Errorf("unable to find mount point for current directory")
	}

	// 获取挂载点的磁盘使用情况
	usage, err := disk.Usage(mountPoint)
	if err != nil {
		return 0, fmt.Errorf("error getting usage for %s: %w", mountPoint, err)
	}
	return usage.UsedPercent, nil
}

// 查找当前目录所在的挂载点
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

// 检查路径是否是挂载点的子路径
func isSubPath(path, mountpoint string) bool {
	rel, err := filepath.Rel(mountpoint, path)
	if err != nil {
		return false
	}
	return !strings.Contains(rel, "..")
}

func ScreenCMD(cmd string, screenName string) error {
	totalCMD := "screen -S \"" + screenName + "\" -p 0 -X stuff \"" + cmd + "\\n\""

	cmdExec := exec.Command("/bin/bash", "-c", totalCMD)
	err := cmdExec.Run()
	if err != nil {
		return err
	}
	return nil
}

// ScreenCMDOutput 执行screen命令，并从日志中获取输出
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

func BashCMD(cmd string) error {
	cmdExec := exec.Command("/bin/bash", "-c", cmd)
	err := cmdExec.Run()
	if err != nil {
		return err
	}
	return nil
}

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

// UniqueSliceKeepOrderString 从一个字符串切片中移除重复的元素，并保持元素的原始顺序
func UniqueSliceKeepOrderString(slice []string) []string {
	encountered := map[string]bool{}
	var result []string

	for _, value := range slice {
		if !encountered[value] {
			encountered[value] = true
			result = append(result, value)
		}
	}

	return result
}

func RemoveDir(dirPath string) error {
	// 调用 os.RemoveAll 删除目录及其所有内容
	err := os.RemoveAll(dirPath)
	if err != nil {
		return fmt.Errorf("删除目录失败: %w", err)
	}
	return nil
}

func RemoveFile(filePath string) error {
	// 删除文件
	err := os.Remove(filePath)
	if err != nil {
		return fmt.Errorf("删除文件失败: %w", err)
	}
	return nil
}

// EnsureDirExists 检查目录是否存在，如果不存在则创建
func EnsureDirExists(dirPath string) error {
	if strings.HasPrefix(dirPath, "~") {
		dirPath = strings.Replace(dirPath, "~", HomeDir, 1)
	}
	// 检查目录是否存在
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// 目录不存在，创建目录
		err := os.MkdirAll(dirPath, os.ModePerm)
		if err != nil {
			return fmt.Errorf("无法创建目录: %w", err)
		}
	} else if err != nil {
		// 其他错误
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
func FileDirectoryExists(filePath string) (bool, error) {
	// 如果路径中包含 ~，则将其替换为用户的 home 目录
	if strings.HasPrefix(filePath, "~") {
		filePath = strings.Replace(filePath, "~", HomeDir, 1)
	}
	// 检查文件是否存在
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return true, nil
	}
}

func BackupGame(cluster Cluster) error {
	err := EnsureDirExists(cluster.GetBackupPath())
	if err != nil {
		return err
	}

	var (
		filePath   string
		sessionErr error
		seasonInfo MetaInfo
		cycles     int
	)

	for _, world := range cluster.Worlds {
		sessionPath := world.GetSessionPath(cluster.ClusterSetting.ClusterName)
		filePath, sessionErr = FindLatestMetaFile(sessionPath)
		if sessionErr == nil {
			break
		}
	}

	if sessionErr != nil {
		seasonInfo, _ = GetMetaInfo("")
		Logger.Error("查询session-meta文件失败", "err", sessionErr)
		cycles = 0
	} else {
		seasonInfo, err = GetMetaInfo(filePath)
		if err != nil {
			Logger.Error("获取meta文件内容失败", "err", err)
			cycles = 0
		} else {
			cycles = seasonInfo.Cycles
		}

	}

	config, err := ReadConfig()
	if err != nil {
		Logger.Error("配置文件读取失败", "err", err)
		return err
	}

	// 删除敏感数据
	config.JwtSecret = ""
	config.Users = nil
	err = WriteBackupConfig(config)
	if err != nil {
		Logger.Error("写入备份配置文件失败", "err", err)
		return err
	}

	currentTime := time.Now().Format("2006-01-02-15-04-05")
	filename := fmt.Sprintf("%s_%d.tgz", currentTime, cycles)
	cmd := fmt.Sprintf("tar zcvf %s/%s %s %s/DstMP.sdb", cluster.GetBackupPath(), filename, cluster.GetMainPath(), BackupPath)
	err = BashCMD(cmd)
	if err != nil {
		return err
	}

	cmd = fmt.Sprintf("rm -f %s/DstMP.sdb", BackupPath)
	err = BashCMD(cmd)
	if err != nil {
		Logger.Error("删除备份配置文件失败", "err", err)
	}

	return nil
}

func (world World) StopGame() error {
	var err error
	if world.GetStatus() {
		err = ScreenCMD("c_shutdown()", world.ScreenName)
		if err != nil {
			Logger.Info("执行ScreenCMD失败", "msg", err, "cmd", "c_shutdown()")
		}
		time.Sleep(1 * time.Second)
	}

	killCMD := fmt.Sprintf("ps -ef | grep %s | grep dontstarve_dedicated_server_nullrenderer | grep -v grep | awk '{print $2}' | xargs kill -9", world.ScreenName)
	err = BashCMD(killCMD)
	if err != nil {
		Logger.Info("执行Bash命令失败", "msg", err, "cmd", killCMD)
	}

	_ = BashCMD("screen -wipe")
	_ = BashCMD(fmt.Sprintf("rm -f %s/.screen/*.%s", HomeDir, world.ScreenName))

	return err
}

func StopClusterAllWorlds(cluster Cluster) error {
	var err error
	for _, world := range cluster.Worlds {
		err = world.StopGame()
		if err != nil {
			Logger.Warn("关闭游戏失败", "集群", cluster.ClusterSetting.ClusterName, "世界", world.Name)
		}
	}

	return err
}

func StopAllClusters(clusters []Cluster) error {
	var err error
	for _, cluster := range clusters {
		err = StopClusterAllWorlds(cluster)
	}

	return err
}

func (world World) StartGame(clusterName, mod string, bit64 bool) error {
	var (
		cmd string
		err error
	)
	if Platform == "darwin" {
		cmd = fmt.Sprintf("cd dst/dontstarve_dedicated_server_nullrenderer.app/Contents/MacOS && export DYLD_LIBRARY_PATH=$DYLD_LIBRARY_PATH:$HOME/steamcmd && screen -d -h 200 -m -S %s ./dontstarve_dedicated_server_nullrenderer -console -cluster %s  -shard %s  ;", world.ScreenName, clusterName, world.Name)
		err = BashCMD(cmd)
		if err != nil {
			Logger.Error("执行BashCMD失败", "err", err, "cmd", cmd)
		}
	} else {
		_ = ReplaceDSTSOFile()
		err = DstModsSetup(mod)
		if err != nil {
			Logger.Error("设置mod下载配置失败", "err", err)
		}
		if bit64 {
			cmd = fmt.Sprintf("cd ~/dst/bin64/ && screen -d -h 200 -m -S %s ./dontstarve_dedicated_server_nullrenderer_x64 -console -cluster %s  -shard %s  ;", world.ScreenName, clusterName, world.Name)
		} else {
			cmd = fmt.Sprintf("cd ~/dst/bin/ && screen -d -h 200 -m -S %s ./dontstarve_dedicated_server_nullrenderer -console -cluster %s  -shard %s  ;", world.ScreenName, clusterName, world.Name)
		}
		err = BashCMD(cmd)
		if err != nil {
			Logger.Error("执行BashCMD失败", "err", err, "cmd", cmd)
		}
	}

	return err
}

func StartClusterAllWorlds(cluster Cluster) error {
	var err error
	_ = BashCMD("screen -wipe")
	time.Sleep(500 * time.Millisecond)
	for _, world := range cluster.Worlds {
		if world.GetStatus() {
			continue
		}
		err = world.StartGame(cluster.ClusterSetting.ClusterName, cluster.Mod, cluster.SysSetting.Bit64)
		if err != nil {
			Logger.Error("启动游戏失败", "集群", cluster.ClusterSetting.ClusterName, "世界", world.Name)
		}
		time.Sleep(500 * time.Millisecond)
	}

	return err
}

func StartAllClusters(clusters []Cluster) error {
	var err error
	for _, cluster := range clusters {
		err = StartClusterAllWorlds(cluster)
	}

	return err
}

// ClearDstFiles 删除脏数据
func (cluster Cluster) ClearDstFiles() error {
	var (
		err      error
		dbWorlds []string
	)

	allWorlds, err := GetDirs(cluster.GetMainPath(), false)
	if err != nil {
		return err
	}

	for _, world := range cluster.Worlds {
		dbWorlds = append(dbWorlds, world.Name)
	}

	for _, dirWorld := range allWorlds {
		if !Contains(dbWorlds, dirWorld) {
			err = RemoveDir(fmt.Sprintf("%s/%s", cluster.GetMainPath(), dirWorld))
		}
	}

	return err
}

func GetTimestamp() int64 {
	now := time.Now()
	// 获取毫秒级时间戳
	milliseconds := now.UnixNano() / int64(time.Millisecond)
	return milliseconds
}

func TimestampToTimestring(ts int64) string {
	//YYYY-MM-DD HH:mm
	t := time.Unix(ts/1000, (ts%1000)*int64(time.Millisecond))
	return t.Format("2006-01-02 15:04")
}

// GetFileAllContent 读取文件内容
func GetFileAllContent(filePath string) (string, error) {
	// 如果路径中包含 ~，则将其替换为用户的 home 目录
	if strings.HasPrefix(filePath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			Logger.Error("无法获取 home 目录", "err", err)
			return "", err
		}
		filePath = strings.Replace(filePath, "~", homeDir, 1)
	}
	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		Logger.Error("打开"+filePath+"文件失败", "err", err)
		return "", err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			Logger.Error("关闭"+filePath+"文件失败", "err", err)
		}
	}(file) // 确保在函数结束时关闭文件
	// 创建一个Reader，可以使用任何实现了io.Reader接口的类型
	reader := file

	// 读取文件内容到byte切片中
	content, err := io.ReadAll(reader)
	if err != nil {
		Logger.Error("读取"+filePath+"文件失败", "err", err)
		return "", err
	}
	return string(content), nil
}

// GetDirs 获取指定目录下的目录，不包含子目录和文件
func GetDirs(dirPath string, fullPath bool) ([]string, error) {
	var dirs []string
	// 如果路径中包含 ~，则将其替换为用户的 home 目录
	if strings.HasPrefix(dirPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			Logger.Error("无法获取 home 目录", "err", err)
			return []string{}, err
		}
		dirPath = strings.Replace(dirPath, "~", homeDir, 1)
	}
	// 打开目录
	dir, err := os.Open(dirPath)
	if err != nil {
		Logger.Error("打开目录失败", "err", err)
		return []string{}, err
	}
	defer func(dir *os.File) {
		err := dir.Close()
		if err != nil {
			Logger.Error("关闭目录失败", "err", err)
		}
	}(dir)

	// 读取目录条目
	entries, err := dir.Readdir(-1)
	if err != nil {
		Logger.Error("读取目录失败", "err", err)
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

func Bool2String(b bool, lang string) string {
	switch lang {
	case "lua":
		if b {
			return "true"
		} else {
			return "false"
		}
	case "python":
		if b {
			return "True"
		} else {
			return "False"
		}

	default:
		return "false"
	}
}

func ReplaceDSTSOFile() error {
	err := BashCMD("mv ~/dst/bin/lib32/steamclient.so ~/dst/bin/lib32/steamclient.so.bak")
	if err != nil {
		return err
	}
	err = BashCMD("cp ~/steamcmd/linux32/steamclient.so ~/dst/bin/lib32/steamclient.so")
	if err != nil {
		return err
	}

	err = BashCMD("mv ~/dst/bin64/lib64/steamclient.so ~/dst/bin64/lib64/steamclient.so.bak")
	if err != nil {
		return err
	}
	err = BashCMD("cp ~/steamcmd/linux64/steamclient.so ~/dst/bin64/lib64/steamclient.so")
	if err != nil {
		return err
	}

	return nil
}

// ExecBashScript 异步执行脚本
func ExecBashScript(scriptPath string, scriptContent string) {
	// 检查文件是否存在，如果存在则删除
	if _, err := os.Stat(scriptPath); err == nil {
		err := os.Remove(scriptPath)
		if err != nil {
			Logger.Error("删除文件失败", "err", err)
			return
		}
	}

	// 创建或打开文件
	file, err := os.OpenFile(scriptPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0775)
	if err != nil {
		Logger.Error("打开文件失败", "err", err)
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			Logger.Error("关闭文件失败", "err", err)
		}
	}(file)

	// 写入内容
	content := []byte(scriptContent)
	_, err = file.Write(content)
	if err != nil {
		Logger.Error("写入文件失败", "err", err)
		return
	}

	// 异步执行脚本
	go func() {
		cmd := exec.Command("/bin/bash", scriptPath) // 使用 /bin/bash 执行脚本
		e := cmd.Run()
		if e != nil {
			Logger.Error("执行安装脚本失败", "err", e)
		}
	}()
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

// GetLastDir 从路径中提取最后一个目录名，并检查是否以 "__" 开头
func GetLastDir(path string) string {
	return filepath.Base(path)
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

// CountFiles 递归统计目录中的文件数量
func CountFiles(path string) (int, error) {
	var fileCount int

	// 使用 filepath.Walk 遍历目录
	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 如果是文件而不是目录，增加计数器
		if !info.IsDir() {
			fileCount++
		}
		return nil
	})

	return fileCount, err
}

// Contains 是否含有元素
func Contains[T comparable](s []T, i T) bool {
	for _, v := range s {
		if v == i {
			return true
		}
	}

	return false
}

// RemoveSliceOne 删除切片中的一个元素
func RemoveSliceOne[T comparable](s []T, elem T) []T {
	for i, v := range s {
		if v == elem {
			return append(s[:i], s[i+1:]...)
		}
	}
	return s
}

func GetLastNElements[T any](slice []T, n int) []T {
	if n <= 0 {
		return nil
	}

	length := len(slice)
	if length <= n {
		return slice
	}

	return slice[length-n:]
}

func DstModsSetup(mod string) error {
	L := lua.NewState()
	defer L.Close()
	if err := L.DoString(mod); err != nil {
		Logger.Error("加载 Lua 文件失败:", "err", err)
		return err
	}
	modsTable := L.Get(-1)
	fileContent := ""
	if tbl, ok := modsTable.(*lua.LTable); ok {
		tbl.ForEach(func(key lua.LValue, value lua.LValue) {
			// 检查键是否是字符串，并且以 "workshop-" 开头
			if strKey, ok := key.(lua.LString); ok && strings.HasPrefix(string(strKey), "workshop-") {
				// 提取 "workshop-" 后面的数字
				workshopID := strings.TrimPrefix(string(strKey), "workshop-")
				fileContent = fileContent + "ServerModSetup(\"" + workshopID + "\")\n"
			}
		})
		var modFilePath string
		if Platform == "darwin" {
			modFilePath = MacGameModSettingPath
		} else {
			modFilePath = GameModSettingPath
		}
		err := TruncAndWriteFile(modFilePath, fileContent)
		if err != nil {
			Logger.Error("mod配置文件写入失败", "err", err, "file", modFilePath)
			return err
		}
	}

	return nil
}

// ReadLinesToSlice 文件内容按行读取到切片中
func ReadLinesToSlice(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			Logger.Error("关闭文件失败", "err", err)
		}
	}(file)

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
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			Logger.Error("关闭文件失败", "err", err)
		}
	}(file)

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, _ = writer.WriteString(line + "\n")
	}
	return writer.Flush()
}

// ParseIniToMap 将ini文件读取为map
func ParseIniToMap(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			Logger.Error("关闭文件失败")
		}
	}(file)

	configMap := make(map[string]string)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		// 检查是否是节标题
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			continue
		}

		// 解析键值对
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			configMap[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return configMap, nil
}

func GetWorldPortFactor(clusterName string) (int, error) {
	config, err := ReadConfig()
	if err != nil {
		return 0, err
	}

	for index, cluster := range config.Clusters {
		if cluster.ClusterSetting.ClusterName == clusterName {
			return index * 10, nil
		}
	}

	return 0, fmt.Errorf("没有对应的集群")
}

func GetFileLastNLines(filename string, n int) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			Logger.Error("文件关闭失败", "err", err)
		}
	}(file)

	var lines []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > n {
			lines = lines[1:] // 移除前面的行，保持最后 n 行
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

func GenerateUpdateModID() string {
	key := []byte("x")
	data := []byte("y")

	// 1. 生成随机Nonce（使用crypto/rand）
	nonce := make([]byte, nonceSize)
	if _, err := cRand.Read(nonce); err != nil {
		return ""
	}

	// 2. 计算 HMAC-SHA256(Nonce || data)
	h := hmac.New(sha256.New, key)
	h.Write(nonce)
	h.Write(data)
	sig := h.Sum(nil)[:sigSize] // 取前4字节

	// 3. Base32编码并截断
	combined := append(nonce, sig...)
	encoded := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(combined)
	if len(encoded) > maxSigChars {
		encoded = encoded[:maxSigChars]
	}
	return encoded
}

func VerifyUpdateModID(signature string) bool {
	key := []byte("x")
	data := []byte("y")
	// 1. 长度检查
	if len(signature) != maxSigChars {
		return false
	}

	// 2. Base32解码
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(signature)
	if err != nil {
		return false
	}

	// 3. 数据完整性检查
	if len(decoded) < nonceSize+sigSize/2 { // 至少需要Nonce+部分签名
		return false
	}

	// 4. 重新计算HMAC
	h := hmac.New(sha256.New, key)
	h.Write(decoded[:nonceSize])
	h.Write(data)
	expectedSig := h.Sum(nil)[:min(sigSize, len(decoded)-nonceSize)]

	// 5. 安全比对
	return hmac.Equal(expectedSig, decoded[nonceSize:])
}

func pkcs7Padding(data []byte, blockSize int) []byte {
	//判断缺少几位长度。最少1，最多 blockSize
	padding := blockSize - len(data)%blockSize
	//补足位数。把切片[]byte{byte(padding)}复制padding个
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(data, padText...)
}

// pkcs7UnPadding 填充的反向操作
func pkcs7UnPadding(data []byte) ([]byte, error) {
	length := len(data)
	if length == 0 {
		return nil, errors.New("加密字符串错误！")
	}
	//获取填充的个数
	unPadding := int(data[length-1])
	return data[:(length - unPadding)], nil
}

// AesEncrypt 加密
func AesEncrypt(data []byte, key []byte) ([]byte, error) {
	//创建加密实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//判断加密块的大小
	blockSize := block.BlockSize()
	//填充
	encryptBytes := pkcs7Padding(data, blockSize)
	//初始化加密数据接收切片
	crypted := make([]byte, len(encryptBytes))
	//使用cbc加密模式
	blockMode := cipher.NewCBCEncrypter(block, key[:blockSize])
	//执行加密
	blockMode.CryptBlocks(crypted, encryptBytes)
	return crypted, nil
}

// AesDecrypt 解密
func AesDecrypt(data, key []byte) ([]byte, error) {
	//创建实例
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	//获取块的大小
	blockSize := block.BlockSize()
	//使用cbc
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	//初始化解密数据接收切片
	crypted := make([]byte, len(data))
	//执行解密
	blockMode.CryptBlocks(crypted, data)
	//去除填充
	crypted, err = pkcs7UnPadding(crypted)
	if err != nil {
		return nil, err
	}
	return crypted, nil
}

func GetMetaInfo(path string) (MetaInfo, error) {
	var seasonInfo MetaInfo
	seasonInfo.Season.En = "Failed to retrieve"
	seasonInfo.Season.Zh = "获取失败"

	seasonInfo.Cycles = -1
	seasonInfo.Phase.En = "Failed to retrieve"
	seasonInfo.Phase.Zh = "获取失败"

	// 读取二进制文件
	data, err := os.ReadFile(path)
	if err != nil {
		return seasonInfo, fmt.Errorf("读取文件失败: %w", err)
	}

	// 创建 Lua 虚拟机
	L := lua.NewState()
	defer L.Close()

	// 将文件内容作为 Lua 代码执行
	content := string(data)
	content = content[:len(content)-1]

	err = L.DoString(content)
	if err != nil {
		return seasonInfo, fmt.Errorf("执行 Lua 代码失败: %w", err)
	}
	// 获取 Lua 脚本的返回值
	lv := L.Get(-1)
	if tbl, ok := lv.(*lua.LTable); ok {
		// 获取 clock 表
		clockTable := tbl.RawGet(lua.LString("clock"))
		if clock, ok := clockTable.(*lua.LTable); ok {
			// 获取 cycles 字段
			cycles := clock.RawGet(lua.LString("cycles"))
			if cyclesValue, ok := cycles.(lua.LNumber); ok {
				seasonInfo.Cycles = int(cyclesValue)
			}
			// 获取 phase 字段
			phase := clock.RawGet(lua.LString("phase"))
			if phaseValue, ok := phase.(lua.LString); ok {
				seasonInfo.Phase.En = string(phaseValue)
			}
		}
		// 获取 seasons 表
		seasonsTable := tbl.RawGet(lua.LString("seasons"))
		if seasons, ok := seasonsTable.(*lua.LTable); ok {
			// 获取 season 字段
			season := seasons.RawGet(lua.LString("season"))
			if seasonValue, ok := season.(lua.LString); ok {
				seasonInfo.Season.En = string(seasonValue)
			}
			// 获取 elapseddaysinseason 字段
			elapsedDays := seasons.RawGet(lua.LString("elapseddaysinseason"))
			if elapsedDaysValue, ok := elapsedDays.(lua.LNumber); ok {
				seasonInfo.ElapsedDays = int(elapsedDaysValue)
			}
			//获取季节长度
			lengthsTable := seasons.RawGet(lua.LString("lengths"))
			if lengths, ok := lengthsTable.(*lua.LTable); ok {
				summer := lengths.RawGet(lua.LString("summer"))
				if summerValue, ok := summer.(lua.LNumber); ok {
					seasonInfo.SeasonLength.Summer = int(summerValue)
				}
				autumn := lengths.RawGet(lua.LString("autumn"))
				if autumnValue, ok := autumn.(lua.LNumber); ok {
					seasonInfo.SeasonLength.Autumn = int(autumnValue)
				}
				spring := lengths.RawGet(lua.LString("spring"))
				if springValue, ok := spring.(lua.LNumber); ok {
					seasonInfo.SeasonLength.Spring = int(springValue)
				}
				winter := lengths.RawGet(lua.LString("winter"))
				if winterValue, ok := winter.(lua.LNumber); ok {
					seasonInfo.SeasonLength.Winter = int(winterValue)
				}

			}
		}
	}

	if seasonInfo.Phase.En == "night" {
		seasonInfo.Phase.Zh = "夜晚"
	}
	if seasonInfo.Phase.En == "day" {
		seasonInfo.Phase.Zh = "白天"
	}
	if seasonInfo.Phase.En == "dusk" {
		seasonInfo.Phase.Zh = "黄昏"
	}

	if seasonInfo.Season.En == "summer" {
		seasonInfo.Season.Zh = "夏天"
	}
	if seasonInfo.Season.En == "autumn" {
		seasonInfo.Season.Zh = "秋天"
	}
	if seasonInfo.Season.En == "spring" {
		seasonInfo.Season.Zh = "春天"
	}
	if seasonInfo.Season.En == "winter" {
		seasonInfo.Season.Zh = "冬天"
	}

	return seasonInfo, nil
}

func FindLatestMetaFile(directory string) (string, error) {
	// 检查指定目录是否存在
	_, err := os.Stat(directory)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("目录不存在：%s", directory)
	}

	// 获取指定目录下的所有子目录
	entries, err := os.ReadDir(directory)
	if err != nil {
		return "", fmt.Errorf("读取目录失败：%s", err)
	}

	// 用于存储最新的.meta文件路径和其修改时间
	var latestMetaFile string
	var latestMetaFileTime time.Time

	for _, entry := range entries {
		// 检查是否是目录
		if entry.IsDir() {
			subDirPath := filepath.Join(directory, entry.Name())

			// 获取子目录下的所有文件
			files, err := os.ReadDir(subDirPath)
			if err != nil {
				return "", fmt.Errorf("读取子目录失败：%s", err)
			}

			for _, file := range files {
				// 检查文件是否是.meta文件
				if !file.IsDir() && filepath.Ext(file.Name()) == ".meta" {
					// 获取文件的完整路径
					fullPath := filepath.Join(subDirPath, file.Name())

					// 获取文件的修改时间
					info, err := file.Info()
					if err != nil {
						return "", fmt.Errorf("获取文件信息失败：%s", err)
					}
					modifiedTime := info.ModTime()

					// 如果找到的文件的修改时间比当前最新的.meta文件的修改时间更晚，则更新最新的.meta文件路径和修改时间
					if modifiedTime.After(latestMetaFileTime) {
						latestMetaFile = fullPath
						latestMetaFileTime = modifiedTime
					}
				}
			}
		}
	}

	if latestMetaFile == "" {
		return "", fmt.Errorf("未找到.meta文件")
	}

	return latestMetaFile, nil
}
