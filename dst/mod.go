package dst

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"github.com/yuin/gopher-lua"
	"os"
	"strconv"
	"strings"
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
		tbl.ForEach(func(key lua.LValue, value lua.LValue) {
			// 检查键是否是字符串，并且以 "workshop-" 开头
			if strKey, ok := key.(lua.LString); ok && strings.HasPrefix(string(strKey), "workshop-") {
				// 提取 "workshop-" 后面的数字
				workshopID := strings.TrimPrefix(string(strKey), "workshop-")
				fileContent = fileContent + "ServerModSetup(\"" + workshopID + "\")\n"
			}
		})
		err := utils.TruncAndWriteFile(utils.GameModSettingPath, fileContent)
		if err != nil {
			return err
		}
	}

	return nil
}

func (g *Game) downloadMod(id int, ugc bool) error {
	var err error

	if ugc {
		// 1. ugc mod 统一下载到 dmp_files/ugc, 也就是dmp_files/ugc/{cluster}/steamapps/workshop{appworkshop_322330.acf  content  downloads}
		// 2. 下载完成后，将下载的mod文件全部移动至dst/ugc_mods/{cluster}/{worlds}/ 删除-复制
		// 3. 读取游戏acf文件和dmp_files的acf文件，更新当前mod-id所对应的所有字段
		// 4. 清理下载的临时文件，rm -rf dmp_files/ugc/{cluster}/*

		// 1
		downloadCmd := g.generateModDownloadCmd(id)
		logger.Logger.Debug(downloadCmd)
		err = utils.BashCMD(downloadCmd)
		if err != nil {
			logger.Logger.Error("下载模组失败", "err", err)
			return err
		}

		// 2
		err = g.removeGameOldMod(id)
		if err != nil {
			logger.Logger.Warn("移动模组失败", "err", err)
			return err
		}
		copyCmd := g.generateModCopyCmd(id)
		logger.Logger.Debug(copyCmd)
		err = utils.BashCMD(copyCmd)
		if err != nil {
			logger.Logger.Warn("移动模组失败", "err", err)
			return err
		}

		// 3
		err = g.processAcf(id)
		if err != nil {
			logger.Logger.Error("修改acf文件失败", "err", err)
			return err
		}

		// 4
		err = utils.RemoveDir(fmt.Sprintf("dmp_files/mods/ugc/%s", g.clusterName))
		if err != nil {
			logger.Logger.Warn("删除临时模组失败", "err", err)
		}

		return nil
	} else {
		return nil
	}
}

func (g *Game) generateModDownloadCmd(id int) string {
	return fmt.Sprintf("steamcmd/steamcmd.sh +force_install_dir %s/dmp_files/mods/ugc/%s +login anonymous +workshop_download_item 322330 %d +quit", db.CurrentDir, g.clusterName, id)
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

	dmpPath := fmt.Sprintf("dmp_files/mods/ugc/%s/steamapps/workshop/content/322330/%d", g.clusterName, id)

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

	dmpAcfPath := fmt.Sprintf("dmp_files/mods/ugc/%s/steamapps/workshop/appworkshop_322330.acf", g.clusterName)
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

	dmpAcfParser := NewParser(string(dmpAcfContent))

	var writtenContent string

	if len(gameAcfContent) == 0 {
		// 如果游戏mod目录没有acf文件，直接使用dmp下载的acf文件
		writtenContent = dmpAcfParser.FileContent()
	} else {
		// 如果游戏mod目录含有acf文件，处理游戏acf文件
		gameAcfParser := NewParser(string(gameAcfContent))
		var (
			gameAcfTargetIndex int
			hasMod             bool
		)
		for index, i := range gameAcfParser.AppWorkshop.WorkshopItemsInstalled {
			if i.ID == acfID {
				gameAcfTargetIndex = index
				hasMod = true
			}
		}
		if hasMod {
			gameAcfParser.AppWorkshop.WorkshopItemsInstalled[gameAcfTargetIndex] = dmpAcfParser.AppWorkshop.WorkshopItemsInstalled[0]
			gameAcfParser.AppWorkshop.WorkshopItemDetails[gameAcfTargetIndex] = dmpAcfParser.AppWorkshop.WorkshopItemDetails[0]
		} else {
			gameAcfParser.AppWorkshop.WorkshopItemsInstalled = append(gameAcfParser.AppWorkshop.WorkshopItemsInstalled, dmpAcfParser.AppWorkshop.WorkshopItemsInstalled[0])
			gameAcfParser.AppWorkshop.WorkshopItemDetails = append(gameAcfParser.AppWorkshop.WorkshopItemDetails, dmpAcfParser.AppWorkshop.WorkshopItemDetails[0])
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
	gameAcfPath := fmt.Sprintf("dst/ugc_mods/%s/%s/appworkshop_322330.acf", g.clusterName, g.worldSaveData[0].WorldName)
	err := utils.EnsureFileExists(gameAcfPath)
	if err != nil {
		logger.Logger.Error("EnsureFileExists失败", "path", gameAcfPath)
		return &[]DownloadedMod{}
	}

	gameAcfContent, err := os.ReadFile(gameAcfPath)
	if err != nil {
		return &[]DownloadedMod{}
	}

	if len(gameAcfContent) == 0 {
		return &[]DownloadedMod{}
	}

	var downloadedMods []DownloadedMod
	gameAcfParser := NewParser(string(gameAcfContent))
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

	return &downloadedMods
}
