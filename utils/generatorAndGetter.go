package utils

import (
	"fmt"
)

/* ============== world 世界相关 ============== */

func (world World) GeneratePlayersListCMD() string {
	return "screen -S \"" + world.ScreenName + "\" -p 0 -X stuff \"for i, v in ipairs(TheNet:GetClientTable()) do  print(string.format(\\\"playerlist %s [%d] %s <-@dmp@-> %s <-@dmp@-> %s\\\", 99999999, i-1, v.userid, v.name, v.prefab )) end$(printf \\\\r)\"\n"
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
