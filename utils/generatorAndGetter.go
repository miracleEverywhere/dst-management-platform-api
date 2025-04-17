package utils

import (
	"fmt"
)

/* ============== world 世界相关 ============== */

func (world World) GeneratePlayersListCMD() string {
	return "screen -S \"" + world.ScreenName + "\" -p 0 -X stuff \"for i, v in ipairs(TheNet:GetClientTable()) do  print(string.format(\\\"playerlist %s [%d] %s <-@dmp@-> %s <-@dmp@-> %s\\\", 99999999, i-1, v.userid, v.name, v.prefab )) end$(printf \\\\r)\"\n"
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
