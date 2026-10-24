package platform

import (
	"bufio"
	"dst-management-platform-api/aichat"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/process"
	lua "github.com/yuin/gopher-lua"
)

type Handler struct {
	userDao          *dao.UserDAO
	roomDao          *dao.RoomDAO
	worldDao         *dao.WorldDAO
	systemDao        *dao.SystemDAO
	globalSettingDao *dao.GlobalSettingDAO
	uidMapDao        *dao.UidMapDAO
	roomSettingDao   *dao.RoomSettingDAO
	pluginDao        *dao.PluginDAO
	dstImageDao      *dao.DstImageDAO
	aiManager        *aichat.Manager
}

func NewHandler(userDao *dao.UserDAO, roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, systemDao *dao.SystemDAO, globalSettingDao *dao.GlobalSettingDAO, uidMapDao *dao.UidMapDAO, roomSettingDao *dao.RoomSettingDAO, pluginDao *dao.PluginDAO, dstImageDao *dao.DstImageDAO, aiManager *aichat.Manager) *Handler {
	return &Handler{
		userDao:          userDao,
		roomDao:          roomDao,
		worldDao:         worldDao,
		systemDao:        systemDao,
		globalSettingDao: globalSettingDao,
		uidMapDao:        uidMapDao,
		roomSettingDao:   roomSettingDao,
		pluginDao:        pluginDao,
		dstImageDao:      dstImageDao,
		aiManager:        aiManager,
	}
}

func getRES() uint64 {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return 0
	}

	memoryInfo, err := p.MemoryInfo()
	if err != nil {
		return 0
	}

	return memoryInfo.RSS
}

type OSInfo struct {
	Architecture    string
	OS              string
	CPUModel        string
	CPUCores        int
	CPUDetail       []cpu.InfoStat
	MemorySize      uint64
	Platform        string
	PlatformVersion string
	Uptime          uint64
}

func getOSInfo() (*OSInfo, error) {
	architecture := runtime.GOARCH

	// 获取CPU信息
	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, err
	}
	cpuModel := cpuInfo[0].ModelName
	cpuCount, _ := cpu.Counts(true)
	cpuCore := cpuCount
	cpuDetail := cpuInfo

	// 获取内存信息
	virtualMemory, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	memorySize := virtualMemory.Total

	// 获取主机信息
	hostInfo, err := host.Info()
	if err != nil {
		return nil, err
	}
	platformVersion := hostInfo.PlatformVersion
	platform := hostInfo.Platform
	uptime := hostInfo.Uptime
	osName := hostInfo.OS
	// 返回系统信息
	return &OSInfo{
		Architecture:    architecture,
		OS:              osName,
		CPUModel:        cpuModel,
		CPUCores:        cpuCore,
		CPUDetail:       cpuDetail,
		MemorySize:      memorySize,
		Platform:        platform,
		Uptime:          uptime,
		PlatformVersion: platformVersion,
	}, nil
}

type Partition struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"pageSize" form:"pageSize"`
}

// stexDir 是旧版本下载的 Stex 二进制目录，现在改用内置解包，仅保留用于升级清理
const stexDir = constant.PluginTmiPath + "/stex"
const tmirID = constant.TmirID
const gameImagesPath = constant.PluginTmiPath + "/dst_images"
const aiChatWikiURL = "https://raw.githubusercontent.com/miracleEverywhere/dst-management-platform-api/master/aichat/wiki/wiki.zip"

var tmirPath = fmt.Sprintf("%s/tmir/steamapps/workshop/content/322330/%d/scripts/TMIR/itemlist/lists", constant.PluginTmiPath, constant.TmirID)

func installAiChatWiki(proxy string) error {
	downloadURL := aiChatWikiURL
	if proxy != "" {
		if proxy[len(proxy)-1] != '/' {
			proxy += "/"
		}
		downloadURL = proxy + aiChatWikiURL
	}

	if err := utils.EnsureDirExists(constant.PluginAiChatPath); err != nil {
		return fmt.Errorf("创建 AI Chat 插件目录失败: %w", err)
	}
	temporaryDir, err := os.MkdirTemp(constant.PluginAiChatPath, ".wiki-install-")
	if err != nil {
		return fmt.Errorf("创建 Wiki 临时目录失败: %w", err)
	}
	defer os.RemoveAll(temporaryDir)

	client := &http.Client{Timeout: constant.HttpTimeout * time.Second}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("下载 Wiki 压缩包失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("下载 Wiki 压缩包失败，HTTP 状态码: %s", resp.Status)
	}

	zipPath := filepath.Join(temporaryDir, "wiki.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("创建 Wiki 压缩包失败: %w", err)
	}
	_, copyErr := io.Copy(zipFile, resp.Body)
	closeErr := zipFile.Close()
	if copyErr != nil {
		return fmt.Errorf("保存 Wiki 压缩包失败: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("关闭 Wiki 压缩包失败: %w", closeErr)
	}

	temporaryWikiDir := filepath.Join(temporaryDir, "wiki")
	if err := utils.Unzip(zipPath, temporaryWikiDir); err != nil {
		return fmt.Errorf("解压 Wiki 压缩包失败: %w", err)
	}
	if err := os.Remove(zipPath); err != nil {
		return fmt.Errorf("清理 Wiki 压缩包失败: %w", err)
	}

	if err := utils.RemoveDir(constant.PluginAiChatWikiPath); err != nil {
		return fmt.Errorf("清理旧 Wiki 目录失败: %w", err)
	}
	if err := os.Rename(temporaryWikiDir, constant.PluginAiChatWikiPath); err != nil {
		return fmt.Errorf("替换 Wiki 目录失败: %w", err)
	}
	return nil
}

// 初始化Tmi工具，仅支持ubuntu24及以上
func initTmi(step int) (int, []models.DstImage, error) {
	steps := []struct {
		fn   func(string) ([]models.DstImage, error)
		code int
	}{
		{installDependency, 1},
		{removeStex, 2},
		{unzipDstImages, 3},
		{texToPng, 4},
		{installTMIR, 5},
	}

	startIndex := 0
	for i, s := range steps {
		if s.code == step {
			startIndex = i
			break
		}
	}

	var (
		err    error
		images []models.DstImage
	)

	for i := startIndex; i < len(steps); i++ {
		images, err = steps[i].fn("")
		if err != nil {
			return steps[i].code, []models.DstImage{}, err
		}
	}

	return 100, images, nil
}

// 安装依赖，使用内置 TEX 解包后只依赖 unzip
func installDependency(string) ([]models.DstImage, error) {
	var images []models.DstImage

	cmd := exec.Command("apt", "install", "-y", "unzip")
	if _, err := cmd.CombinedOutput(); err != nil {
		logger.Logger.Errorf("安装依赖失败: %v", err)
		return images, fmt.Errorf("安装依赖失败: %v", err)
	}

	return images, nil
}

// 旧版本会下载安装 Stex 二进制，现在改用内置 TEX 解包，这里清理遗留的安装目录
func removeStex(string) ([]models.DstImage, error) {
	var images []models.DstImage

	_ = utils.RemoveDir(stexDir)

	return images, nil
}

// 解压游戏tex文件
func unzipDstImages(string) ([]models.DstImage, error) {
	var images []models.DstImage

	// 幂等
	_ = utils.RemoveDir(gameImagesPath)

	_ = utils.EnsureDirExists(gameImagesPath)

	var dstDataPath string
	if cache.OsType == constant.Darwin {
		dstDataPath = constant.DarwinDstContentsPath + "/data"
	} else {
		dstDataPath = constant.GameMainPath + "/data"
	}

	cmdArgs := []string{
		dstDataPath + "/databundles/images.zip",
		"-d",
		gameImagesPath,
	}
	cmd := exec.Command("unzip", cmdArgs...)
	if _, err := cmd.CombinedOutput(); err != nil {
		_ = utils.RemoveDir(gameImagesPath)
		logger.Logger.Errorf("解压游戏tex失败: %v", err)
		return images, fmt.Errorf("解压游戏tex失败: %v", err)
	}

	return images, nil
}

// 游戏tex文件转为png文件，使用内置 TEX 解包
func texToPng(string) ([]models.DstImage, error) {
	var images []models.DstImage

	inventoryimages := []string{
		"inventoryimages1",
		"inventoryimages2",
		"inventoryimages3",
		"inventoryimages4",
	}
	for _, inventoryimage := range inventoryimages {
		xmlPath := filepath.Join(gameImagesPath, "images", inventoryimage+".xml")
		outputDir, _, err := utils.Unpack(xmlPath, gameImagesPath)
		if err != nil {
			cleanTexOutput(inventoryimages)
			logger.Logger.Errorf("游戏tex: %s 转png失败: %v", inventoryimage, err)
			return images, fmt.Errorf("游戏tex: %s 转png失败: %v", inventoryimage, err)
		}
		if err := movePngFiles(outputDir, gameImagesPath); err != nil {
			cleanTexOutput(inventoryimages)
			logger.Logger.Errorf("整理png文件失败: %v", err)
			return images, fmt.Errorf("整理png文件失败: %v", err)
		}
		_ = utils.RemoveDir(outputDir)
	}
	_ = utils.RemoveDir(filepath.Join(gameImagesPath, "images"))

	return images, nil
}

// movePngFiles 把解包目录里的 png 移到图集根目录，前端按 <prefab>.png 读取
func movePngFiles(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".png") {
			continue
		}
		if err := os.Rename(filepath.Join(srcDir, entry.Name()), filepath.Join(dstDir, entry.Name())); err != nil {
			return err
		}
	}

	return nil
}

// cleanTexOutput 清理本次生成的部分产物，避免失败后留下残缺数据
func cleanTexOutput(inventoryimages []string) {
	entries, err := os.ReadDir(gameImagesPath)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".png") {
			continue
		}
		_ = os.Remove(filepath.Join(gameImagesPath, entry.Name()))
	}

	for _, inventoryimage := range inventoryimages {
		_ = utils.RemoveDir(filepath.Join(gameImagesPath, inventoryimage))
	}
}

// 安装TMIR 与 获取 翻译文件
func installTMIR(string) ([]models.DstImage, error) {
	var images []models.DstImage

	modDir := filepath.Join(cache.CurrentDir, constant.PluginTmiPath, "tmir")
	cmdArgs := []string{
		"+force_install_dir",
		modDir,
		"+login",
		"anonymous",
		"+workshop_download_item",
		"322330",
		fmt.Sprintf("%d", tmirID),
		"+quit",
	}
	cmd := exec.Command("steamcmd/steamcmd.sh", cmdArgs...)
	if _, err := cmd.CombinedOutput(); err != nil {
		_ = utils.RemoveDir(modDir)
		logger.Logger.Errorf("下载TMIR模组失败: %v", err)
		return images, fmt.Errorf("下载TMIR模组失败: %v", err)
	}

	var dstDataPath string
	if cache.OsType == constant.Darwin {
		dstDataPath = constant.DarwinDstContentsPath + "/data"
	} else {
		dstDataPath = constant.GameMainPath + "/data"
	}

	scriptPath := filepath.Join(dstDataPath, "databundles", "scripts.zip")
	tmpDir := filepath.Join(constant.PluginTmiPath, "tmp")
	gameChinesePo := filepath.Join(tmpDir, "scripts", "languages", "chinese_s.po")

	_ = utils.RemoveDir(tmpDir)

	cmdArgs = []string{
		scriptPath,
		"-d",
		tmpDir,
	}
	cmd = exec.Command("unzip", cmdArgs...)
	if _, err := cmd.CombinedOutput(); err != nil {
		logger.Logger.Errorf("解压游戏scripts失败: %v", err)
		return images, fmt.Errorf("解压游戏scripts失败: %v", err)
	}

	// 解析 PO 文件，构建 prefab -> {en, zh} 映射
	translations, err := parseChinesePo(gameChinesePo)
	if err != nil {
		logger.Logger.Errorf("解析PO文件失败: %v", err)
		return images, fmt.Errorf("解析PO文件失败: %v", err)
	}

	// 构建数据库数据
	items, err := tmiItemLists()
	if err != nil {
		logger.Logger.Errorf("获取tmir模组信息失败: %v", err)
		return images, fmt.Errorf("获取tmir模组信息失败: %v", err)
	}
	for _, category := range items {
		for _, prefab := range category.Items {
			nameEn, nameZh := "", ""
			if entry, ok := translations[strings.ToUpper(prefab)]; ok {
				nameEn = entry.MsgID
				nameZh = entry.MsgStr
			}
			images = append(images, models.DstImage{
				Prefab:   prefab,
				Category: category.Category,
				NameEn:   nameEn,
				NameZh:   nameZh,
			})
		}
	}

	// 删除临时目录
	_ = utils.RemoveDir(tmpDir)

	return images, nil
}

type poEntry struct {
	MsgID  string
	MsgStr string
}

// parseChinesePo 解析中文 PO 文件，返回大写 prefab -> {英文名, 中文名} 的映射
// 只解析 msgctxt 以 STRINGS.NAMES. 开头的条目
func parseChinesePo(filePath string) (map[string]poEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开PO文件失败: %v", err)
	}
	defer file.Close()

	result := make(map[string]poEntry)
	scanner := bufio.NewScanner(file)

	var currentCtxt, currentMsgID, currentMsgStr string

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "msgctxt "):
			// 遇到新条目，保存上一个（如果有）
			if currentCtxt != "" && strings.HasPrefix(currentCtxt, "STRINGS.NAMES.") {
				prefab := strings.TrimPrefix(currentCtxt, "STRINGS.NAMES.")
				result[prefab] = poEntry{MsgID: currentMsgID, MsgStr: currentMsgStr}
			}
			currentCtxt = parsePoValue(line)
			currentMsgID = ""
			currentMsgStr = ""
		case strings.HasPrefix(line, "msgid "):
			currentMsgID = parsePoValue(line)
		case strings.HasPrefix(line, "msgstr "):
			currentMsgStr = parsePoValue(line)
		}
	}

	// 保存最后一个条目
	if currentCtxt != "" && strings.HasPrefix(currentCtxt, "STRINGS.NAMES.") {
		prefab := strings.TrimPrefix(currentCtxt, "STRINGS.NAMES.")
		result[prefab] = poEntry{MsgID: currentMsgID, MsgStr: currentMsgStr}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取PO文件失败: %v", err)
	}

	return result, nil
}

// parsePoValue 解析 PO 文件中单行带引号的值，如 "Birchnut" -> Birchnut
func parsePoValue(line string) string {
	start := strings.Index(line, "\"")
	if start == -1 {
		return ""
	}
	end := strings.LastIndex(line, "\"")
	if end <= start {
		return ""
	}
	return line[start+1 : end]
}

type itemCategory struct {
	Category string   `json:"category"` // 分类名 (如 tool, food, equip)
	Items    []string `json:"items"`    // 该分类下的物品代码列表
}

// tmiItemLists 解析 mod 的 Lua 物品列表文件，返回按分类组织的物品代码
func tmiItemLists() ([]itemCategory, error) {
	entries, err := os.ReadDir(tmirPath)
	if err != nil {
		return nil, fmt.Errorf("读取物品列表目录失败: %v", err)
	}

	var categories []itemCategory

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "itemlist_") || !strings.HasSuffix(entry.Name(), ".lua") {
			continue
		}

		// 从文件名提取分类名: itemlist_tool.lua → tool
		category := strings.TrimPrefix(entry.Name(), "itemlist_")
		category = strings.TrimSuffix(category, ".lua")

		filePath := filepath.Join(tmirPath, entry.Name())
		items, err := parseLuaItemList(filePath)
		if err != nil {
			return nil, fmt.Errorf("解析 %s 失败: %v", entry.Name(), err)
		}

		categories = append(categories, itemCategory{
			Category: category,
			Items:    items,
		})
	}

	// 按分类名排序，保证输出稳定
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Category < categories[j].Category
	})

	return categories, nil
}

// parseLuaItemList 解析单个 Lua 物品列表文件，格式为 return{"item1","item2",...}
func parseLuaItemList(filePath string) ([]string, error) {
	L := utils.NewSafeLuaState()
	defer L.Close()

	if err := L.DoFile(filePath); err != nil {
		return nil, fmt.Errorf("执行 Lua 文件失败: %v", err)
	}

	// 获取返回值 (Lua 栈顶)
	ret := L.Get(-1)
	if ret.Type() != lua.LTTable {
		return nil, fmt.Errorf("返回值不是 table 类型: %s", ret.Type().String())
	}

	table := ret.(*lua.LTable)
	var items []string

	table.ForEach(func(_, value lua.LValue) {
		if value.Type() == lua.LTString {
			items = append(items, value.String())
		}
	})

	return items, nil
}
