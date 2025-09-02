package utils

import (
	"encoding/hex"
	"fmt"
	"github.com/shirou/gopsutil/v3/process"
	"regexp"
	"strconv"
	"strings"
	"time"
)

/* ============== world 世界相关 ============== */

func (world World) GeneratePlayersListCMD() string {
	return "screen -S \"" + world.ScreenName + "\" -p 0 -X stuff \"for i, v in ipairs(TheNet:GetClientTable()) do  print(string.format(\\\"playerlist %s [%d] %s <-@dmp@-> %s <-@dmp@-> %s\\\", 99999999, i-1, v.userid, v.name, v.prefab )) end$(printf \\\\r)\"\n"
}

func (world World) GetProcessStatus(clusterName string) (bool, float64, float64, float64, int64) {
	diskUsed, err := GetDirSize(world.GetMainPath(clusterName))
	if err != nil {
		Logger.Warn("获取世界磁盘使用量失败", "world", world.Name, "err", err)
		diskUsed = 0
	}

	status := world.GetStatus()
	if !status {
		return false, 0, 0, 0, diskUsed
	}

	cmd := fmt.Sprintf("ps -ef | grep $(ps -ef | grep %s | grep dontstarve_dedicated_server_nullrenderer | grep -v grep | awk '{print $2}') | grep -v grep | grep -vi screen |awk '{print $2}'", world.ScreenName)
	out, _, _ := BashCMDOutput(cmd)

	if len(out) < 2 {
		Logger.Warn("获取世界PID失败", "world", world.Name)
		return true, 0, 0, 0, diskUsed
	}

	pid, err := strconv.Atoi(strings.TrimSpace(out))
	if err != nil {
		Logger.Warn("获取世界PID失败", "world", world.Name, "err", err)
		return true, 0, 0, 0, diskUsed
	}

	p, err := process.NewProcess(int32(pid))
	if err != nil {
		Logger.Warn("获取世界进程失败", "world", world.Name, "err", err)
		return true, 0, 0, 0, diskUsed
	}

	cpu, err := p.Percent(time.Millisecond * 100)
	if err != nil {
		Logger.Warn("获取世界CPU失败", "world", world.Name, "err", err)
		return true, 0, 0, 0, diskUsed
	}

	mem, err := p.MemoryPercent()
	if err != nil {
		Logger.Warn("获取世界内存使用率失败", "world", world.Name, "err", err)
		return true, cpu, 0, 0, diskUsed
	}

	memSize, err := p.MemoryInfo()
	if err != nil {
		Logger.Warn("获取世界内存使用量失败", "world", world.Name, "err", err)
		return true, cpu, float64(mem), 0, diskUsed
	}

	return true, cpu, float64(mem), float64(memSize.RSS / 1024 / 1024), diskUsed
}

func (world World) GetWorldType() string {
	re := regexp.MustCompile(`location\s*=\s*"([^"]+)"`)
	matches := re.FindStringSubmatch(world.LevelData)

	if len(matches) >= 2 {
		return matches[1] // 输出: Location: forest
	} else {
		return "None"
	}
}

func (world World) GetMainPath(clusterName string) string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s", HomeDir, clusterName, world.Name)
}

func (world World) GetServerLogFile(clusterName string) string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s/server_log.txt", HomeDir, clusterName, world.Name)
}

func (world World) GetBackupServerLogPath(clusterName string) string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s/backup/server_log", HomeDir, clusterName, world.Name)
}

func (world World) GetChatLogFile(clusterName string) string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s/server_chat_log.txt", HomeDir, clusterName, world.Name)
}

func (world World) GetBackupChatLogPath(clusterName string) string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s/backup/server_chat_log", HomeDir, clusterName, world.Name)
}

func (world World) GetLevelDataFile(clusterName string) string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s/leveldataoverride.lua", HomeDir, clusterName, world.Name)
}

func (world World) GetModFile(clusterName string) string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s/modoverrides.lua", HomeDir, clusterName, world.Name)
}

func (world World) GetIniFile(clusterName string) string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s/server.ini", HomeDir, clusterName, world.Name)
}

func (world World) GetSavePath(clusterName string) string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s/save", HomeDir, clusterName, world.Name)
}

func (world World) GetSessionPath(clusterName string) string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s/save/session", HomeDir, clusterName, world.Name)
}

func (world World) GetDstModPath(clusterName string) string {
	return fmt.Sprintf("dst/ugc_mods/%s/%s/content/322330", clusterName, world.Name)
}

// GetStatus 获取世界状态
func (world World) GetStatus() bool {
	cmd := fmt.Sprintf("ps -ef | grep %s | grep -v grep", world.ScreenName)
	err := BashCMD(cmd)
	if err != nil {
		return false
	} else {
		return true
	}
}

/* ============== cluster 集群相关 ============== */

func (cluster Cluster) GetUIDMapFile() string {
	return fmt.Sprintf("%s/%s.json", UidFilePath, cluster.ClusterSetting.ClusterName)
}

func (cluster Cluster) GetBackupPath() string {
	return fmt.Sprintf("%s/%s", BackupPath, cluster.ClusterSetting.ClusterName)
}

func (cluster Cluster) GetMainPath() string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s", HomeDir, cluster.ClusterSetting.ClusterName)
}

func (cluster Cluster) GetIniFile() string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/cluster.ini", HomeDir, cluster.ClusterSetting.ClusterName)
}

func (cluster Cluster) GetTokenFile() string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/cluster_token.txt", HomeDir, cluster.ClusterSetting.ClusterName)
}

func (cluster Cluster) GetAdminListFile() string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/adminlist.txt", HomeDir, cluster.ClusterSetting.ClusterName)
}

func (cluster Cluster) GetBlockListFile() string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/blocklist.txt", HomeDir, cluster.ClusterSetting.ClusterName)
}

func (cluster Cluster) GetWhiteListFile() string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/whitelist.txt", HomeDir, cluster.ClusterSetting.ClusterName)
}

func (cluster Cluster) GetWhiteListSlot() int {
	fileContent, err := ReadLinesToSlice(cluster.GetWhiteListFile())
	if err != nil {
		Logger.Info("没有找到白名单文件", "err", err)
		return 0
	}

	var whiteList []string
	for _, i := range fileContent {
		uid := strings.TrimSpace(i)
		if uid != "" {
			whiteList = append(whiteList, uid)
		}
	}

	return len(whiteList)
}

func (cluster Cluster) GetModUgcPath() []string {
	var paths []string
	for _, world := range cluster.Worlds {
		paths = append(paths, fmt.Sprintf("dst/ugc_mods/%s/%s/content/322330", cluster.ClusterSetting.ClusterName, world.Name))
	}
	return paths
}

func (cluster Cluster) GetModUgcPathRoot() string {
	return fmt.Sprintf("dst/ugc_mods/%s", cluster.ClusterSetting.ClusterName)
}

func (cluster Cluster) GetModNoUgcPath() string {
	return "dst/mods"
}

func (cluster Cluster) GetKickCmd(uid string) string {
	return fmt.Sprintf("TheNet:Kick('%s')", uid)
}

func (cluster Cluster) GetKillCmd(uid string) string {
	return fmt.Sprintf(`for k, v in pairs(AllPlayers) do if v.userid=='%s' then AllPlayers[k]:PushEvent('death') end end`, uid)
}

func (cluster Cluster) GetRespawnCmd(uid string) string {
	return fmt.Sprintf(`UserToPlayer('%s'):PushEvent('respawnfromghost')`, uid)
}

func (cluster Cluster) GetDespawnCmd(uid string) string {
	return fmt.Sprintf(`for k, v in pairs(AllPlayers) do if v.userid=='%s' then c_despawn(AllPlayers[k]) end end`, uid)
}

/* ============== config 配置相关 ============== */

func (config Config) GetClusterWithName(clusterName string) (Cluster, error) {
	for _, cluster := range config.Clusters {
		if cluster.ClusterSetting.ClusterName == clusterName {
			return cluster, nil
		}
	}
	return Cluster{}, fmt.Errorf("没有发现名为%s的集群", clusterName)
}

func (config Config) GetClusterAndIndexWithName(clusterName string) (int, Cluster, error) {
	for index, cluster := range config.Clusters {
		if cluster.ClusterSetting.ClusterName == clusterName {
			return index, cluster, nil
		}
	}
	return -1, Cluster{}, fmt.Errorf("没有发现名为%s的集群", clusterName)
}

func (config Config) GetWorldWithName(clusterName, worldName string) (World, error) {
	for _, cluster := range config.Clusters {
		if cluster.ClusterSetting.ClusterName == clusterName {
			for _, world := range cluster.Worlds {
				if world.Name == worldName {
					return world, nil
				}
			}
		}
	}
	return World{}, fmt.Errorf("在集群%s中，没有发现名为%s的世界", clusterName, worldName)
}

func (config Config) GetUserWithUsername(username string) User {
	for _, user := range config.Users {
		if user.Username == username {
			return user
		}
	}
	return User{}
}

/* ============== Key API相关 ============== */

func GetSteamApiKey() string {
	obfuscated := []byte{
		0xD5, 0xED, 0xDA, 0x66, 0x64, 0xFF, 0x23, 0xA6,
		0xB3, 0xD8, 0x50, 0x2C, 0x63, 0xB1, 0xBF, 0x6D,
	}
	var data []byte
	for _, b := range obfuscated {
		data = append(data, b^0x55)
	}
	return hex.EncodeToString(data)
}

func GetDSTUpdateCmd() string {
	return "cd ~/steamcmd ; ./steamcmd.sh +login anonymous +force_install_dir ~/dst +app_update 343050 validate +quit"
}

func GetMacVersionCmd() string {
	return "cd dst/dontstarve_dedicated_server_nullrenderer.app/Contents/MacOS && strings dontstarve_dedicated_server_nullrenderer | grep -A 1 PRODUCTION | grep -E '\\d+'"
}

func GetDSTRoomsApi(region string) string {
	return fmt.Sprintf("https://lobby-v2-cdn.klei.com/%s-Steam.json.gz", region)
}

func GetAesKey() []byte {
	var (
		decoded []byte
		key     = byte(0x55)
	)

	encoded := []byte{
		100 ^ key, 77 ^ key, 80 ^ key, 95 ^ key,
		97 ^ key, 69 ^ key, 83 ^ key, 95 ^ key,
		50 ^ key, 112 ^ key, 118 ^ key, 118 ^ key,
		68 ^ key, 95 ^ key, 103 ^ key, 79 ^ key,
	}

	for _, b := range encoded {
		decoded = append(decoded, b^key)
	}

	return decoded
}

func GetPlayerLocationCmd(uid string) string {
	return fmt.Sprintf("print(UserToPlayer('%s').Transform:GetWorldPosition())", uid)
}

/* Linux交叉编译：CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -o /root/dmp_darwin */
/* MacOS编译：CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 ~/go/go1.23.1/bin/go build -o /Users/xxx/dmp */
