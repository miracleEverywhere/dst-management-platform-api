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

func (g *Game) GetModConfigureOptions(worldID, modID int, ugc bool) (*[]ConfigurationOption, error) {
	return g.getModConfigureOptions(worldID, modID, ugc)
}

// ModEnable 保存文件，返回给handler函数保存到数据库
func (g *Game) ModEnable(worldID, modID int, ugc bool) error {
	return g.modEnable(worldID, modID, ugc)
}
