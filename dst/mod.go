package dst

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/yuin/gopher-lua"
)

type modSaveData struct {
	ugcPath string
}

func (g *Game) dsModsSetup() error {
	g.roomMutex.Lock()
	defer g.roomMutex.Unlock()

	var modData string
	if g.room.ModInOne {
		modData = g.room.ModData
	} else {
		modData = g.worldSaveData[0].ModData
	}

	L := lua.NewState()
	defer L.Close()
	if err := L.DoString(modData); err != nil {
		return err
	}
	modsTable := L.Get(-1)
	fileContent := ""
	if tbl, ok := modsTable.(*lua.LTable); ok {
		// 有配置，但为空
		if tbl.Len() == 0 {
			err := utils.TruncAndWriteFile(utils.GameModSettingPath, fileContent)
			if err != nil {
				return err
			}
		}
		tbl.ForEach(func(key lua.LValue, value lua.LValue) {
			// 检查键是否是字符串，并且以 "workshop-" 开头
			if strKey, ok := key.(lua.LString); ok && strings.HasPrefix(string(strKey), "workshop-") {
				// 提取 "workshop-" 后面的数字
				workshopID := strings.TrimPrefix(string(strKey), "workshop-")
				fileContent = fileContent + "ServerModSetup(\"" + workshopID + "\")\n"
			}
		})
		// 有配置，不为空
		err := utils.TruncAndWriteFile(utils.GameModSettingPath, fileContent)
		if err != nil {
			return err
		}
	} else {
		// 无配置
		err := utils.TruncAndWriteFile(utils.GameModSettingPath, fileContent)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Game) downloadMod(id int, fileURL string) (error, int64) {
	atomic.AddInt32(&db.ModDownloadExecuting, 1)
	defer atomic.AddInt32(&db.ModDownloadExecuting, -1)

	var (
		err     error
		ugc     bool
		modSize int64
	)

	if fileURL == "" {
		ugc = true
	}

	if ugc {
		// 1. ugc mod 统一下载到 dmp_files/ugc, 也就是dmp_files/ugc/{cluster}/steamapps/workshop{appworkshop_322330.acf  content  downloads}
		// 2. 下载完成后，将下载的mod文件全部移动至dst/ugc_mods/{cluster}/{worlds}/ 删除-复制
		// 3. 读取游戏acf文件和dmp_files的acf文件，更新当前mod-id所对应的所有字段

		// 1
		downloadCmd := g.generateModDownloadCmd(id)
		logger.Logger.Debug(downloadCmd)
		err = utils.BashCMD(downloadCmd)
		if err != nil {
			logger.Logger.Error("下载模组失败", "err", err)
			return err, modSize
		}
		time.Sleep(500 * time.Millisecond)

		// 2
		err = g.removeGameOldMod(id)
		if err != nil {
			logger.Logger.Error("移动模组失败", "err", err)
			return err, modSize
		}
		copyCmd := g.generateModCopyCmd(id)
		logger.Logger.Debug(copyCmd)
		err = utils.BashCMD(copyCmd)
		if err != nil {
			logger.Logger.Error("移动模组失败", "err", err)
			return err, modSize
		}
		time.Sleep(500 * time.Millisecond)

		// 3
		gameAcfPath := fmt.Sprintf("dst/ugc_mods/%s/%s/appworkshop_322330.acf", g.clusterName, g.worldSaveData[0].WorldName)
		gameAcfContent, err := utils.ReadLinesToSlice(gameAcfPath)
		if err != nil {
			gameAcfContent = []string{}
		}
		err = g.processAcf(id)
		if err != nil {
			logger.Logger.Error("修改acf文件失败", "err", err)
			// 下载失败就恢复下载前的acf文件
			logger.Logger.Info("正在恢复旧的acf文件")
			for _, world := range g.worldSaveData {
				gameAcfPath = fmt.Sprintf("dst/ugc_mods/%s/%s/appworkshop_322330.acf", g.clusterName, world.WorldName)
				writeErr := utils.WriteLinesFromSlice(gameAcfPath, gameAcfContent)
				if writeErr != nil {
					logger.Logger.Error("恢复acf文件失败", "err", writeErr)
				}
			}

			return err, modSize
		}
		time.Sleep(500 * time.Millisecond)

		modSize, err = utils.GetDirSize(fmt.Sprintf("dst/ugc_mods/%s/%s/content/322330/%d", g.clusterName, g.worldSaveData[0].WorldName, id))
		logger.Logger.DebugF("模组路径为%s", fmt.Sprintf("dst/ugc_mods/%s/%s/content/322330/%d", g.clusterName, g.worldSaveData[0].WorldName, id))
		logger.Logger.DebugF("模组大小为%d", modSize)
		if err != nil {
			logger.Logger.Error("获取模组大小失败", "err", err)
			return err, modSize
		}

	} else {
		// 1. 下载zip文件并保存
		// 2. 解压zip文件至dst/mods/workshop-id
		err, modSize = downloadNotUGCMod(fileURL, id)
		if err != nil {
			logger.Logger.Error("下载mod失败", "err", err)
			return err, modSize
		}
	}

	return nil, modSize
}

func (g *Game) generateModDownloadCmd(id int) string {
	return fmt.Sprintf("steamcmd/steamcmd.sh +force_install_dir %s/%s/mods/ugc/%s +login anonymous +workshop_download_item 322330 %d +quit", db.CurrentDir, utils.DmpFiles, g.clusterName, id)
}

func (g *Game) removeGameOldMod(id int) error {
	for _, world := range g.worldSaveData {
		path := fmt.Sprintf("dst/ugc_mods/%s/%s/content/322330/%d", g.clusterName, world.WorldName, id)
		err := utils.RemoveDir(path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *Game) generateModCopyCmd(id int) string {
	if len(g.worldSaveData) == 0 {
		return ""
	}

	dmpPath := fmt.Sprintf("%s/mods/ugc/%s/steamapps/workshop/content/322330/%d", utils.DmpFiles, g.clusterName, id)

	var cmds []string

	// 生成 复制 命令
	for _, world := range g.worldSaveData {
		gamePath := fmt.Sprintf("dst/ugc_mods/%s/%s/content/322330/%d", g.clusterName, world.WorldName, id)
		cmd := fmt.Sprintf("mkdir -p dst/ugc_mods/%s/%s/content/322330", g.clusterName, world.WorldName)
		cmds = append(cmds, cmd)
		cmd = fmt.Sprintf("cp -r %s %s", dmpPath, gamePath)
		cmds = append(cmds, cmd)
	}

	return strings.Join(cmds, " && ")
}

func (g *Game) processAcf(id int) error {
	g.acfMutex.Lock()
	defer g.acfMutex.Unlock()

	acfID := strconv.Itoa(id)

	dmpAcfPath := fmt.Sprintf("%s/mods/ugc/%s/steamapps/workshop/appworkshop_322330.acf", utils.DmpFiles, g.clusterName)
	gameAcfPath := fmt.Sprintf("dst/ugc_mods/%s/%s/appworkshop_322330.acf", g.clusterName, g.worldSaveData[0].WorldName)

	err := utils.EnsureFileExists(gameAcfPath)
	if err != nil {
		logger.Logger.Error("EnsureFileExists失败", "path", gameAcfPath)
		return err
	}

	dmpAcfContent, err := os.ReadFile(dmpAcfPath)
	if err != nil {
		return err
	}
	gameAcfContent, err := os.ReadFile(gameAcfPath)
	if err != nil {
		return err
	}

	dmpAcfParser := NewAcfParser(string(dmpAcfContent))

	var writtenContent string

	if len(gameAcfContent) == 0 {
		// 如果游戏mod目录没有acf文件，直接使用dmp下载的acf文件
		writtenContent = dmpAcfParser.FileContent()
	} else {
		// 如果游戏mod目录含有acf文件，处理游戏acf文件
		gameAcfParser := NewAcfParser(string(gameAcfContent))
		var (
			gameAcfTargetIndex int
			hasMod             bool
		)

		if len(gameAcfParser.AppWorkshop.WorkshopItemsInstalled) > len(gameAcfParser.AppWorkshop.WorkshopItemDetails) {
			// 防止index溢出导致接口500
			return fmt.Errorf("acf文件异常，WorkshopItemsInstalled与WorkshopItemDetails长度不一致")
		}

		for index, i := range gameAcfParser.AppWorkshop.WorkshopItemsInstalled {
			if i.ID == acfID {
				gameAcfTargetIndex = index
				hasMod = true
			}
		}
		if hasMod {
			for index, mod := range dmpAcfParser.AppWorkshop.WorkshopItemsInstalled {
				if strconv.Itoa(id) == mod.ID {
					gameAcfParser.AppWorkshop.WorkshopItemsInstalled[gameAcfTargetIndex] = dmpAcfParser.AppWorkshop.WorkshopItemsInstalled[index]
					gameAcfParser.AppWorkshop.WorkshopItemDetails[gameAcfTargetIndex] = dmpAcfParser.AppWorkshop.WorkshopItemDetails[index]
				}
			}
		} else {
			for index, mod := range dmpAcfParser.AppWorkshop.WorkshopItemsInstalled {
				if strconv.Itoa(id) == mod.ID {
					gameAcfParser.AppWorkshop.WorkshopItemsInstalled = append(gameAcfParser.AppWorkshop.WorkshopItemsInstalled, dmpAcfParser.AppWorkshop.WorkshopItemsInstalled[index])
					gameAcfParser.AppWorkshop.WorkshopItemDetails = append(gameAcfParser.AppWorkshop.WorkshopItemDetails, dmpAcfParser.AppWorkshop.WorkshopItemDetails[index])
				}
			}

		}

		writtenContent = gameAcfParser.FileContent()
	}

	for _, world := range g.worldSaveData {
		gameAcfPath = fmt.Sprintf("dst/ugc_mods/%s/%s/appworkshop_322330.acf", g.clusterName, world.WorldName)
		err = utils.EnsureDirExists(fmt.Sprintf("%s/%s", g.ugcPath, world.WorldName))
		if err != nil {
			return err
		}
		err = utils.TruncAndWriteFile(gameAcfPath, writtenContent)
		if err != nil {
			return err
		}
	}

	return nil
}

type DownloadedMod struct {
	ID         int    `json:"id"`
	Name       string `json:"name"`
	LocalSize  string `json:"localSize"`
	ServerSize string `json:"serverSize"`
	FileURL    string `json:"file_url"`
	PreviewURL string `json:"preview_url"`
}

func (g *Game) getDownloadedMods() *[]DownloadedMod {
	var downloadedMods []DownloadedMod

	// 获取非ugc
	modDirs, err := utils.GetDirs("dst/mods", false)
	for _, dir := range modDirs {
		if strings.HasPrefix(dir, "workshop") {
			parts := strings.Split(dir, "-")
			if len(parts) == 2 {
				idStr := parts[len(parts)-1]
				id, err := strconv.Atoi(idStr)
				if err == nil {
					downloadedMods = append(downloadedMods, DownloadedMod{
						ID:        id,
						LocalSize: "0",
					})
				}
			}
		}
	}

	// 获取ugc
	gameAcfPath := fmt.Sprintf("dst/ugc_mods/%s/%s/appworkshop_322330.acf", g.clusterName, g.worldSaveData[0].WorldName)
	err = utils.EnsureFileExists(gameAcfPath)
	if err != nil {
		logger.Logger.Error("EnsureFileExists失败", "path", gameAcfPath)
		return &downloadedMods
	}

	gameAcfContent, err := os.ReadFile(gameAcfPath)
	if err != nil {
		return &downloadedMods
	}

	if len(gameAcfContent) != 0 {
		gameAcfParser := NewAcfParser(string(gameAcfContent))
		for _, mod := range gameAcfParser.AppWorkshop.WorkshopItemsInstalled {
			id, err := strconv.Atoi(mod.ID)
			if err != nil {
				id = 0
			}
			downloadedMods = append(downloadedMods, DownloadedMod{
				ID:        id,
				LocalSize: mod.Size,
			})
		}
	}

	return &downloadedMods
}

func (g *Game) getModConfigureOptions(worldID, modID int, ugc bool) (*[]ConfigurationOption, error) {
	var modinfoLuaPath string
	if g.room.ModInOne {
		if ugc {
			modinfoLuaPath = fmt.Sprintf("%s/%s/content/322330/%d/modinfo.lua", g.ugcPath, g.worldSaveData[0].WorldName, modID)
		} else {
			modinfoLuaPath = fmt.Sprintf("dst/mods/workshop-%d/modinfo.lua", modID)
		}
	} else {
		if ugc {
			var wi int
			for index, world := range g.worldSaveData {
				if worldID == world.ID {
					wi = index
					break
				}
			}
			modinfoLuaPath = fmt.Sprintf("%s/%s/content/322330/%d/modinfo.lua", g.ugcPath, g.worldSaveData[wi].WorldName, modID)
		} else {
			modinfoLuaPath = fmt.Sprintf("dst/mods/workshop-%d/modinfo.lua", modID)
		}
	}

	parser, err := NewModInfoParser(modinfoLuaPath, modID)
	if err != nil {
		logger.Logger.Error("读取modinfo文件失败", "err", err)
		return parser.Configuration, err
	}

	err = parser.Parse(g.lang)
	if err != nil {
		logger.Logger.Error("解析modinfo文件失败", "err", err)
		return parser.Configuration, err
	}

	return parser.Configuration, nil
}

func (g *Game) getModConfigureOptionsValues(worldID, modID int, ugc bool) (*ModORConfig, error) {
	modORParser := NewModORParser()
	defer modORParser.close()

	logger.Logger.DebugF("ugc is %t", ugc)

	var modORContent string
	if g.room.ModInOne {
		modORContent = g.room.ModData
	} else {
		world, err := g.getWorldByID(worldID)
		if err != nil {
			logger.Logger.Debug("这里出问题?", "err", err)
			return &ModORConfig{}, err
		}
		modORContent = world.ModData
	}

	mods, err := modORParser.Parse(modORContent, g.lang)
	if err != nil {
		logger.Logger.Debug("这里出问题?", "err", err)
		return &ModORConfig{}, err
	}

	for key, mod := range mods {
		modKey := fmt.Sprintf("workshop-%d", modID)
		if key == modKey {
			return mod, nil
		}
	}

	return &ModORConfig{}, fmt.Errorf("在modoverrides.lua文件中没有找到该mod的配置")
}

func (g *Game) modEnable(worldID, modID int, ugc bool) error {
	var (
		err     error
		options *[]ConfigurationOption
	)
	// 区分是否为禁本地配置
	if modID == 0 {
		options = &[]ConfigurationOption{}
	} else {
		options, err = g.getModConfigureOptions(worldID, modID, ugc)
		if err != nil {
			logger.Logger.Debug("这里出问题?", "err", err)
			return err
		}
	}

	newModConfig := &ModORConfig{
		ConfigurationOptions: make(map[string]any),
		Enabled:              true,
	}
	for _, option := range *options {
		key := option.Name
		value := option.Default
		newModConfig.ConfigurationOptions[key] = value
	}

	modORParser := NewModORParser()
	defer modORParser.close()

	var modORContent string
	if g.room.ModInOne {
		modORContent = g.room.ModData
		mods := make(ModORCollection)
		if modORContent != "" {
			mods, err = modORParser.Parse(modORContent, g.lang)
			if err != nil {
				logger.Logger.Debug("这里出问题?", "err", err)
				return err
			}
		}
		// 区分是否为禁本地配置
		if modID == 0 {
			mods.AddModConfig(fmt.Sprintf("client_mods_disabled"), newModConfig)
		} else {
			mods.AddModConfig(fmt.Sprintf("workshop-%d", modID), newModConfig)
		}
		newModORContent := mods.ToLuaCode()
		g.room.ModData = newModORContent
	} else {
		// 为保留每个世界的独立模组配置，需要分开处理，增加指定的mod，并修改db，最后返回
		worlds := *g.worlds
		for i, world := range g.worldSaveData {
			modORContent = world.ModData
			mods := make(ModORCollection)
			if modORContent != "" {
				mods, err = modORParser.Parse(modORContent, g.lang)
				if err != nil {
					logger.Logger.Debug("这里出问题?", "err", err)
					return err
				}
			}

			// 区分是否为禁本地配置
			if modID == 0 {
				mods.AddModConfig(fmt.Sprintf("client_mods_disabled"), newModConfig)
			} else {
				mods.AddModConfig(fmt.Sprintf("workshop-%d", modID), newModConfig)
			}
			newModORContent := mods.ToLuaCode()

			worlds[i].ModData = newModORContent
		}
	}

	// 统一保存文件
	return g.saveMods()
}

func (g *Game) saveMods() error {
	var modContent string

	for idx, world := range *g.worlds {
		if g.room.ModInOne {
			modContent = g.room.ModData
		} else {
			modContent = world.ModData
		}
		err := utils.TruncAndWriteFile(g.worldSaveData[idx].modOverridesPath, modContent)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Game) modConfigureOptionsValuesChange(worldID, modID int, modConfig *ModORConfig) error {
	g.modMutex.Lock()
	defer g.modMutex.Unlock()

	modORParser := NewModORParser()
	defer modORParser.close()

	var modORContent string
	if g.room.ModInOne {
		modORContent = g.room.ModData
	} else {
		world, err := g.getWorldByID(worldID)
		if err != nil {
			logger.Logger.Debug("这里出问题?", "err", err)
			return err
		}
		modORContent = world.ModData
	}

	mods, err := modORParser.Parse(modORContent, g.lang)
	if err != nil {
		logger.Logger.Debug("这里出问题?", "err", err)
		return err
	}

	modKey := fmt.Sprintf("workshop-%d", modID)

	mods[modKey] = modConfig

	newModORContent := mods.ToLuaCode()

	if g.room.ModInOne {
		g.room.ModData = newModORContent
	} else {
		for i := range g.worldSaveData {
			worlds := *g.worlds
			if worlds[i].ID == worldID {
				worlds[i].ModData = newModORContent
			}
		}
	}

	return g.saveMods()
}

func (g *Game) getEnabledMods(worldID int) ([]DownloadedMod, error) {
	modORParser := NewModORParser()
	defer modORParser.close()

	var modORContent string
	if g.room.ModInOne {
		modORContent = g.room.ModData
	} else {
		world, err := g.getWorldByID(worldID)
		if err != nil {
			logger.Logger.Debug("这里出问题?", "err", err)
			return []DownloadedMod{}, err
		}
		modORContent = world.ModData
	}

	if modORContent == "" {
		return []DownloadedMod{}, nil
	}

	mods, err := modORParser.Parse(modORContent, g.lang)
	if err != nil {
		logger.Logger.Debug("这里出问题?", "err", err)
		return []DownloadedMod{}, err
	}

	var modsID []DownloadedMod
	for k := range mods {
		modIDSlice := strings.Split(k, "-")
		var modID int
		if len(modIDSlice) < 2 {
			// 禁本地配置
			modID = 0
		} else {
			modID, err = strconv.Atoi(modIDSlice[1])
			if err != nil {
				modID = 0
			}
		}
		modsID = append(modsID, DownloadedMod{
			ID: modID,
		})
	}

	return modsID, nil
}

func (g *Game) modDisable(modID int) error {
	modORParser := NewModORParser()
	defer modORParser.close()

	var modORContent string
	if g.room.ModInOne {
		modORContent = g.room.ModData
		mods, err := modORParser.Parse(modORContent, g.lang)
		if err != nil {
			logger.Logger.Debug("这里出问题?", "err", err)
			return err
		}
		// 区分是否为禁本地配置
		if modID == 0 {
			delete(mods, fmt.Sprintf("client_mods_disabled"))
		} else {
			delete(mods, fmt.Sprintf("workshop-%d", modID))
		}

		newModORContent := mods.ToLuaCode()

		g.room.ModData = newModORContent
	} else {
		// 为保留每个世界的独立模组配置，需要分开处理，删除指定的mod，并修改db，最后返回
		worlds := *g.worlds
		for i, world := range g.worldSaveData {
			modORContent = world.ModData
			mods, err := modORParser.Parse(modORContent, g.lang)
			if err != nil {
				logger.Logger.Debug("这里出问题?", "err", err)
				return err
			}

			// 区分是否为禁本地配置
			if modID == 0 {
				delete(mods, fmt.Sprintf("client_mods_disabled"))
			} else {
				delete(mods, fmt.Sprintf("workshop-%d", modID))
			}

			newModORContent := mods.ToLuaCode()

			worlds[i].ModData = newModORContent
		}
	}

	return g.saveMods()
}

func (g *Game) deleteMod(modID int, fileURL string) error {
	var ugc bool

	if fileURL == "" {
		ugc = true
	}

	if ugc {
		g.acfMutex.Lock()
		defer g.acfMutex.Unlock()

		acfID := strconv.Itoa(modID)

		for _, world := range g.worldSaveData {
			gameAcfPath := fmt.Sprintf("dst/ugc_mods/%s/%s/appworkshop_322330.acf", g.clusterName, world.WorldName)

			err := utils.EnsureFileExists(gameAcfPath)
			if err != nil {
				logger.Logger.Error("acf文件不存在", "path", gameAcfPath)
				return err
			}
			gameAcfContent, err := os.ReadFile(gameAcfPath)
			if err != nil {
				return err
			}

			gameAcfParser := NewAcfParser(string(gameAcfContent))
			for index, mod := range gameAcfParser.AppWorkshop.WorkshopItemsInstalled {
				if mod.ID == acfID {
					gameAcfParser.AppWorkshop.WorkshopItemsInstalled = append(gameAcfParser.AppWorkshop.WorkshopItemsInstalled[:index], gameAcfParser.AppWorkshop.WorkshopItemsInstalled[index+1:]...)
					break
				}
			}
			for index, mod := range gameAcfParser.AppWorkshop.WorkshopItemDetails {
				if mod.ID == acfID {
					gameAcfParser.AppWorkshop.WorkshopItemDetails = append(gameAcfParser.AppWorkshop.WorkshopItemDetails[:index], gameAcfParser.AppWorkshop.WorkshopItemDetails[index+1:]...)
				}
			}

			writtenContent := gameAcfParser.FileContent()
			err = utils.TruncAndWriteFile(gameAcfPath, writtenContent)
			if err != nil {
				return err
			}

			modPath := fmt.Sprintf("dst/ugc_mods/%s/%s/content/322330/%d", g.clusterName, world.WorldName, modID)
			err = utils.RemoveDir(modPath)
			if err != nil {
				logger.Logger.Error("删除模组失败", "err", err)
				return err
			}
		}
	} else {
		err := utils.RemoveDir(fmt.Sprintf("dst/mods/workshop-%d", modID))
		if err != nil {
			logger.Logger.Error("删除模组失败", "err", err)
			return err
		}
	}

	return nil
}
