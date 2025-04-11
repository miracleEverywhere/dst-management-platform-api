package utils

import (
	"fmt"
)

func (world World) GeneratePlayersListCMD() string {
	return "screen -S \"" + world.ScreenName + "\" -p 0 -X stuff \"for i, v in ipairs(TheNet:GetClientTable()) do  print(string.format(\\\"playerlist %s [%d] %s <-@dmp@-> %s <-@dmp@-> %s\\\", 99999999, i-1, v.userid, v.name, v.prefab )) end$(printf \\\\r)\"\n"
}

func (world World) GetServerLogPath(clusterName string) string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s/server_log.txt", HomeDir, clusterName, world.Name)
}

func (world World) GetChatLogPath(clusterName string) string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s/%s/server_chat_log.txt", HomeDir, clusterName, world.Name)
}

func (cluster Cluster) GetUIDMapPath() string {
	return fmt.Sprintf("./dmp_files/uid_map/%s.json", cluster.ClusterSetting.ClusterName)
}

func (cluster Cluster) GetBackupPath() string {
	return fmt.Sprintf("./dmp_files/backup/%s", cluster.ClusterSetting.ClusterName)
}

func (cluster Cluster) GetMainPath() string {
	return fmt.Sprintf("%s/.klei/DoNotStarveTogether/%s", HomeDir, cluster.ClusterSetting.ClusterName)
}
