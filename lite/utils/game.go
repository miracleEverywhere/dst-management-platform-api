package utils

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	gameModCfgPath = "dst/mods/dedicated_server_mods_setup.lua"
	clusterPath    = mustResolveClusterPath()
)

func SaveRoomFile(room Room) error {
	if err := ensureDirExists(clusterPath); err != nil {
		return fmt.Errorf("无法创建目录：%v", err)
	}

	clusterIniPath := filepath.Join(clusterPath, "cluster.ini")
	if err := checkAndWriteFile(clusterIniPath, getClusterIni(room)); err != nil {
		return err
	}

	whitelistPath := filepath.Join(clusterPath, "whitelist.txt")
	if err := checkAndWriteFile(whitelistPath, room.Whitelist); err != nil {
		return err
	}

	adminlistPath := filepath.Join(clusterPath, "adminlist.txt")
	if err := checkAndWriteFile(adminlistPath, room.Adminlist); err != nil {
		return err
	}

	blocklistPath := filepath.Join(clusterPath, "blocklist.txt")
	if err := checkAndWriteFile(blocklistPath, room.Blocklist); err != nil {
		return err
	}

	tokenPath := filepath.Join(clusterPath, "cluster_token.txt")
	if err := checkAndWriteFile(tokenPath, room.Token); err != nil {
		return err
	}

	return nil
}

func SaveWorldFile(world World) error {
	if world.LevelData == "" {
		return nil
	}

	if err := ensureDirExists(clusterPath); err != nil {
		return fmt.Errorf("无法创建目录：%v", err)
	}

	worldDir := filepath.Join(clusterPath, world.Name)
	if err := ensureDirExists(worldDir); err != nil {
		return fmt.Errorf("无法创建目录：%v", err)
	}

	serverIni := filepath.Join(clusterPath, "server.ini")
	if err := checkAndWriteFile(serverIni, getWorldIni(world)); err != nil {
		return err
	}

	levelData := filepath.Join(clusterPath, "leveldataoverride.lua")
	if err := checkAndWriteFile(levelData, world.LevelData); err != nil {
		return err
	}

	return nil
}

func SaveModFile(modContent string, worlds []World) error {
	if err := ensureDirExists(clusterPath); err != nil {
		return fmt.Errorf("无法创建目录：%v", err)
	}

	for _, world := range worlds {
		worldDir := filepath.Join(clusterPath, world.Name)
		if err := ensureDirExists(worldDir); err != nil {
			return fmt.Errorf("无法创建目录：%v", err)
		}

		modoverrides := filepath.Join(worldDir, "modoverrides.lua")
		if err := checkAndWriteFile(modoverrides, modContent); err != nil {
			return err
		}
	}

	if err := checkAndWriteFile(gameModCfgPath, extractWorkshopIDs(modContent)); err != nil {
		return err
	}

	return nil
}

func GameExec(cmdType, worldName, startType string) error {
	screenName := "DMP_LITE_" + worldName

	switch cmdType {
	case "start":
		if ScreenAlive(screenName) {
			return nil
		}
		cmd := getStartCmd(worldName, screenName, startType)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("启动 %s: %w, 输出=%s", worldName, err, out)
		}
		return nil

	case "stop":
		if !ScreenAlive(screenName) {
			return nil
		}

		if out, err := exec.Command("screen",
			"-S", screenName, "-p", "0", "-X", "stuff", "c_shutdown()\n",
		).CombinedOutput(); err != nil {
			return fmt.Errorf("发送 c_shutdown 到 %s: %w, 输出=%s", worldName, err, out)
		}

		deadline := time.Now().Add(15 * time.Second)
		for time.Now().Before(deadline) {
			time.Sleep(500 * time.Millisecond)
			if !ScreenAlive(screenName) {
				return nil
			}
		}

		_ = exec.Command("screen", "-S", screenName, "-X", "quit").Run()
		Logger.Info(fmt.Sprintf("超时强杀 %s", worldName))
		return nil

	case "update":
		go func() {
			_ = exec.Command("bash", "./manual_update.sh").Run()
		}()
		Logger.Info("执行后台更新脚本")

		return nil

	default:
		return fmt.Errorf("不支持的命令: %q", cmdType)
	}
}

func ScreenAlive(name string) bool {
	_ = exec.Command("screen", "-wipe").Run()
	out, _ := exec.Command("screen", "-ls", name).CombinedOutput()

	return strings.Contains(string(out), name)
}

func getStartCmd(worldName, screenName, startType string) *exec.Cmd {
	bin, dir := "./dontstarve_dedicated_server_nullrenderer", "dst/bin"
	if startType == "64-bit" {
		bin, dir = "./dontstarve_dedicated_server_nullrenderer_x64", "dst/bin64"
	}

	cmd := exec.Command("screen",
		"-d", "-h", "200", "-m",
		"-S", screenName,
		bin,
		"-console", "-cluster", "Cluster_1", "-shard", worldName,
	)
	cmd.Dir = dir

	return cmd
}

func checkAndWriteFile(filePath, content string) error {
	if err := ensureFileExists(filePath); err != nil {
		return fmt.Errorf("无法创建文件：%v", err)
	}
	if err := truncAndWriteFile(filePath, content); err != nil {
		return fmt.Errorf("无法写文件：%v", err)
	}

	return nil
}

// workshopKeyRe 匹配 lua 配置中的 ["workshop-<ID>"] 键
var workshopKeyRe = regexp.MustCompile(`\["workshop-(\d+)"\]`)

// ExtractWorkshopIDs 从 lua 配置字符串中提取所有 workshop ID，
// 返回每行一个的 ServerModSetup("ID") 字符串。
func extractWorkshopIDs(luaStr string) string {
	matches := workshopKeyRe.FindAllStringSubmatch(luaStr, -1)

	ids := make([]string, 0, len(matches))
	for _, m := range matches {
		ids = append(ids, `ServerModSetup("`+m[1]+`")`)
	}
	return strings.Join(ids, "\n")
}

func getClusterIni(room Room) string {
	contents := `[GAMEPLAY]
game_mode = ` + room.GameMode + `
max_players = ` + strconv.Itoa(room.MaxPlayer) + `
pvp = ` + strconv.FormatBool(room.Pvp) + `
pause_when_empty = ` + strconv.FormatBool(room.PauseEmpty) + `
vote_enabled = ` + strconv.FormatBool(room.Vote) + `
vote_kick_enabled = ` + strconv.FormatBool(room.Vote) + `

[NETWORK]
cluster_description = ` + room.Description + `
whitelist_slots = ` + strconv.Itoa(len(strings.Split(room.Whitelist, "\n"))) + `
cluster_name = ` + room.GameName + `
cluster_password = ` + room.Password + `
cluster_language = zh
tick_rate = 15

[MISC]
console_enabled = true
max_snapshots = ` + strconv.Itoa(room.MaxRollBack) + `

[SHARD]
shard_enabled = true
bind_ip = 0.0.0.0
master_ip = ` + room.MasterIP + `
master_port = ` + strconv.Itoa(room.MasterPort) + `
cluster_key = ` + room.ClusterKey

	return contents
}

func getWorldIni(world World) string {
	contents := `[NETWORK]
server_port = ` + strconv.Itoa(world.ServerPort) + `

[SHARD]
id = ` + strconv.Itoa(world.ID) + `
is_master = ` + strconv.FormatBool(world.IsMaster) + `
name = ` + world.Name + `

[STEAM]
master_server_port = ` + strconv.Itoa(world.MasterServerPort) + `
authentication_port = ` + strconv.Itoa(world.AuthenticationPort) + `

[ACCOUNT]
encode_user_path = ` + strconv.FormatBool(world.EncodeUserPath)

	return contents
}

func truncAndWriteFile(fileName string, fileContent string) error {
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

func mustResolveClusterPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		panic(fmt.Errorf("获取当前用户 Home 目录失败: %w", err))
	}

	return filepath.Join(homeDir, ".klei", "DoNotStarveTogether", "Cluster_1")
}

func ensureDirExists(dirPath string) error {
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

func ensureFileExists(filePath string) error {
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
