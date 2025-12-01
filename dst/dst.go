package dst

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

// ConsoleCmd 指定世界执行命令
func (g *Game) ConsoleCmd(cmd string, worldID int) error {
	return g.consoleCmd(cmd, worldID)
}

// SessionInfo 获取存档信息
func (g *Game) SessionInfo() *RoomSessionInfo {
	return g.sessionInfo()
}

// DownloadMod 下载模组
func (g *Game) DownloadMod(id int, fileURL string, update bool) {
	if update {
		g.downloadMod(id, fileURL)
	} else {
		go g.downloadMod(id, fileURL)
	}
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

// LogContent 获取日志
func (g *Game) LogContent(logType string, id, lines int) []string {
	return g.getLogContent(logType, id, lines)
}

// GetPlayerList 获取玩家列表
func (g *Game) GetPlayerList(id int) ([]string, error) {
	return g.getPlayerList(id)
}
