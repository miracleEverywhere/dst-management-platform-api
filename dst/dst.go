package dst

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

func (g *Game) StartWorld(id int) error {
	return g.startWorld(id)
}

func (g *Game) StartAllWorld() error {
	return g.startAllWorld()
}

func (g *Game) DownloadMod(id int, ugc bool) {
	go g.downloadMod(id, ugc)
}

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
