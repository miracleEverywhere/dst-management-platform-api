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

func (g *Game) downloadMod(id int, ugc bool) {
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
		}

		// 2
		mvCmd := g.generateModMoveCmd(id)
		logger.Logger.Debug(mvCmd)
		err = utils.BashCMD(mvCmd)
		if err != nil {
			logger.Logger.Error("移动模组失败", "err", err)
		}

		// 3
		err = g.processAcf(id)
		if err != nil {
			logger.Logger.Error("修改acf文件失败", "err", err)
		}

		// 4
		rmCmd := fmt.Sprintf("rm -rf dmp_files/ugc/%s/*", g.clusterName)
		logger.Logger.Debug(rmCmd)
		err = utils.BashCMD(rmCmd)
		if err != nil {
			logger.Logger.Warn("删除临时模组失败", "err", err)
		}

	} else {

	}
}

// AppWorkshop 定义acf文件结构体
type AppWorkshop struct {
	AppID                  string                   `json:"appid"`
	SizeOnDisk             string                   `json:"SizeOnDisk"`
	NeedsUpdate            string                   `json:"NeedsUpdate"`
	NeedsDownload          string                   `json:"NeedsDownload"`
	TimeLastUpdated        string                   `json:"TimeLastUpdated"`
	TimeLastAppRan         string                   `json:"TimeLastAppRan"`
	LastBuildID            string                   `json:"LastBuildID"`
	WorkshopItemsInstalled map[string]ItemInstalled `json:"WorkshopItemsInstalled"`
	WorkshopItemDetails    map[string]ItemDetails   `json:"WorkshopItemDetails"`
}

type ItemInstalled struct {
	Size        string `json:"size"`
	TimeUpdated string `json:"timeupdated"`
}

type ItemDetails struct {
	Manifest          string `json:"manifest"`
	TimeUpdated       string `json:"timeupdated"`
	TimeTouched       string `json:"timetouched"`
	LatestTimeUpdated string `json:"latest_timeupdated"`
	LatestManifest    string `json:"latest_manifest"`
}

// 解析ACF文件内容
func parseACFFile(content string) (*AppWorkshop, error) {
	lines := strings.Split(content, "\n")
	appWorkshop := &AppWorkshop{
		WorkshopItemsInstalled: make(map[string]ItemInstalled),
		WorkshopItemDetails:    make(map[string]ItemDetails),
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
		} else if strings.HasPrefix(line, "\"WorkshopItemDetails\"") {
			inItemsInstalled = false
			inItemDetails = true
			continue
		} else if strings.HasPrefix(line, "}") && !strings.Contains(line, "{") {
			if inItemsInstalled || inItemDetails {
				// 结束当前项目
				if currentItemID != "" && inItemsInstalled {
					appWorkshop.WorkshopItemsInstalled[currentItemID] = currentInstalled
					currentInstalled = ItemInstalled{}
					currentItemID = ""
				} else if currentItemID != "" && inItemDetails {
					appWorkshop.WorkshopItemDetails[currentItemID] = currentDetail
					currentDetail = ItemDetails{}
					currentItemID = ""
				}
			}
			if line == "}" {
				inItemsInstalled = false
				inItemDetails = false
			}
			continue
		}

		if inItemsInstalled || inItemDetails {
			// 解析项目ID
			if strings.HasPrefix(line, "\"") && strings.HasSuffix(line, "{") {
				itemID := strings.Trim(line, "\"{")
				currentItemID = strings.TrimSpace(itemID)
				continue
			}

			// 解析项目字段
			if currentItemID != "" {
				parts := strings.SplitN(line, "\"", 4)
				if len(parts) >= 4 {
					key := strings.TrimSpace(parts[1])
					value := strings.TrimSpace(strings.Trim(parts[3], "\""))

					if inItemsInstalled {
						switch key {
						case "size":
							currentInstalled.Size = value
						case "timeupdated":
							currentInstalled.TimeUpdated = value
						}
					} else if inItemDetails {
						switch key {
						case "manifest":
							currentDetail.Manifest = value
						case "timeupdated":
							currentDetail.TimeUpdated = value
						case "timetouched":
							currentDetail.TimeTouched = value
						case "latest_timeupdated":
							currentDetail.LatestTimeUpdated = value
						case "latest_manifest":
							currentDetail.LatestManifest = value
						}
					}
				}
			}
		} else {
			// 解析基本字段
			if strings.Contains(line, "\"") {
				parts := strings.SplitN(line, "\"", 5)
				if len(parts) >= 5 {
					key := strings.TrimSpace(parts[1])
					value := strings.TrimSpace(strings.Trim(parts[3], "\""))

					switch key {
					case "appid":
						appWorkshop.AppID = value
					case "SizeOnDisk":
						appWorkshop.SizeOnDisk = value
					case "NeedsUpdate":
						appWorkshop.NeedsUpdate = value
					case "NeedsDownload":
						appWorkshop.NeedsDownload = value
					case "TimeLastUpdated":
						appWorkshop.TimeLastUpdated = value
					case "TimeLastAppRan":
						appWorkshop.TimeLastAppRan = value
					case "LastBuildID":
						appWorkshop.LastBuildID = value
					}
				}
			}
		}
	}

	return appWorkshop, nil
}

//// 生成ACF文件内容
//func generateACFContent(appWorkshop *AppWorkshop) (string, error) {
//	var buffer bytes.Buffer
//
//	// 写入基本字段
//	writeSectionHeader(&buffer, "AppWorkshop", 0)
//
//	fields := []struct {
//		key   string
//		value string
//	}{
//		{"appid", appWorkshop.AppID},
//		{"SizeOnDisk", appWorkshop.SizeOnDisk},
//		{"NeedsUpdate", appWorkshop.NeedsUpdate},
//		{"NeedsDownload", appWorkshop.NeedsDownload},
//		{"TimeLastUpdated", appWorkshop.TimeLastUpdated},
//		{"TimeLastAppRan", appWorkshop.TimeLastAppRan},
//		{"LastBuildID", appWorkshop.LastBuildID},
//	}
//
//	for _, field := range fields {
//		if field.value != "" {
//			writeKeyValue(&buffer, field.key, field.value, 1)
//		}
//	}
//
//	// 写入WorkshopItemsInstalled
//	if len(appWorkshop.WorkshopItemsInstalled) > 0 {
//		writeSectionHeader(&buffer, "WorkshopItemsInstalled", 1)
//
//		itemIDs := getSortedKeys(appWorkshop.WorkshopItemsInstalled)
//		for _, id := range itemIDs {
//			item := appWorkshop.WorkshopItemsInstalled[id]
//			writeSectionHeader(&buffer, id, 2)
//
//			writeKeyValue(&buffer, "size", item.Size, 3)
//			writeKeyValue(&buffer, "timeupdated", item.TimeUpdated, 3)
//			writeKeyValue(&buffer, "manifest", item.Manifest, 3)
//
//			writeSectionFooter(&buffer, 2)
//		}
//
//		writeSectionFooter(&buffer, 1)
//	}
//
//	// 写入WorkshopItemDetails
//	if len(appWorkshop.WorkshopItemDetails) > 0 {
//		writeSectionHeader(&buffer, "WorkshopItemDetails", 1)
//
//		detailIDs := getSortedKeys(appWorkshop.WorkshopItemDetails)
//		for _, id := range detailIDs {
//			detail := appWorkshop.WorkshopItemDetails[id]
//			writeSectionHeader(&buffer, id, 2)
//
//			writeKeyValue(&buffer, "manifest", detail.Manifest, 3)
//			writeKeyValue(&buffer, "timeupdated", detail.TimeUpdated, 3)
//			writeKeyValue(&buffer, "timetouched", detail.TimeTouched, 3)
//			writeKeyValue(&buffer, "latest_timeupdated", detail.LatestTimeUpdated, 3)
//			writeKeyValue(&buffer, "latest_manifest", detail.LatestManifest, 3)
//
//			writeSectionFooter(&buffer, 2)
//		}
//
//		writeSectionFooter(&buffer, 1)
//	}
//
//	writeSectionFooter(&buffer, 0)
//
//	return buffer.String(), nil
//}
//
//// 写入节头部
//func writeSectionHeader(buffer *bytes.Buffer, name string, indentLevel int) {
//	indent := strings.Repeat("\t", indentLevel)
//	buffer.WriteString(fmt.Sprintf("%s\"%s\"\n", indent, name))
//	buffer.WriteString(fmt.Sprintf("%s{\n", indent))
//}
//
//// 写入节尾部
//func writeSectionFooter(buffer *bytes.Buffer, indentLevel int) {
//	indent := strings.Repeat("\t", indentLevel)
//	buffer.WriteString(fmt.Sprintf("%s}\n", indent))
//}
//
//// 键值对写入
//func writeKeyValue(buffer *bytes.Buffer, key, value string, indentLevel int) {
//	if value == "" {
//		return
//	}
//
//	indent := strings.Repeat("\t", indentLevel)
//
//	// 判断值类型
//	var formattedValue string
//	if isNumeric(value) {
//		formattedValue = value
//	} else {
//		formattedValue = fmt.Sprintf("\"%s\"", value)
//	}
//
//	buffer.WriteString(fmt.Sprintf("%s\"%s\"\t\t%s\n", indent, key, formattedValue))
//}
//
//// 判断字符串是否为数字
//func isNumeric(s string) bool {
//	_, err := strconv.ParseInt(s, 10, 64)
//	return err == nil
//}
//
//// 获取排序的键
//func getSortedKeys(m interface{}) []string {
//	var keys []string
//
//	switch m := m.(type) {
//	case map[string]WorkshopItem:
//		keys = make([]string, 0, len(m))
//		for k := range m {
//			keys = append(keys, k)
//		}
//	case map[string]ItemDetails:
//		keys = make([]string, 0, len(m))
//		for k := range m {
//			keys = append(keys, k)
//		}
//	default:
//		return nil
//	}
//
//	sort.Strings(keys)
//	return keys
//}
//
//// ACF写入函数
//func writeACFFile(appWorkshop *AppWorkshop, filename string) error {
//	content, err := generateACFContent(appWorkshop)
//	if err != nil {
//		return err
//	}
//
//	return os.WriteFile(filename, []byte(content), 0644)
//}

func (g *Game) generateModDownloadCmd(id int) string {
	return fmt.Sprintf("steamcmd/steamcmd.sh +force_install_dir %s/dmp_files/mods/ugc/%s +login anonymous +workshop_download_item 322330 %d +quit", db.CurrentDir, g.clusterName, id)
}

func (g *Game) generateModMoveCmd(id int) string {
	if len(g.worldSaveData) == 0 {
		return ""
	}

	dmpPath := fmt.Sprintf("dmp_files/mods/ugc/%s/steamapps/workshop/content/322330/%d", g.clusterName, id)

	var cmds []string

	// 命令有执行顺序，使用多个循环生成
	// 生成 删除 命令
	for _, world := range g.worldSaveData {
		cmd := fmt.Sprintf("rm -rf dst/ugc_mods/%s/%s/content/322330/%d", g.clusterName, world.WorldName, id)
		cmds = append(cmds, cmd)
	}
	// 生成 复制 命令
	for _, world := range g.worldSaveData {
		gamePath := fmt.Sprintf("dst/ugc_mods/%s/%s/content/322330/%d", g.clusterName, world.WorldName, id)
		cmd := fmt.Sprintf("cp -r %s %s", dmpPath, gamePath)
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

	dmpAcf, err := parseACFFile(string(dmpAcfContent))
	logger.Logger.Debug(utils.StructToFlatString(dmpAcf))
	gameAcf, err := parseACFFile(string(gameAcfContent))
	logger.Logger.Debug(utils.StructToFlatString(gameAcf))

	gameAcf.WorkshopItemsInstalled[acfID] = dmpAcf.WorkshopItemsInstalled[acfID]

	//err = writeACFFile(gameAcf, gameAcfPath)

	return err
}
