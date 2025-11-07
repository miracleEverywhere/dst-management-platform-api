package dst

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"github.com/yuin/gopher-lua"
	"os"
	"regexp"
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

	dmpAcfContent, err := os.ReadFile(dmpAcfPath)
	if err != nil {
		return err
	}
	gameAcfContent, err := os.ReadFile(gameAcfPath)
	if err != nil {
		return err
	}

	dmpAcfParser := NewParser(string(dmpAcfContent))
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

	for _, world := range g.worldSaveData {
		gameAcfPath = fmt.Sprintf("dst/ugc_mods/%s/%s/appworkshop_322330.acf", g.clusterName, world.WorldName)
		err = utils.EnsureDirExists(fmt.Sprintf("%s/%s", g.ugcPath, world.WorldName))
		if err != nil {
			return err
		}
		err = utils.TruncAndWriteFile(gameAcfPath, gameAcfParser.FileContent())
		if err != nil {
			return err
		}
	}

	return nil
}

type Parser struct {
	content     string
	AppWorkshop *AppWorkshop
}

func NewParser(c string) *Parser {
	p := &Parser{
		content:     c,
		AppWorkshop: &AppWorkshop{},
	}

	p.parse()

	return p
}

type AppWorkshop struct {
	AppID                  string
	SizeOnDisk             string
	NeedsUpdate            string
	NeedsDownload          string
	TimeLastUpdated        string
	TimeLastAppRan         string
	LastBuildID            string
	WorkshopItemsInstalled []ItemInstalled
	WorkshopItemDetails    []ItemDetails
}

type ItemInstalled struct {
	ID          string
	Size        string
	TimeUpdated string
	Manifest    string
}

type ItemDetails struct {
	ID                string
	Manifest          string
	TimeUpdated       string
	TimeTouched       string
	LatestTimeUpdated string
	LatestManifest    string
}

func (p *Parser) parse() {
	lines := strings.Split(p.content, "\n")
	appWorkshop := &AppWorkshop{
		WorkshopItemsInstalled: []ItemInstalled{},
		WorkshopItemDetails:    []ItemDetails{},
	}
	var currentItemID string
	var currentInstalled ItemInstalled
	var currentDetail ItemDetails
	inItemsInstalled := false
	inItemDetails := false

	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "\"WorkshopItemsInstalled\"") {
			inItemsInstalled = true
			inItemDetails = false
			continue
		}
		if strings.HasPrefix(line, "\"WorkshopItemDetails\"") {
			inItemsInstalled = false
			inItemDetails = true
			continue
		}
		if inItemsInstalled || inItemDetails {
			line = strings.ReplaceAll(line, "\"", "")
			line = strings.ReplaceAll(line, "\t", "")
			if line == "{" {
				continue
			}
			if line == "}" {
				continue
			}

			intRe := regexp.MustCompile(`^(\d+)$`)
			intReMatches := intRe.FindStringSubmatch(line)
			if intReMatches != nil {
				currentItemID = intReMatches[1]
				continue
			}
			if currentItemID != "" {
				sizeRe := regexp.MustCompile(`^size(\d+)$`)
				sizeReMatches := sizeRe.FindStringSubmatch(line)
				if sizeReMatches != nil {
					currentInstalled.Size = sizeReMatches[1]

					continue
				}

				timeupdatedRe := regexp.MustCompile(`^timeupdated(\d+)$`)
				timeupdatedReMatches := timeupdatedRe.FindStringSubmatch(line)
				if timeupdatedReMatches != nil {
					if inItemsInstalled {
						currentInstalled.TimeUpdated = timeupdatedReMatches[1]
					}
					if inItemDetails {
						currentDetail.TimeUpdated = timeupdatedReMatches[1]
					}

					continue
				}

				manifestRe := regexp.MustCompile(`^manifest(\d+)$`)
				manifestReMatches := manifestRe.FindStringSubmatch(line)
				if manifestReMatches != nil {
					if inItemsInstalled {
						currentInstalled.Manifest = manifestReMatches[1]
						currentInstalled.ID = currentItemID
						appWorkshop.WorkshopItemsInstalled = append(appWorkshop.WorkshopItemsInstalled, currentInstalled)
						currentInstalled = ItemInstalled{}
						currentItemID = ""
					}
					if inItemDetails {
						currentDetail.Manifest = manifestReMatches[1]
					}

					continue
				}

				timetouchedRe := regexp.MustCompile(`^timetouched(\d+)$`)
				timetouchedReMatches := timetouchedRe.FindStringSubmatch(line)
				if timetouchedReMatches != nil {
					currentDetail.TimeTouched = timetouchedReMatches[1]

					continue
				}

				latestTimeupdatedRe := regexp.MustCompile(`^latest_timeupdated(\d+)$`)
				latestTimeupdatedReMatches := latestTimeupdatedRe.FindStringSubmatch(line)
				if latestTimeupdatedReMatches != nil {
					currentDetail.LatestTimeUpdated = latestTimeupdatedReMatches[1]

					continue
				}

				latestManifestRe := regexp.MustCompile(`^latest_manifest(\d+)$`)
				latestManifestReMatches := latestManifestRe.FindStringSubmatch(line)
				if latestManifestReMatches != nil {
					currentDetail.LatestManifest = latestManifestReMatches[1]
					currentDetail.ID = currentItemID
					appWorkshop.WorkshopItemDetails = append(appWorkshop.WorkshopItemDetails, currentDetail)
					currentDetail = ItemDetails{}
					currentItemID = ""

					continue
				}

			}
		} else {
			line = strings.ReplaceAll(line, "\"", "")
			line = strings.ReplaceAll(line, "\t", "")

			appidRe := regexp.MustCompile(`^appid(\d+)$`)
			appidReMatches := appidRe.FindStringSubmatch(line)
			if appidReMatches != nil {
				appWorkshop.AppID = appidReMatches[1]

				continue
			}

			sizeOnDiskRe := regexp.MustCompile(`^SizeOnDisk(\d+)$`)
			sizeOnDiskReMatches := sizeOnDiskRe.FindStringSubmatch(line)
			if sizeOnDiskReMatches != nil {
				appWorkshop.SizeOnDisk = sizeOnDiskReMatches[1]

				continue
			}

			needsUpdateRe := regexp.MustCompile(`^NeedsUpdate(\d+)$`)
			needsUpdateReMatches := needsUpdateRe.FindStringSubmatch(line)
			if needsUpdateReMatches != nil {
				appWorkshop.NeedsUpdate = needsUpdateReMatches[1]

				continue
			}

			needsDownloadRe := regexp.MustCompile(`^NeedsDownload(\d+)$`)
			needsDownloadReMatches := needsDownloadRe.FindStringSubmatch(line)
			if needsDownloadReMatches != nil {
				appWorkshop.NeedsDownload = needsDownloadReMatches[1]

				continue
			}

			timeLastUpdatedRe := regexp.MustCompile(`^TimeLastUpdated(\d+)$`)
			timeLastUpdatedReMatches := timeLastUpdatedRe.FindStringSubmatch(line)
			if timeLastUpdatedReMatches != nil {
				appWorkshop.TimeLastUpdated = timeLastUpdatedReMatches[1]

				continue
			}

			timeLastAppRanRe := regexp.MustCompile(`^TimeLastAppRan(\d+)$`)
			timeLastAppRanReMatches := timeLastAppRanRe.FindStringSubmatch(line)
			if timeLastAppRanReMatches != nil {
				appWorkshop.TimeLastAppRan = timeLastAppRanReMatches[1]

				continue
			}

			lastBuildIDRe := regexp.MustCompile(`^LastBuildID(\d+)$`)
			lastBuildIDReMatches := lastBuildIDRe.FindStringSubmatch(line)
			if lastBuildIDReMatches != nil {
				appWorkshop.LastBuildID = lastBuildIDReMatches[1]

				continue
			}
		}
	}

	p.AppWorkshop = appWorkshop
}

func (p *Parser) FileContent() string {
	var (
		workshopItemsInstalled string
		workshopItemDetails    string
	)

	for _, itemInstalled := range p.AppWorkshop.WorkshopItemsInstalled {
		workshopItemsInstalled = workshopItemsInstalled + generateItemInstalled(itemInstalled)
	}

	for _, itemDetails := range p.AppWorkshop.WorkshopItemDetails {
		workshopItemDetails = workshopItemDetails + generateItemDetails(itemDetails)
	}

	content := `"AppWorkshop"
{
	"appid"		"322330"
	"SizeOnDisk"		"2071004"
	"NeedsUpdate"		"0"
	"NeedsDownload"		"0"
	"TimeLastUpdated"		"0"
	"TimeLastAppRan"		"0"
	"LastBuildID"		"0"
	"WorkshopItemsInstalled"
	{
` + workshopItemsInstalled + `
	}
	"WorkshopItemDetails"
	{
` + workshopItemDetails + `
	}
}`

	return content
}

func generateItemInstalled(i ItemInstalled) string {
	return `		"` + i.ID + `"
		{
			"size"		"` + i.Size + `"
			"timeupdated"		"` + i.TimeUpdated + `"
			"manifest"		"` + i.Manifest + `"
		}
`
}

func generateItemDetails(i ItemDetails) string {
	return `		"` + i.ID + `"
		{
			"manifest"		"` + i.Manifest + `"
			"timeupdated"		"` + i.TimeUpdated + `"
			"timetouched"		"` + i.TimeTouched + `"
			"latest_timeupdated"		"` + i.LatestTimeUpdated + `"
			"latest_manifest"		"` + i.LatestManifest + `"
		}
`
}
