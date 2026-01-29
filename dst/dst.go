package dst

import (
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
)

// SaveAll 保存所有配置文件
func (g *Game) SaveAll() error {
	var err error

	// cluster
	err = g.createRoom()
	if err != nil {
		return err
	}

	// worlds
	err = g.createWorlds()
	if err != nil {
		return err
	}

	return nil
}

// StartWorld 启动一个世界
func (g *Game) StartWorld(id int) error {
	return g.startWorld(id)
}

// StartAllWorld 启动所有世界
func (g *Game) StartAllWorld() error {
	return g.startAllWorld()
}

// StopWorld 关闭一个世界
func (g *Game) StopWorld(id int) error {
	return g.stopWorld(id)
}

// StopAllWorld 关闭所有世界
func (g *Game) StopAllWorld() error {
	return g.stopAllWorld()
}

func (g *Game) WorldUpStatus(id int) bool {
	return g.worldUpStatus(id)
}

func (g *Game) WorldPerformanceStatus(id int) PerformanceStatus {
	return g.worldPerformanceStatus(id)
}

// DeleteWorld 删除指定世界
func (g *Game) DeleteWorld(id int) error {
	return g.deleteWorld(id)
}

// Reset 重置世界，force：关闭世界--删除世界--启动世界
func (g *Game) Reset(force bool) error {
	return g.reset(force)
}

// Announce 宣告，会循环所有世界，直到执行成功
func (g *Game) Announce(message string) error {
	return g.announce(message)
}

func (g *Game) SystemMsg(message string) error {
	return g.systemMsg(message)
}

// ConsoleCmd 指定世界执行命令
func (g *Game) ConsoleCmd(cmd string, worldID int) error {
	return g.consoleCmd(cmd, worldID)
}

// SessionInfo 获取存档信息
func (g *Game) SessionInfo() *RoomSessionInfo {
	return g.sessionInfo()
}

// DownloadMod 下载模组
func (g *Game) DownloadMod(id int, fileURL string) (error, int64) {
	return g.downloadMod(id, fileURL)
}

// GetDownloadedMods 获取已经下载的模组
func (g *Game) GetDownloadedMods() *[]DownloadedMod {
	return g.getDownloadedMods()
}

// GetModConfigureOptions 返回动态表单结构
func (g *Game) GetModConfigureOptions(worldID, modID int, ugc bool) (*[]ConfigurationOption, error) {
	return g.getModConfigureOptions(worldID, modID, ugc)
}

// GetModConfigureOptionsValues 返回动态表单数据
func (g *Game) GetModConfigureOptionsValues(worldID, modID int, ugc bool) (*ModORConfig, error) {
	return g.getModConfigureOptionsValues(worldID, modID, ugc)
}

// ModConfigureOptionsValuesChange 修改mod配置，返回给handler函数保存到数据库
func (g *Game) ModConfigureOptionsValuesChange(worldID, modID int, modConfig *ModORConfig) error {
	return g.modConfigureOptionsValuesChange(worldID, modID, modConfig)
}

// ModEnable 启用mod，保存文件，返回给handler函数保存到数据库
func (g *Game) ModEnable(worldID, modID int, ugc bool) error {
	return g.modEnable(worldID, modID, ugc)
}

// GetEnabledMods 获取启用的mod列表
func (g *Game) GetEnabledMods(worldID int) ([]DownloadedMod, error) {
	return g.getEnabledMods(worldID)
}

// ModDisable 禁用mod，保存文件，返回给handler函数保存到数据库
func (g *Game) ModDisable(modID int) error {
	return g.modDisable(modID)
}

// ModDelete 删除模组
func (g *Game) ModDelete(modID int, fileURL string) error {
	return g.deleteMod(modID, fileURL)
}

// LogContent 获取日志
func (g *Game) LogContent(logType string, id, lines int) []string {
	return g.getLogContent(logType, id, lines)
}

// HistoryFileList 获取历史日志文件列表
func (g *Game) HistoryFileList(logType string, id int) []string {
	return g.historyFileList(logType, id)
}

// HistoryFileContent 获取历史日志文件内容
func (g *Game) HistoryFileContent(logType, logfileName string, id int) string {
	return g.historyFileContent(logType, logfileName, id)
}

// LogsInfo 获取日志大小
func (g *Game) LogsInfo() LogInfo {
	return g.logsInfo()
}

// LogsClean 删除日志
func (g *Game) LogsClean(cleanLogs *CleanLogs) bool {
	return g.logsClean(cleanLogs)
}

// LogsList 获取日志文件列表
func (g *Game) LogsList(admin bool) []string {
	return g.logsList(admin)
}

// GetOnlinePlayerList 获取玩家列表
func (g *Game) GetOnlinePlayerList(id int) ([]string, error) {
	return g.getOnlinePlayerList(id)
}

// GetLastAliveTime 获取指定世界最后的存活时间
func (g *Game) GetLastAliveTime(id int) (string, error) {
	return g.getLastAliveTime(id)
}

// Backup 创建备份文件
func (g *Game) Backup() error {
	return g.backup()
}

// Restore 恢复备份
func (g *Game) Restore(filename string) (*SaveJson, error) {
	return g.restore(filename)
}

// GetBackups 获取备份文件
func (g *Game) GetBackups() ([]BackupFile, error) {
	return g.getBackups()
}

// DeleteBackups 批量删除备份文件，返回删除的个数
func (g *Game) DeleteBackups(filenames []string) int {
	return g.deleteBackups(filenames)
}

// RunningScreens 获取正在运行的screen
func (g *Game) RunningScreens() ([]string, error) {
	return g.runningScreen()
}

// DeleteRoom 删除房间相关文件
func (g *Game) DeleteRoom() error {
	return g.deleteRoom()
}

// AddPlayerList 三个名单添加uid
func (g *Game) AddPlayerList(uids []string, listType string) error {
	return g.addPlayerList(uids, listType)
}

// RemovePlayerList 三个名单删除uid
func (g *Game) RemovePlayerList(uid, listType string) error {
	return g.removePlayerList(uid, listType)
}

// GetPlayerList 获取三个名单
func (g *Game) GetPlayerList(listType string) []string {
	switch listType {
	case "adminlist":
		logger.Logger.Debug(utils.StructToFlatString(g.adminlist))
		return g.adminlist
	case "blocklist":
		return g.blocklist
	case "whitelist":
		return g.whitelist
	default:
		return []string{}
	}
}

// GenerateBackgroundMap filepath: 最新的存档文件 返回背景地图base64
func (g *Game) GenerateBackgroundMap(worldID int) (MapData, error) {
	return g.generateBackgroundMap(worldID)
}

// CoordinateToPx 返回地图上的xy坐标
func (g *Game) CoordinateToPx(size, a, b int) (int, int) {
	return coordinateToPx(size, a, b)
}

// GetCoordinate 获取游戏内prefab的坐标
func (g *Game) GetCoordinate(cmd string, worldID int) (int, int, error) {
	return g.getCoordinate(cmd, worldID)
}

// CountPrefabs 统计指定世界prefab的个数
func (g *Game) CountPrefabs(worldID int) []PrefabItem {
	return g.countPrefabs(worldID)
}

// PlayerPosition 获取玩家实时坐标
func (g *Game) PlayerPosition(worldID int) []PlayerPosition {
	return g.playerPosition(worldID)
}

// GetSnapshot 获取饥荒存档文件
func (g *Game) GetSnapshot() ([]SnapshotFile, error) {
	return g.getSnapshot()
}

// DeleteSnapshot 删除饥荒存档文件，所有世界
func (g *Game) DeleteSnapshot(filename string) error {
	return g.deleteSnapshot(filename)
}
