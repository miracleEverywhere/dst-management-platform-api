package utils

import (
	"bufio"
	"encoding/json"
	"fmt"
	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

var STATISTICS []Statistics

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

type RoomSettingBase struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	GameMode    string `json:"gameMode"`
	PVP         bool   `json:"pvp"`
	PlayerNum   int    `json:"playerNum"`
	BackDays    int    `json:"backDays"`
	Vote        bool   `json:"vote"`
	Password    string `json:"password"`
	Token       string `json:"token"`
	MasterPort  int    `json:"masterPort"`
	CavesPort   int    `json:"cavesPort"`
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
}

type Statistics struct {
	Timestamp int64     `json:"timestamp"`
	Num       int       `json:"num"`
	Players   []Players `json:"players"`
}

type Keepalive struct {
	Enable        bool   `json:"enable"`
	Frequency     int    `json:"frequency"`
	LastTime      string `json:"lastTime"`
	CavesLastTime string `json:"cavesLastTime"`
}

type Config struct {
	Username     string         `json:"username"`
	Nickname     string         `json:"nickname"`
	Password     string         `json:"password"`
	JwtSecret    string         `json:"jwtSecret"`
	RoomSetting  RoomSetting    `json:"roomSetting"`
	AutoUpdate   AutoUpdate     `json:"autoUpdate"`
	AutoAnnounce []AutoAnnounce `json:"autoAnnounce"`
	AutoBackup   AutoBackup     `json:"autoBackup"`
	Players      []Players      `json:"players"`
	Statistics   []Statistics   `json:"statistics"`
	Keepalive    Keepalive      `json:"keepalive"`
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
	_, err := os.Stat("DstMP.sdb")
	if !os.IsNotExist(err) {
		Logger.Info("执行数据库检查中，发现数据库文件")
		config, err := ReadConfig()
		if err != nil {
			Logger.Error("执行数据库检查中，打开数据库文件失败", "err", err)
			return
		}
		if config.Keepalive.Frequency == 30 && config.Statistics == nil && config.Players == nil {
			Logger.Info("数据库检查完成")
			return
		}
		Logger.Info("执行数据库检查中，自动保活设置为30秒")
		config.Keepalive.Frequency = 30
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

	err = WriteConfig(config)
	if err != nil {
		Logger.Error("写入数据库失败", "err", err)
		panic("数据库初始化失败")
	}
	Logger.Info("数据库初始化完成")
}

func ReadConfig() (Config, error) {
	content, err := os.ReadFile("DstMP.sdb")
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
	file, err := os.OpenFile("DstMP.sdb", os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0666)
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

func BashCMD(cmd string) error {
	cmdExec := exec.Command("/bin/bash", "-c", cmd)
	err := cmdExec.Run()
	if err != nil {
		return err
	}
	return nil
}

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

func FileDirectoryExists(filePath string) (bool, error) {
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

//func GetModList() ([]string, error) {
//	var modList []string
//	L := lua.NewState()
//	defer L.Close()
//	if err := L.DoFile(MasterModPath); err != nil {
//		return []string{}, fmt.Errorf("加载 Lua 文件失败: %w", err)
//	}
//	modsTable := L.Get(-1)
//	if tbl, ok := modsTable.(*lua.LTable); ok {
//		tbl.ForEach(func(key lua.LValue, value lua.LValue) {
//			// 检查键是否是字符串，并且以 "workshop-" 开头
//			if strKey, ok := key.(lua.LString); ok && strings.HasPrefix(string(strKey), "workshop-") {
//				// 提取 "workshop-" 后面的数字
//				workshopID := strings.TrimPrefix(string(strKey), "workshop-")
//				modList = append(modList, workshopID)
//			}
//		})
//	}
//	return modList, nil
//}
//
//func DownloadMod(modList []string) error {
//	if len(modList) == 0 {
//		return nil
//	}
//	err := TruncAndWriteFile(GameModSettingPath, "")
//	if err != nil {
//		return err
//	}
//
//	downloadCMD := "steamcmd/steamcmd.sh +force_install_dir dl +login anonymous"
//	for _, mod := range modList {
//		downloadCMD = downloadCMD + " +workshop_download_item 322330 " + mod
//	}
//	downloadCMD = downloadCMD + " +quit"
//	err = BashCMD(downloadCMD)
//	if err != nil {
//		return err
//	}
//
//	for _, mod := range modList {
//		mvCMD := "mv ~/steamcmd/dl/steamapps/workshop/content/322330/" + mod + " ~/dst/mods/workshop-" + mod
//		err = BashCMD(mvCMD)
//		if err != nil {
//			return err
//		}
//	}
//
//	rmCMD := "rm -rf ~/dl"
//	err = BashCMD(rmCMD)
//	if err != nil {
//		return err
//	}
//
//	return nil
//}

func GetTimestamp() int64 {
	now := time.Now()
	// 获取毫秒级时间戳
	milliseconds := now.UnixNano() / int64(time.Millisecond)
	return milliseconds
}

func GetFileAllContent(filePath string) (string, error) {
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
