package utils

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

/* ============== world 世界相关 ============== */

func (world World) GeneratePlayersListCMD() string {
	return "screen -S \"" + world.ScreenName + "\" -p 0 -X stuff \"for i, v in ipairs(TheNet:GetClientTable()) do  print(string.format(\\\"playerlist %s [%d] %s <-@dmp@-> %s <-@dmp@-> %s\\\", 99999999, i-1, v.userid, v.name, v.prefab )) end$(printf \\\\r)\"\n"
}

func (world World) GetProcessStatus() (float64, float64) {
	cmd := fmt.Sprintf("top -b -n 1 -p $(ps -ef | grep $(ps -ef | grep %s | grep -v grep | awk '{print $2}') | grep -v grep | grep -vi screen |awk '{print $2}') | tail -1 | awk '{print $9\"-\"$10}'", world.ScreenName)
	out, _, _ := BashCMDOutput(cmd)

	if len(out) < 2 {
		Logger.Warn("获取世界CPU内存失败", "world", world.Name)
		return 0, 0
	}

	out = strings.TrimSpace(out)
	stats := strings.Split(out, "-")

	cpu, err := strconv.ParseFloat(stats[0], 64)
	if err != nil {
		cpu = 0
	}
	mem, err := strconv.ParseFloat(stats[1], 64)
	if err != nil {
		mem = 0
	}

	return cpu, mem
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

func (world World) GetSessionPath(clusterName string) string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s/save/session", HomeDir, clusterName, world.Name)
}

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
	return fmt.Sprintf("./dmp_files/uid_map/%s.json", cluster.ClusterSetting.ClusterName)
}

func (cluster Cluster) GetBackupPath() string {
	return fmt.Sprintf("./dmp_files/backup/%s", cluster.ClusterSetting.ClusterName)
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

/* ============== config 配置相关 ============== */

func (config Config) GetClusterWithName(clusterName string) (Cluster, error) {
	for _, cluster := range config.Clusters {
		if cluster.ClusterSetting.ClusterName == clusterName {
			return cluster, nil
		}
	}
	return Cluster{}, fmt.Errorf("没有发现名为%s的集群", clusterName)
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
