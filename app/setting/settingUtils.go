package setting

import (
	"bufio"
	"dst-management-platform-api/utils"
	lua "github.com/yuin/gopher-lua"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
)

func clusterTemplate(config utils.Config) string {
	var bindIP string
	if !config.MultiHost {
		bindIP = "127.0.0.1"
	} else {
		bindIP = "0.0.0.0"
	}
	base := config.RoomSetting.Base
	contents := `
[GAMEPLAY]
game_mode = ` + base.GameMode + `
max_players = ` + strconv.Itoa(base.PlayerNum) + `
pvp = ` + strconv.FormatBool(base.PVP) + `
pause_when_empty = true
vote_enabled = ` + strconv.FormatBool(base.Vote) + `
vote_kick_enabled = ` + strconv.FormatBool(base.Vote) + `

[NETWORK]
cluster_description = ` + base.Description + `
whitelist_slots = 0
cluster_name = ` + base.Name + `
cluster_password = ` + base.Password + `
cluster_language = zh
tick_rate = ` + strconv.Itoa(config.TickRate) + `

[MISC]
console_enabled = true
max_snapshots = ` + strconv.Itoa(base.BackDays) + `

[SHARD]
shard_enabled = true
bind_ip = ` + bindIP + `
master_ip = ` + base.ShardMasterIp + `
master_port = ` + strconv.Itoa(base.ShardMasterPort) + `
cluster_key = ` + base.ClusterKey + `
`
	return contents
}

func masterServerTemplate(config utils.Config) string {
	base := config.RoomSetting.Base
	content := `
[NETWORK]
server_port = ` + strconv.Itoa(base.MasterPort) + `

[SHARD]
is_master = true

[STEAM]
master_server_port = ` + strconv.Itoa(base.SteamMasterPort) + `
authentication_port = ` + strconv.Itoa(base.SteamAuthenticationPort) + `

[ACCOUNT]
encode_user_path = ` + strconv.FormatBool(config.EncodeUserPath.Ground) + `
`
	return content
}

func cavesServerTemplate(config utils.Config) string {
	var (
		SteamMasterPort         int
		SteamAuthenticationPort int
	)
	if !config.MultiHost {
		SteamMasterPort = config.RoomSetting.Base.SteamMasterPort + 1
		SteamAuthenticationPort = config.RoomSetting.Base.SteamAuthenticationPort + 1
	} else {
		SteamMasterPort = config.RoomSetting.Base.SteamMasterPort
		SteamAuthenticationPort = config.RoomSetting.Base.SteamAuthenticationPort
	}
	base := config.RoomSetting.Base
	content := `
[NETWORK]
server_port = ` + strconv.Itoa(base.CavesPort) + `

[SHARD]
is_master = false
name = Caves

[STEAM]
master_server_port = ` + strconv.Itoa(SteamMasterPort) + `
authentication_port = ` + strconv.Itoa(SteamAuthenticationPort) + `

[ACCOUNT]
encode_user_path = ` + strconv.FormatBool(config.EncodeUserPath.Cave) + `
`
	return content
}

func saveSetting(config utils.Config) error {
	clusterIniFileContent := clusterTemplate(config)

	//cluster.ini
	err := utils.TruncAndWriteFile(utils.ServerSettingPath, clusterIniFileContent)
	if err != nil {
		return err
	}

	//cluster_token.txt
	err = utils.TruncAndWriteFile(utils.ServerTokenPath, config.RoomSetting.Base.Token)
	if err != nil {
		return err
	}

	if config.RoomSetting.Ground != "" {
		err = utils.EnsureDirExists(utils.ServerPath + utils.MasterName)
		if err != nil {
			utils.Logger.Error("创建Master目录失败", "err", err)
		}
		//Master/leveldataoverride.lua
		err = utils.TruncAndWriteFile(utils.MasterSettingPath, config.RoomSetting.Ground)
		if err != nil {
			return err
		}

		//Master/modoverrides.lua
		err = utils.TruncAndWriteFile(utils.MasterModPath, config.RoomSetting.Mod)
		if err != nil {
			return err
		}

		//Master/server.ini
		err = utils.TruncAndWriteFile(utils.MasterServerPath, masterServerTemplate(config))
		if err != nil {
			return err
		}
	} else {
		err = utils.RemoveDir(utils.ServerPath + utils.MasterName)
		if err != nil {
			utils.Logger.Error("删除目录下文件失败", "err", err)
		}
	}

	if config.RoomSetting.Cave != "" {
		//Caves/leveldataoverride.lua
		err = utils.EnsureDirExists(utils.ServerPath + utils.CavesName)
		if err != nil {
			utils.Logger.Error("创建Caves目录失败", "err", err)
		}
		err = utils.TruncAndWriteFile(utils.CavesSettingPath, config.RoomSetting.Cave)
		if err != nil {
			return err
		}
		//Caves/modoverrides.lua
		err = utils.TruncAndWriteFile(utils.CavesModPath, config.RoomSetting.Mod)
		if err != nil {
			return err
		}
		//Caves/server.ini
		err = utils.TruncAndWriteFile(utils.CavesServerPath, cavesServerTemplate(config))
		if err != nil {
			return err
		}
	} else {
		err = utils.RemoveDir(utils.ServerPath + utils.CavesName)
		if err != nil {
			utils.Logger.Error("删除目录下文件失败", "err", err)
		}
	}

	return nil
}

func generateWorld() {
	//关闭Master进程
	/*cmdStopMaster := exec.Command("/bin/bash", "-c", utils.StopMasterCMD)
	err := cmdStopMaster.Run()
	if err != nil {
		utils.Logger.Error("关闭地面失败", "err", err)
	}
	//关闭Caves进程
	cmdStopCaves := exec.Command("/bin/bash", "-c", utils.StopCavesCMD)
	err = cmdStopCaves.Run()
	if err != nil {
		utils.Logger.Error("关闭洞穴失败", "err", err)
	}*/
	err := utils.StopGame()
	if err != nil {
		utils.Logger.Error("关闭游戏失败", "err", err)
	}
	//删除Master/save目录
	err = utils.DeleteDir(utils.MasterSavePath)
	if err != nil {
		utils.Logger.Error("删除地面文件失败", "err", err, "dir", utils.MasterSavePath)
	}
	//等待3秒
	time.Sleep(3 * time.Second)
	//启动Master
	/*cmdStartMaster := exec.Command("/bin/bash", "-c", utils.StartMasterCMD)
	err = cmdStartMaster.Run()
	if err != nil {
		utils.Logger.Error("启动地面失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	if config.RoomSetting.Cave != "" {
		//删除Caves/save目录
		err = utils.DeleteDir(utils.CavesSavePath)
		if err != nil {
			utils.Logger.Error("删除洞穴文件失败", "err", err, "dir", utils.CavesSavePath)
		}
		//启动Caves
		cmdStartCaves := exec.Command("/bin/bash", "-c", utils.StartCavesCMD)
		err = cmdStartCaves.Run()
		if err != nil {
			utils.Logger.Error("启动洞穴失败", "err", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}
	}*/
	err = utils.StartGame()
	if err != nil {
		utils.Logger.Error("启动游戏失败", "err", err)
	}
}

func DstModsSetup() error {
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		return err
	}

	L := lua.NewState()
	defer L.Close()
	if err := L.DoString(config.RoomSetting.Mod); err != nil {
		utils.Logger.Error("加载 Lua 文件失败:", "err", err)
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
		if config.Platform == "darwin" {
			modFilePath = utils.MacGameModSettingPath
		} else {
			modFilePath = utils.GameModSettingPath
		}
		err := utils.TruncAndWriteFile(modFilePath, fileContent)
		if err != nil {
			utils.Logger.Error("mod配置文件写入失败", "err", err, "file", modFilePath)
			return err
		}
	}

	return nil
}

func getList(filepath string) []string {
	// 预留位 黑名单 管理员
	al, err := readLines(filepath)
	if err != nil {
		utils.Logger.Error("读取文件失败", "err", err, "file", filepath)
		return []string{}
	}
	var uidList []string
	for _, a := range al {
		uid := strings.TrimSpace(a)
		uidList = append(uidList, uid)
	}
	if uidList == nil {
		return []string{}
	}
	return uidList
}

func addList(uid string, filePath string) error {
	// 要追加的内容
	content := "\n" + uid
	// 打开文件，使用 os.O_APPEND | os.O_CREATE | os.O_WRONLY 选项
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err = file.Close()
		if err != nil {
			utils.Logger.Error("关闭文件失败", "err", err)
		}
	}(file) // 确保在函数结束时关闭文件
	// 写入内容到文件
	if _, err = file.WriteString(content); err != nil {
		return err
	}

	return nil
}

func deleteList(uid string, filePath string) error {
	// 读取文件内容
	lines, err := readLines(filePath)
	if err != nil {
		return err
	}

	// 删除指定行
	for i := 0; i < len(lines); i++ {
		if lines[i] == uid {
			lines = append(lines[:i], lines[i+1:]...)
			break
		}
	}

	// 将修改后的内容写回文件
	err = writeLines(filePath, lines)
	if err != nil {
		return err
	}

	return nil
}

// 读取文件内容到切片中
func readLines(filePath string) ([]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			utils.Logger.Error("关闭文件失败", "err", err)
		}
	}(file)

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// 将切片内容写回文件
func writeLines(filePath string, lines []string) error {
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			utils.Logger.Error("关闭文件失败", "err", err)
		}
	}(file)

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, _ = writer.WriteString(line + "\n")
	}
	return writer.Flush()
}

type UIDForm struct {
	UID string `json:"uid"`
}

func kick(uid string, world string) error {
	cmd := "TheNet:Kick('" + uid + "')"
	return utils.ScreenCMD(cmd, world)
}

func checkZipFile(filename string) (bool, error) {
	filePath := utils.ImportFileUploadPath + filename
	err := utils.EnsureDirExists(utils.ImportFileUnzipPath)
	if err != nil {
		utils.Logger.Error("解压目录创建失败", "err", err)
		return false, err
	}
	err = utils.BashCMD("unzip -qo " + filePath + " -d " + utils.ImportFileUnzipPath)
	if err != nil {
		utils.Logger.Error("解压失败", "err", err)
		return false, err
	}

	var result bool
	checkItems := []string{"cluster.ini", "cluster_token.txt", "Master/leveldataoverride.lua", "Master/modoverrides.lua", "Master/server.ini"}
	for _, item := range checkItems {
		filePath = utils.ImportFileUnzipPath + item
		result, err = utils.FileDirectoryExists(filePath)
		if err != nil {
			utils.Logger.Error("检查文件"+filePath+"失败", "err", err)
			return false, err
		}
		if !result {
			utils.Logger.Error("文件" + filePath + "不存在")
			return false, nil
		}
	}
	return true, nil
}

func WriteDatabase() error {
	//地面配置
	ground, err := utils.GetFileAllContent(utils.MasterSettingPath)
	if err != nil {
		utils.Logger.Error("读取地面配置文件失败", "err", err)
		return err
	}
	//模组配置
	mod, err := utils.GetFileAllContent(utils.MasterModPath)
	if err != nil {
		utils.Logger.Error("读取mod配置文件失败", "err", err)
		return err
	}
	//洞穴配置
	caves, err := utils.GetFileAllContent(utils.CavesSettingPath)
	if err != nil {
		utils.Logger.Warn("读取洞穴配置文件失败", "err", err)
		caves = ""
	}

	var baseSetting utils.RoomSettingBase
	baseSetting, err = utils.GetRoomSettingBase()
	if err != nil {
		utils.Logger.Error("读取cluster配置文件失败", "err", err)
		return err
	}

	masterPort, err := utils.GetServerPort(utils.MasterServerPath)
	if err != nil {
		utils.Logger.Error("获取地面端口失败", "err", err)
		return err
	}
	baseSetting.MasterPort = masterPort
	if caves != "" {
		cavesPort, err := utils.GetServerPort(utils.CavesServerPath)
		if err != nil {
			utils.Logger.Error("获取洞穴端口失败", "err", err)
			return err
		}
		baseSetting.CavesPort = cavesPort
	}

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		return err
	}

	utils.SetInitInfo()

	config.RoomSetting.Base = baseSetting
	config.RoomSetting.Ground = ground
	config.RoomSetting.Cave = caves
	config.RoomSetting.Mod = mod

	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		return err
	}

	return nil
}

func clearUpZipFile() {
	err := utils.BashCMD("rm -rf " + utils.ImportFileUploadPath + "*")
	if err != nil {
		utils.Logger.Error("清理导入的压缩文件失败", "err", err)
	}
}

func changeWhitelistSlots() error {
	err := utils.EnsureFileExists(utils.WhiteListPath)
	if err != nil {
		utils.Logger.Error("打开白名单失败")
		return err
	}

	fileContent, err := readLines(utils.WhiteListPath)
	if err != nil {
		utils.Logger.Error("读取白名单失败")
		return err
	}

	var whiteList []string
	for _, i := range fileContent {
		uid := strings.TrimSpace(i)
		if uid != "" {
			whiteList = append(whiteList, uid)
		}
	}

	clusterIniContent, err := readLines(utils.ServerSettingPath)
	if err != nil {
		utils.Logger.Error("读取cluster.ini失败")
		return err
	}

	var newClusterIni []string

	for _, i := range clusterIniContent {
		line := strings.TrimSpace(i)
		if strings.HasPrefix(line, "cluster_name") {
			newClusterIni = append(newClusterIni, "whitelist_slots = "+strconv.Itoa(len(whiteList)))
		}
		if strings.HasPrefix(line, "whitelist_slots") {
			continue
		}
		newClusterIni = append(newClusterIni, line)
	}

	err = writeLines(utils.ServerSettingPath, newClusterIni)
	if err != nil {
		utils.Logger.Error("写入cluster.ini失败")
		return err
	}

	return nil
}

type SystemSettingForm struct {
	KeepaliveDisable   bool                       `json:"keepaliveDisable"`
	KeepaliveFrequency int                        `json:"keepaliveFrequency"`
	PlayerGetFrequency int                        `json:"playerGetFrequency"`
	UIDMaintain        utils.SchedulerSettingItem `json:"UIDMaintain"`
	SysMetricsGet      utils.SchedulerSettingItem `json:"sysMetricsGet"`
	Bit64              bool                       `json:"bit64"`
	TickRate           int                        `json:"tickRate"`
	EncodeUserPath     utils.EncodeUserPath       `json:"encodeUserPath"`
}

func GetUserDataEncodeStatus(uid string, world string) (bool, error) {
	userPathEncode, err := utils.ScreenCMDOutput(utils.UserDataEncode, uid+"UserDataEncode", world)
	if err != nil {
		return false, err
	}
	if userPathEncode == "true" {
		return true, nil
	} else {
		return false, nil
	}
}

func GetPlayerAgePrefab(uid string, world string, userPathEncode bool) (int, string, error) {
	var (
		path      string
		cmdAge    string
		cmdPrefab string
	)

	if userPathEncode {
		sessionFileCmd := "TheNet:GetUserSessionFile(ShardGameIndex:GetSession(), '" + uid + "')"
		userSessionFile, err := utils.ScreenCMDOutput(sessionFileCmd, uid+"UserSessionFile", world)
		if err != nil {
			return 0, "", err
		}

		if world == "Master" {
			path = utils.MasterSavePath + "/" + userSessionFile
		} else {
			path = utils.CavesSavePath + "/" + userSessionFile
		}

		ok, _ := utils.FileDirectoryExists(path)
		if !ok {
			return 0, "", err
		}

	} else {
		cmd := "find " + utils.ServerPath + world + "/save/session/*/" + uid + "_/ -name \"*.meta\" -type f -printf \"%T@ %p\\n\" | sort -n | tail -n 1 | cut -d' ' -f2"
		stdout, _, err := utils.BashCMDOutput(cmd)
		if err != nil || stdout == "" {
			utils.Logger.Warn("Bash命令执行失败", "err", err, "cmd", cmd)
			return 0, "", err
		}
		path = stdout[:len(stdout)-6]
	}

	if utils.PLATFORM == "darwin" {
		cmdAge = "ggrep -aoP 'age=\\d+\\.\\d+' " + path + " | awk -F'=' '{print $2}'"
	} else {
		cmdAge = "grep -aoP 'age=\\d+\\.\\d+' " + path + " | awk -F'=' '{print $2}'"
	}

	stdout, _, err := utils.BashCMDOutput(cmdAge)
	if err != nil || stdout == "" {
		utils.Logger.Error("Bash命令执行失败", "err", err, "cmd", cmdAge)
		return 0, "", err
	}

	stdout = strings.TrimSpace(stdout)
	age, err := strconv.ParseFloat(stdout, 64)
	if err != nil {
		utils.Logger.Error("玩家游戏时长转换失败", "err", err)
		age = 0
	}
	age = age / 480
	ageInt := int(math.Round(age))

	if utils.PLATFORM == "darwin" {
		cmdPrefab = "ggrep -aoP '},age=\\d+,prefab=\"(.+)\"}' " + path + " | awk -F'[\"]' '{print $2}'"
	} else {
		cmdPrefab = "grep -aoP '},age=\\d+,prefab=\"(.+)\"}' " + path + " | awk -F'[\"]' '{print $2}'"
	}

	stdout, _, err = utils.BashCMDOutput(cmdPrefab)
	if err != nil || stdout == "" {
		utils.Logger.Error("Bash命令执行失败", "err", err, "cmd", cmdPrefab)
		return ageInt, "", nil
	}
	prefab := strings.TrimSpace(stdout)

	return ageInt, prefab, nil
}
