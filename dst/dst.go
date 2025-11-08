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
