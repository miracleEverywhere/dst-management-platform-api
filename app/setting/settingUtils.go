package setting

import (
	"bufio"
	"dst-management-platform-api/utils"
	"fmt"
	"github.com/gin-gonic/gin"
	lua "github.com/yuin/gopher-lua"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func clusterTemplate(base utils.RoomSettingBase) string {
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
cluster_name = ` + base.Name + `
cluster_password = ` + base.Password + `
cluster_language = zh

[MISC]
console_enabled = true
max_snapshots = ` + strconv.Itoa(base.BackDays) + `

[SHARD]
shard_enabled = true
bind_ip = 127.0.0.1
master_ip = 127.0.0.1
master_port = 10889
cluster_key = supersecretkey
`
	return contents
}

func masterServerTemplate() string {
	content := `
[NETWORK]
server_port = 11000

[SHARD]
is_master = true

[STEAM]
master_server_port = 27018
authentication_port = 8768
`
	return content
}

func cavesServerTemplate() string {
	content := `
[NETWORK]
server_port = 11001

[SHARD]
is_master = false
name = Caves

[STEAM]
master_server_port = 27019
authentication_port = 8769
`
	return content
}

func saveSetting(config utils.Config) {
	clusterIniFileContent := clusterTemplate(config.RoomSetting.Base)

	//cluster.ini
	utils.TruncAndWriteFile(utils.ServerSettingPath, clusterIniFileContent)

	//cluster_token.txt
	utils.TruncAndWriteFile(utils.ServerTokenPath, config.RoomSetting.Base.Token)

	//Master/leveldataoverride.lua
	utils.TruncAndWriteFile(utils.MasterSettingPath, config.RoomSetting.Ground)

	//Master/modoverrides.lua
	utils.TruncAndWriteFile(utils.MasterModPath, config.RoomSetting.Mod)

	//Master/server.ini
	utils.TruncAndWriteFile(utils.MasterServerPath, masterServerTemplate())

	if config.RoomSetting.Cave != "" {
		//Caves/leveldataoverride.lua
		utils.TruncAndWriteFile(utils.CavesSettingPath, config.RoomSetting.Cave)
		//Caves/modoverrides.lua
		utils.TruncAndWriteFile(utils.CavesModPath, config.RoomSetting.Mod)
		//Caves/server.ini
		utils.TruncAndWriteFile(utils.CavesServerPath, cavesServerTemplate())
	}
}

func restartWorld(c *gin.Context, config utils.Config, langStr string) {
	//关闭Master进程
	_ = utils.BashCMD(utils.StopMasterCMD)
	//关闭Caves进程
	_ = utils.BashCMD(utils.StopCavesCMD)
	//等待3秒
	time.Sleep(3 * time.Second)
	//启动Master
	cmdStartMaster := exec.Command("/bin/bash", "-c", utils.StartMasterCMD)
	err := cmdStartMaster.Run()
	if err != nil {
		fmt.Println("Error Start Master:", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	if config.RoomSetting.Cave != "" {
		//启动Caves
		cmdStartCaves := exec.Command("/bin/bash", "-c", utils.StartCavesCMD)
		err = cmdStartCaves.Run()
		if err != nil {
			fmt.Println("Error Start Caves:", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}
	}
}

func generateWorld(c *gin.Context, config utils.Config, langStr string) {
	//关闭Master进程
	cmdStopMaster := exec.Command("/bin/bash", "-c", utils.StopMasterCMD)
	err := cmdStopMaster.Run()
	if err != nil {
		fmt.Println("Error Stop Master:", err)
	}
	//关闭Caves进程
	cmdStopCaves := exec.Command("/bin/bash", "-c", utils.StopCavesCMD)
	err = cmdStopCaves.Run()
	if err != nil {
		fmt.Println("Error Stop Caves:", err)
	}
	//删除Master/save目录
	utils.DeleteDir(utils.MasterSavePath)
	//等待3秒
	time.Sleep(3 * time.Second)
	//启动Master
	cmdStartMaster := exec.Command("/bin/bash", "-c", utils.StartMasterCMD)
	err = cmdStartMaster.Run()
	if err != nil {
		fmt.Println("Error Start Master:", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	if config.RoomSetting.Cave != "" {
		//删除Caves/save目录
		utils.DeleteDir(utils.CavesSavePath)
		//启动Caves
		cmdStartCaves := exec.Command("/bin/bash", "-c", utils.StartCavesCMD)
		err = cmdStartCaves.Run()
		if err != nil {
			fmt.Println("Error Start Caves:", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}
	}
}

func dstModsSetup() {
	L := lua.NewState()
	defer L.Close()
	if err := L.DoFile(utils.MasterModPath); err != nil {
		fmt.Println("加载 Lua 文件失败:", err)
		return
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
		utils.TruncAndWriteFile(utils.GameModSettingPath, fileContent)
	}
}

func getList(filepath string) []string {
	// 预留位 黑名单 管理员
	al, err := readLines(filepath)
	if err != nil {
		return []string{}
	}
	var adminList []string
	for _, a := range al {
		admin := strings.TrimSpace(a)
		adminList = append(adminList, admin)
	}
	if adminList == nil {
		return []string{}
	}
	return adminList
}

func addList(uid string, filePath string) error {
	// 要追加的内容
	content := "\n" + uid
	// 打开文件，使用 os.O_APPEND | os.O_CREATE | os.O_WRONLY 选项
	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Println("打开文件错误:", err)
		return err
	}
	defer file.Close() // 确保在函数结束时关闭文件
	// 写入内容到文件
	if _, err := file.WriteString(content); err != nil {
		fmt.Println("写入文件错误:", err)
		return err
	}

	return nil
}

func deleteList(uid string, filePath string) error {
	// 读取文件内容
	lines, err := readLines(filePath)
	if err != nil {
		fmt.Println("读取文件错误:", err)
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
		fmt.Println("写入文件错误:", err)
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
	defer file.Close()

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
	defer file.Close()

	writer := bufio.NewWriter(file)
	for _, line := range lines {
		_, _ = writer.WriteString(line + "\n")
	}
	return writer.Flush()
}

type UIDForm struct {
	UID string `json:"uid"`
}
