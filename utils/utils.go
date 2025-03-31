package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"io"
	"io/fs"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var (
	// STATISTICS 玩家统计
	STATISTICS []Statistics
	// SYS_METRICS 系统监控
	SYS_METRICS []SysMetrics
	// BindPort flag绑定的变量
	BindPort      int
	ConsoleOutput bool
	VersionShow   bool
	ConfDir       string
	PLATFORM      string
)

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

type RoomSettingBase struct {
	Name                    string `json:"name"`
	Description             string `json:"description"`
	GameMode                string `json:"gameMode"`
	PVP                     bool   `json:"pvp"`
	PlayerNum               int    `json:"playerNum"`
	BackDays                int    `json:"backDays"`
	Vote                    bool   `json:"vote"`
	Password                string `json:"password"`
	Token                   string `json:"token"`
	MasterPort              int    `json:"masterPort"`
	CavesPort               int    `json:"cavesPort"`
	ClusterKey              string `json:"clusterKey"`
	ShardMasterIp           string `json:"shardMasterIp"`
	ShardMasterPort         int    `json:"shardMasterPort"`
	SteamMasterPort         int    `json:"steamMasterPort"`
	SteamAuthenticationPort int    `json:"steamAuthenticationPort"`
}

type RoomSetting struct {
	Base   RoomSettingBase `json:"base"`
	Ground string          `json:"ground"`
	Cave   string          `json:"cave"`
	Mod    string          `json:"mod"`
}

type AutoUpdate struct {
	Enable bool   `json:"enable"`
	Time   string `json:"time"`
}

type AutoAnnounce struct {
	Name      string `json:"name"`
	Enable    bool   `json:"enable"`
	Content   string `json:"content"`
	Frequency int    `json:"frequency"`
}

type AutoBackup struct {
	Enable bool   `json:"enable"`
	Time   string `json:"time"`
}

type Players struct {
	UID      string `json:"uid"`
	NickName string `json:"nickName"`
	Prefab   string `json:"prefab"`
}

type Statistics struct {
	Timestamp int64     `json:"timestamp"`
	Num       int       `json:"num"`
	Players   []Players `json:"players"`
}

type SysMetrics struct {
	Timestamp   int64   `json:"timestamp"`
	Cpu         float64 `json:"cpu"`
	Memory      float64 `json:"memory"`
	NetUplink   float64 `json:"netUplink"`
	NetDownlink float64 `json:"netDownlink"`
}

type Keepalive struct {
	Enable        bool   `json:"enable"`
	Frequency     int    `json:"frequency"`
	LastTime      string `json:"lastTime"`
	CavesLastTime string `json:"cavesLastTime"`
}

type SchedulerSettingItem struct {
	// disable的原因是1.1.3版本之前都是默认打开的，新增配置后应该也是默认打开
	// 所以 disable=false
	Disable   bool `json:"disable"`
	Frequency int  `json:"frequency"`
}

type SchedulerSetting struct {
	PlayerGetFrequency int                  `json:"playerGetFrequency"`
	UIDMaintain        SchedulerSettingItem `json:"UIDMaintain"`
	SysMetricsGet      SchedulerSettingItem `json:"sysMetricsGet"`
}

type SysSetting struct {
	SchedulerSetting SchedulerSetting `json:"schedulerSetting"`
}

type EncodeUserPath struct {
	Ground bool `json:"ground"`
	Cave   bool `json:"cave"`
}

type Config struct {
	Username       string         `json:"username"`
	Nickname       string         `json:"nickname"`
	Password       string         `json:"password"`
	JwtSecret      string         `json:"jwtSecret"`
	RoomSetting    RoomSetting    `json:"roomSetting"`
	MultiHost      bool           `json:"multiHost"`
	AutoUpdate     AutoUpdate     `json:"autoUpdate"`
	AutoAnnounce   []AutoAnnounce `json:"autoAnnounce"`
	AutoBackup     AutoBackup     `json:"autoBackup"`
	Players        []Players      `json:"players"`
	Statistics     []Statistics   `json:"statistics"`
	Keepalive      Keepalive      `json:"keepalive"`
	AnnouncedID    int            `json:"announcedID"`
	SysSetting     SysSetting     `json:"sysSetting"`
	Bit64          bool           `json:"bit64"`
	Platform       string         `json:"platform"`
	TickRate       int            `json:"tickRate"`
	EncodeUserPath EncodeUserPath `json:"encodeUserPath"`
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

func GenerateJWT(username string, jwtSecret []byte, expiration int) (string, error) {
	// 定义一个自定义的声明结构

	claims := Claims{
		Username: username,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(expiration) * time.Hour).Unix(), // 过期时间
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

func ValidateJWT(tokenString string, jwtSecret []byte) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil {
		Logger.Warn("JWT验证失败")
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

func CreateConfig() {
	_ = EnsureDirExists(ConfDir)
	_, err := os.Stat(ConfDir + "/DstMP.sdb")
	if !os.IsNotExist(err) {
		Logger.Info("执行数据库检查中，发现数据库文件")
		config, err := ReadConfig()
		if err != nil {
			Logger.Error("执行数据库检查中，打开数据库文件失败", "err", err)
			return
		}
		if config.SysSetting.SchedulerSetting.PlayerGetFrequency == 0 {
			Logger.Info("设置玩家列表定时任务默认频率")
			config.SysSetting.SchedulerSetting.PlayerGetFrequency = 30
		}
		if config.SysSetting.SchedulerSetting.UIDMaintain.Frequency == 0 {
			Logger.Info("设置UID字典定时维护任务默认频率")
			config.SysSetting.SchedulerSetting.UIDMaintain.Frequency = 5
		}
		if config.Keepalive.Frequency == 0 {
			Logger.Info("设置自动保活任务默认频率")
			config.Keepalive.Frequency = 30
		}
		if config.TickRate == 0 {
			Logger.Info("设置默认TickRate")
			config.TickRate = 15
		}

		Logger.Info("执行数据库检查中，清除历史脏数据")
		config.Statistics = nil
		config.Players = nil
		err = WriteConfig(config)
		if err != nil {
			Logger.Error("写入数据库失败", "err", err)
		}
		Logger.Info("数据库检查完成")
		return
	}
	Logger.Info("执行数据库检查中，初始化数据库")
	var config Config
	config.Username = "admin"
	config.Password = "ba3253876aed6bc22d4a6ff53d8406c6ad864195ed144ab5c87621b6c233b548baeae6956df346ec8c17f5ea10f35ee3cbc514797ed7ddd3145464e2a0bab413"
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	charset := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length := 26
	randomString := make([]byte, length)
	for i := range randomString {
		// 从字符集中随机选择一个字符
		randomString[i] = charset[r.Intn(len(charset))]
	}
	config.JwtSecret = string(randomString)

	config.AutoUpdate.Time = "06:13:57"
	config.AutoUpdate.Enable = true

	config.AutoBackup.Time = "06:52:18"
	config.AutoBackup.Enable = true

	config.Keepalive.Enable = true
	config.Keepalive.Frequency = 30

	config.SysSetting.SchedulerSetting.PlayerGetFrequency = 30
	config.SysSetting.SchedulerSetting.UIDMaintain.Frequency = 5

	config.TickRate = 15

	err = WriteConfig(config)
	if err != nil {
		Logger.Error("写入数据库失败", "err", err)
		panic("数据库初始化失败")
	}
	Logger.Info("数据库初始化完成")
}

func ReadConfig() (Config, error) {
	content, err := os.ReadFile(ConfDir + "/DstMP.sdb")
	if err != nil {
		return Config{}, err
	}
	//jsonData := Base64Decode(string(content))
	jsonData := string(content)
	var config Config
	err = json.Unmarshal([]byte(jsonData), &config)
	if err != nil {
		return Config{}, fmt.Errorf("解析 JSON 失败: %w", err)
	}
	return config, nil
}

func WriteConfig(config Config) error {
	if config.Username == "" {
		return fmt.Errorf("传入的配置文件异常")
	}
	data, err := json.MarshalIndent(config, "", "    ") // 格式化输出
	if err != nil {
		return fmt.Errorf("Error marshalling JSON:" + err.Error())
	}
	file, err := os.OpenFile(ConfDir+"/DstMP.sdb", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
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

func ReadUidMap() (map[string]interface{}, error) {
	uidMap := make(map[string]interface{})
	content, err := os.ReadFile(NicknameUIDPath)
	if err != nil {
		return uidMap, err
	}
	jsonData := string(content)
	err = json.Unmarshal([]byte(jsonData), &uidMap)
	if err != nil {
		return uidMap, fmt.Errorf("解析 JSON 失败: %w", err)
	}
	return uidMap, nil
}

func WriteUidMap(uidMap map[string]interface{}) error {
	data, err := json.MarshalIndent(uidMap, "", "    ") // 格式化输出
	if err != nil {
		return fmt.Errorf("Error marshalling JSON:" + err.Error())
	}
	file, err := os.OpenFile(NicknameUIDPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
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
	var manualInstallScript string
	config, err := ReadConfig()
	if err != nil {
		Logger.Error("启动检查出现致命错误：获取数据库失败", "err", err)
		panic(err)
	}

	if config.Platform == "darwin" {
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
	err = EnsureDirExists(BackupPath)
	if err != nil {
		Logger.Error("创建备份目录失败", "err", err)
	} else {
		Logger.Info("备份目录检查完成")
	}

	err = EnsureDirExists(ModDownloadPath)
	if err != nil {
		Logger.Error("创建模组下载目录失败", "err", err)
	}
	err = EnsureDirExists(ModDownloadPath + "/not_ugc")
	if err != nil {
		Logger.Error("创建非UGC模组下载目录失败", "err", err)
	} else {
		err = EnsureDirExists(ModDownloadPath + "/steamapps/workshop/content/322330")
	}

	if err != nil {
		Logger.Error("创建UGC模组下载目录失败", "err", err)
	} else {
		Logger.Info("模组下载目录检查完成")
	}

	err = EnsureDirExists(ServerPath + MasterName)
	if err != nil {
		Logger.Error("创建Master目录失败", "err", err)
	} else {
		Logger.Info("Master目录检查完成")
	}

	err = EnsureDirExists(ServerPath + CavesName)
	if err != nil {
		Logger.Error("创建Caves目录失败", "err", err)
	} else {
		Logger.Info("Caves目录检查完成")
	}

	err = EnsureDirExists(MasterModUgcPath)
	if err != nil {
		Logger.Error("创建Master Mod目录失败", "err", err)
	} else {
		Logger.Info("Master Mod目录检查完成")
	}

	err = EnsureDirExists(CavesModUgcPath)
	if err != nil {
		Logger.Error("创建Caves Mod目录失败", "err", err)
	} else {
		Logger.Info("Caves Mod目录检查完成")
	}

}

func CheckFiles(checkItem string) {
	var (
		err   error
		exist bool
	)

	if checkItem == "uidMap" || checkItem == "all" {
		exist, err = FileDirectoryExists(NicknameUIDPath)
		if err != nil {
			Logger.Error("检查uid_map.json文件失败")
			return
		}

		if !exist {
			if err = EnsureFileExists(NicknameUIDPath); err != nil {
				Logger.Error("创建uid_map.json文件失败")
				return
			}

			if err = TruncAndWriteFile(NicknameUIDPath, "{}"); err != nil {
				Logger.Error("初始化uid_map.json文件失败")
				return
			}

			Logger.Info("uid_map.json文件检查完成")
		}

		if checkItem == "uidMap" {
			return
		}
	}

}

func CheckPlatform() {
	osInfo, err := GetOSInfo()
	if err != nil {
		Logger.Error("启动检查出现致命错误：获取系统信息失败", "err", err)
		panic(err)
	}

	config, err := ReadConfig()
	if err != nil {
		Logger.Error("启动检查出现致命错误：获取数据库失败", "err", err)
		panic(err)
	}
	config.Platform = osInfo.Platform

	PLATFORM = osInfo.Platform

	err = WriteConfig(config)
	if err != nil {
		Logger.Error("启动检查出现致命错误：写入数据库失败", "err", err)
		panic(err)
	}

	Logger.Info("系统检查通过")
}

func SetInitInfo() {
	config, err := ReadConfig()
	if err != nil {
		Logger.Error("读取配置文件失败", "err", err)
		return
	}

	if config.RoomSetting.Base.Name == "" {
		return
	}

	if config.RoomSetting.Ground != "" {
		cmd := "grep encode_user_path " + MasterServerPath + " | awk -F'=' '{print $2}'"
		out, _, err := BashCMDOutput(cmd)
		if err != nil {
			Logger.Warn("获取地面encode_user_path失败，跳过", "err", err)
			goto doCave
		}
		out = strings.TrimSpace(out)
		result, err := strconv.ParseBool(out)
		if err != nil {
			Logger.Warn("获取地面encode_user_path失败，跳过", "err", err)
			goto doCave
		}
		config.EncodeUserPath.Ground = result
		err = WriteConfig(config)
		if err != nil {
			Logger.Error("写入配置文件失败", "err", err)
		}
	}
doCave:
	if config.RoomSetting.Cave != "" {
		cmd := "grep encode_user_path " + CavesServerPath + " | awk -F'=' '{print $2}'"
		out, _, err := BashCMDOutput(cmd)
		if err != nil {
			Logger.Warn("获取洞穴encode_user_path失败，跳过", "err", err)
			return
		}
		out = strings.TrimSpace(out)
		result, err := strconv.ParseBool(out)
		if err != nil {
			Logger.Warn("获取洞穴encode_user_path失败，跳过", "err", err)
			return
		}
		config.EncodeUserPath.Cave = result
		err = WriteConfig(config)
		if err != nil {
			Logger.Error("写入配置文件失败", "err", err)
		}
	}
}

func BindFlags() {
	flag.IntVar(&BindPort, "l", 80, "监听端口，如： -l 8080 (Listening Port, e.g. -l 8080)")
	flag.StringVar(&ConfDir, "s", "./", "数据库文件目录，如： -s ./conf (Database Directory, e.g. -s ./conf)")
	flag.BoolVar(&ConsoleOutput, "c", false, "开启控制台日志输出，如： -c (Enable console log output, e.g. -c)")
	flag.BoolVar(&VersionShow, "v", false, "查看版本，如： -v (Check version, e.g. -v)")
	flag.Parse()
}

func MWlang() gin.HandlerFunc {
	return func(c *gin.Context) {
		lang := c.Request.Header.Get("X-I18n-Lang")
		c.Set("lang", lang)
	}
}

func MWtoken() gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.Request.Header.Get("authorization")
		config, err := ReadConfig()
		if err != nil {
			Logger.Error("配置文件打开失败", "err", err)
			return
		}
		tokenSecret := config.JwtSecret
		_, err = ValidateJWT(token, []byte(tokenSecret))
		if err != nil {
			lang := c.Request.Header.Get("X-I18n-Lang")
			RespondWithError(c, 420, lang)
			c.Abort()
			return
		}
		c.Next()
	}
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

func CpuUsage() (float64, error) {
	// 获取 CPU 使用率
	percent, err := cpu.Percent(0, false)
	if err != nil {
		return 0, fmt.Errorf("error getting CPU percent: %w", err)
	}
	return percent[0], nil
}

func MemoryUsage() (float64, error) {
	// 获取内存信息
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return 0, fmt.Errorf("error getting virtual memory info: %w", err)
	}
	return vmStat.UsedPercent, nil
}

func NetStatus() (float64, float64, error) {
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

func ScreenCMD(cmd string, world string) error {
	var totalCMD string
	if world == MasterName {
		totalCMD = "screen -S \"" + MasterScreenName + "\" -p 0 -X stuff \"" + cmd + "\\n\""
	}
	if world == CavesName {
		totalCMD = "screen -S \"" + CavesScreenName + "\" -p 0 -X stuff \"" + cmd + "\\n\""
	}

	cmdExec := exec.Command("/bin/bash", "-c", totalCMD)
	err := cmdExec.Run()
	if err != nil {
		return err
	}
	return nil
}

// ScreenCMDOutput 执行screen命令，并从日志中获取输出
// 自动添加print命令，cmdIdentifier是该命令在日志中输出的唯一标识符
func ScreenCMDOutput(cmd string, cmdIdentifier string, world string) (string, error) {
	var (
		totalCMD string
		logPath  string
	)

	if world == MasterName {
		totalCMD = "screen -S \"" + MasterScreenName + "\" -p 0 -X stuff \"print('" + cmdIdentifier + "' .. 'DMPSCREENCMD' .. tostring(" + cmd + "))\\n\""
		logPath = MasterLogPath
	}
	if world == CavesName {
		totalCMD = "screen -S \"" + CavesScreenName + "\" -p 0 -X stuff \"print('" + cmdIdentifier + "' .. 'DMPSCREENCMD' .. tostring(" + cmd + "))\\n\""
		logPath = CavesLogPath
	}

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
		homeDir, err := os.UserHomeDir()
		if err != nil {
			Logger.Error("无法获取 home 目录", "err", err)
			return err
		}
		dirPath = strings.Replace(dirPath, "~", homeDir, 1)
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

func FileDirectoryExists(filePath string) (bool, error) {
	// 如果路径中包含 ~，则将其替换为用户的 home 目录
	if strings.HasPrefix(filePath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			Logger.Error("无法获取 home 目录", "err", err)
			return false, err
		}
		filePath = strings.Replace(filePath, "~", homeDir, 1)
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

func BackupGame() error {
	err := EnsureDirExists(BackupPath)
	if err != nil {
		return err
	}
	currentTime := time.Now()
	timestampSeconds := currentTime.Unix()
	timestampSecondsStr := strconv.FormatInt(timestampSeconds, 10)
	cmd := "tar zcvf " + BackupPath + "/" + timestampSecondsStr + ".tgz " + ServerPath[:len(ServerPath)-1]
	err = BashCMD(cmd)
	if err != nil {
		return err
	}
	return nil
}

func StopGame() error {
	config, err := ReadConfig()
	if err != nil {
		Logger.Error("配置文件读取失败", "err", err)
		return err
	}

	cmd := "c_shutdown()"
	if config.RoomSetting.Ground != "" {
		err = ScreenCMD(cmd, MasterName)
		if err != nil {
			Logger.Info("执行ScreenCMD失败", "msg", err, "cmd", cmd)
		}
	}
	if config.RoomSetting.Cave != "" {
		err = ScreenCMD(cmd, CavesName)
		if err != nil {
			Logger.Info("执行ScreenCMD失败", "msg", err, "cmd", cmd)
		}
	}

	time.Sleep(2 * time.Second)
	if config.RoomSetting.Ground != "" {
		err = BashCMD(StopMasterCMD)
		if err != nil {
			Logger.Info("执行BashCMD失败", "msg", err, "cmd", StopMasterCMD)
		}
	}
	if config.RoomSetting.Cave != "" {
		err = BashCMD(StopCavesCMD)
		if err != nil {
			Logger.Info("执行BashCMD失败", "msg", err, "cmd", StopCavesCMD)
		}
	}

	time.Sleep(1 * time.Second)

	err = BashCMD(KillDST)
	if err != nil {
		Logger.Info("执行BashCMD失败", "msg", err, "cmd", KillDST)
	}
	err = BashCMD(ClearScreenCMD)
	if err != nil {
		Logger.Info("执行BashCMD失败", "msg", err, "cmd", ClearScreenCMD)
	}

	return nil
}

func StartGame() error {
	config, err := ReadConfig()
	if err != nil {
		Logger.Error("配置文件读取失败", "err", err)
		return err
	}

	if config.Platform == "darwin" {
		if config.RoomSetting.Ground != "" {
			err = BashCMD(MacStartMasterCMD)
			if err != nil {
				Logger.Error("执行BashCMD失败", "err", err, "cmd", MacStartMasterCMD)
			}
		}
		if config.RoomSetting.Cave != "" {
			err = BashCMD(MacStartCavesCMD)
			if err != nil {
				Logger.Error("执行BashCMD失败", "err", err, "cmd", MacStartCavesCMD)
			}
		}
	} else {
		_ = ReplaceDSTSOFile()
		if config.RoomSetting.Ground != "" {
			var cmd string
			if config.Bit64 {
				cmd = StartMaster64CMD
			} else {
				cmd = StartMasterCMD
			}
			err = BashCMD(cmd)
			if err != nil {
				Logger.Error("执行BashCMD失败", "err", err, "cmd", cmd)
			}
		}
		if config.RoomSetting.Cave != "" {
			var cmd string
			if config.Bit64 {
				cmd = StartCaves64CMD
			} else {
				cmd = StartCavesCMD
			}
			err = BashCMD(cmd)
			if err != nil {
				Logger.Error("执行BashCMD失败", "err", err, "cmd", cmd)
			}
		}
	}
	return nil
}

func RecoveryGame(backupFile string) error {
	// 检查文件是否存在
	exist, err := FileDirectoryExists(backupFile)
	if !exist || err != nil {
		return fmt.Errorf("文件不存在，%w", err)
	}
	// 停止进程
	cmd := "c_shutdown()"
	err = ScreenCMD(cmd, MasterName)
	if err != nil {
		Logger.Warn("ScreenCMD执行失败", "err", err, "cmd", cmd, "world", MasterName)
	}

	err = ScreenCMD(cmd, CavesName)
	if err != nil {
		Logger.Warn("ScreenCMD执行失败", "err", err, "cmd", cmd, "world", CavesName)
	}

	time.Sleep(2 * time.Second)

	err = BashCMD(StopMasterCMD)
	if err != nil {
		Logger.Error("BashCMD执行失败", "err", err, "cmd", StopMasterCMD)
	}

	err = BashCMD(StopCavesCMD)
	if err != nil {
		Logger.Error("BashCMD执行失败", "err", err, "cmd", StopCavesCMD)
	}

	err = BashCMD(ClearScreenCMD)
	if err != nil {
		Logger.Error("BashCMD执行失败", "err", err, "cmd", ClearScreenCMD)
	}

	// 删除主目录
	err = RemoveDir(ServerPath)
	if err != nil {
		Logger.Error("删除主目录失败", "err", err)
		return err
	}

	// 解压备份文件
	cmd = "tar zxvf " + backupFile
	err = BashCMD(cmd)
	if err != nil {
		Logger.Error("BashCMD执行失败", "err", err, "cmd", cmd)
		return err
	}

	return nil
}

func GetTimestamp() int64 {
	now := time.Now()
	// 获取毫秒级时间戳
	milliseconds := now.UnixNano() / int64(time.Millisecond)
	return milliseconds
}

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
func GetDirs(dirPath string) ([]string, error) {
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
			dirs = append(dirs, entry.Name())
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

func GetRoomSettingBase() (RoomSettingBase, error) {
	roomSettings := RoomSettingBase{}
	// 打开文件
	file, err := os.Open(ServerSettingPath)
	if err != nil {
		Logger.Error("打开cluster.ini文件失败", "err", err)
		return RoomSettingBase{}, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			Logger.Error("关闭cluster.ini文件失败", "err", err)
		}
	}(file)

	// 定义要读取的字段映射
	fieldsToRead := map[string]string{
		"cluster_name":        "Name",
		"cluster_description": "Description",
		"game_mode":           "GameMode",
		"pvp":                 "PVP",
		"max_players":         "PlayerNum",
		"vote_enabled":        "Vote",
		"cluster_password":    "Password",
	}

	// 使用bufio.Scanner逐行读取文件内容
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		// 跳过注释和空行
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") || line == "" {
			continue
		}
		// 解析字段和值
		for field, structField := range fieldsToRead {
			if strings.HasPrefix(line, field+" =") {
				value := strings.TrimPrefix(line, field+" =")
				value = strings.TrimSpace(value)

				// 根据结构体字段类型设置值
				switch structField {
				case "Name":
					roomSettings.Name = value
				case "Description":
					roomSettings.Description = value
				case "GameMode":
					roomSettings.GameMode = value
				case "PVP":
					roomSettings.PVP, _ = strconv.ParseBool(value)
				case "PlayerNum":
					roomSettings.PlayerNum, _ = strconv.Atoi(value)
				case "Vote":
					roomSettings.Vote, _ = strconv.ParseBool(value)
				case "Password":
					roomSettings.Password = value
				}
				break
			}
		}
	}

	// 检查是否有错误
	if err := scanner.Err(); err != nil {
		Logger.Error("读取cluster.ini文件失败", "err", err)
		return RoomSettingBase{}, err
	}

	//token文件
	token, err := GetFileAllContent(ServerTokenPath)
	if err != nil {
		Logger.Error("读取token文件失败", "err", err)
		return RoomSettingBase{}, err
	}
	roomSettings.Token = token

	return roomSettings, nil
}

func GetServerPort(serverFile string) (int, error) {
	file, err := os.Open(serverFile)
	if err != nil {
		Logger.Error("打开"+serverFile+"文件失败", "err", err)
		return 0, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			Logger.Error("关闭"+serverFile+"文件失败", "err", err)
		}
	}(file)
	// 使用bufio.Scanner逐行读取文件内容
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)
		// 跳过注释和空行
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") || line == "" {
			continue
		}
		// 解析字段和值
		if strings.HasPrefix(line, "server_port =") {
			value := strings.TrimPrefix(line, "server_port =")
			value = strings.TrimSpace(value)
			port, err := strconv.Atoi(value)
			if err != nil {
				Logger.Error("获取端口失败，端口必须为数字", "err", err)
				return 0, err
			}
			return port, nil
		}
	}
	return 0, fmt.Errorf("没有找到端口配置")
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
