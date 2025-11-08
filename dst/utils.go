package dst

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/utils"
	"fmt"
	"regexp"
	"strings"
	"sync"
)

type Game struct {
	room    *models.Room
	worlds  *[]models.World
	setting *models.RoomSetting
	lang    string
	roomSaveData
	worldSaveData []worldSaveData
	playerSaveData
	modSaveData
	// room全局文件锁
	roomMutex sync.Mutex
	// world全局文件锁
	worldMutex sync.Mutex
	// player全局文件锁
	playerMutex sync.Mutex
	// acf文件锁
	acfMutex sync.Mutex
}

func NewGameController(room *models.Room, worlds *[]models.World, setting *models.RoomSetting, lang string) *Game {
	game := &Game{
		room:    room,
		worlds:  worlds,
		setting: setting,
		lang:    lang,
	}

	game.initInfo()

	return game
}

func (g *Game) initInfo() {
	// room
	g.clusterName = fmt.Sprintf("Cluster_%d", g.room.ID)
	g.clusterPath = fmt.Sprintf("%s/%s", utils.ClusterPath, g.clusterName)
	g.clusterIniPath = fmt.Sprintf("%s/cluster.ini", g.clusterPath)
	g.clusterTokenTxtPath = fmt.Sprintf("%s/cluster_token.txt", g.clusterPath)

	// worlds
	for _, world := range *g.worlds {
		worldPath := fmt.Sprintf("%s/%s", g.clusterPath, world.WorldName)
		serverIniPath := fmt.Sprintf("%s/server.ini", worldPath)
		levelDataOverridePath := fmt.Sprintf("%s/leveldataoverride.lua", worldPath)
		modOverridesPath := fmt.Sprintf("%s/modoverrides.lua", worldPath)
		screenName := fmt.Sprintf("DMP_%s_%s", g.clusterName, world.WorldName)

		var startCmd string
		switch g.setting.StartType {
		case "32-bit":
			startCmd = fmt.Sprintf("cd dst/bin/ && screen -d -h 200 -m -S %s ./dontstarve_dedicated_server_nullrenderer -console -cluster %s -shard %s", screenName, g.clusterName, world.WorldName)
		case "64-bit":
			startCmd = fmt.Sprintf("cd dst/bin64/ && screen -d -h 200 -m -S %s ./dontstarve_dedicated_server_nullrenderer_x64 -console -cluster %s -shard %s", screenName, g.clusterName, world.WorldName)
		default:
			startCmd = "exit 1"
		}

		g.worldSaveData = append(g.worldSaveData, worldSaveData{
			worldPath:             worldPath,
			serverIniPath:         serverIniPath,
			levelDataOverridePath: levelDataOverridePath,
			modOverridesPath:      modOverridesPath,
			startCmd:              startCmd,
			screenName:            screenName,
			World:                 world,
		})
	}

	// players
	g.adminlistPath = fmt.Sprintf("%s/adminlist.txt", g.clusterPath)
	g.whitelistPath = fmt.Sprintf("%s/whitelist.txt", g.clusterPath)
	g.blocklistPath = fmt.Sprintf("%s/blocklist.txt", g.clusterPath)
	g.adminlist = getPlayerList(g.adminlistPath)
	g.whitelist = getPlayerList(g.whitelistPath)
	g.blocklist = getPlayerList(g.blocklistPath)

	// mods
	g.ugcPath = fmt.Sprintf("%s/dst/ugc_mods/%s", db.CurrentDir, g.clusterName)
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
