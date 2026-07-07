package platform

import (
	"bufio"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
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
}

func NewHandler(userDao *dao.UserDAO, roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, systemDao *dao.SystemDAO, globalSettingDao *dao.GlobalSettingDAO, uidMapDao *dao.UidMapDAO, roomSettingDao *dao.RoomSettingDAO, pluginDao *dao.PluginDAO, dstImageDao *dao.DstImageDAO) *Handler {
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

const stexDir = utils.PluginTmiPath + "/stex"
const StexBin = stexDir + "/bin/stex"
const tmirID = utils.TmirID
const gameImagesPath = utils.PluginTmiPath + "/dst_images"

var tmirPath = fmt.Sprintf("%s/tmir/steamapps/workshop/content/322330/%d/scripts/TMIR/itemlist/lists", utils.PluginTmiPath, utils.TmirID)

// 初始化Tmi工具，仅支持ubuntu24及以上
func initTmi(proxy string, step int) (int, []models.DstImage, error) {
	steps := []struct {
		fn   func(string) ([]models.DstImage, error)
		code int
	}{
		{installDependency, 1},
		{installStex, 2},
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
		images, err = steps[i].fn(proxy)
		if err != nil {
			return steps[i].code, []models.DstImage{}, err
		}
	}

	return 100, images, nil
}

// 安装依赖
func installDependency(less string) ([]models.DstImage, error) {
	var (
		images  []models.DstImage
		cmdArgs []string
	)

	if less == "1" {
		cmdArgs = []string{
			"install",
			"-y",
			"unzip",
		}
	} else {
		cmdArgs = []string{
			"install",
			"-y",
			"libxcb-glx0",
			"libx11-xcb1",
			"libxkbcommon-x11-0",
			"libxcb-cursor0",
			"libxcb-icccm4",
			"libxcb-image0",
			"libxcb-keysyms1",
			"libxcb-randr0",
			"libxcb-render-util0",
			"libxcb-shm0",
			"libxcb-sync1",
			"libxcb-xfixes0",
			"libxcb-render0",
			"libxcb-shape0",
			"libxcb-xkb1",
			"libsm6",
			"libice6",
			"libwayland-egl1",
			"libwayland-client0",
			"libwayland-cursor0",
			"libglx0",
			"libopengl0",
			"libegl1",
			"libharfbuzz0b",
			"libpcre2-16-0",
			"unzip",
		}
	}
	cmd := exec.Command("apt", cmdArgs...)
	if _, err := cmd.CombinedOutput(); err != nil {
		logger.Logger.Errorf("安装依赖失败: %v", err)
		return images, fmt.Errorf("安装依赖失败: %v", err)
	}

	return images, nil
}

// 安装stex
func installStex(proxy string) ([]models.DstImage, error) {
	var images []models.DstImage

	// 幂等
	_ = utils.RemoveDir(stexDir)

	_ = utils.EnsureDirExists(stexDir)
	filename := "Stex_v0.6_Linux_Static_x64.24.04.g++-14.zip"
	url := "https://github.com/oblivioncth/Stexatlaser/releases/download/v0.6/" + filename
	if proxy != "" {
		if proxy[len(proxy)-1] != '/' {
			proxy += "/"
		}
		url = fmt.Sprintf("%s%s", proxy, url)
	}
	client := &http.Client{
		Timeout: utils.HttpTimeout * time.Second,
	}
	resp, err := client.Get(url)
	if err != nil {
		logger.Logger.Errorf("下载Stex失败: %v", err)
		return images, fmt.Errorf("下载Stex失败: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		logger.Logger.Errorf("下载Stex失败，HTTP代码: %d", resp.Status)
		return images, fmt.Errorf("下载Stex失败，HTTP代码: " + resp.Status)
	}
	filePath := filepath.Join(stexDir, filename)
	out, err := os.Create(filePath)
	if err != nil {
		_ = utils.RemoveDir(stexDir)
		logger.Logger.Errorf("下载Stex失败: %v", err)
		return images, fmt.Errorf("下载Stex失败: %v", err)
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		_ = utils.RemoveDir(stexDir)
		logger.Logger.Errorf("下载Stex失败: %v", err)
		return images, fmt.Errorf("下载Stex失败: %v", err)
	}
	cmdArgs := []string{
		filePath,
		"-d",
		stexDir,
	}
	cmd := exec.Command("unzip", cmdArgs...)
	if _, err := cmd.CombinedOutput(); err != nil {
		_ = utils.RemoveDir(stexDir)
		logger.Logger.Errorf("解压Stex失败: %v", err)
		return images, fmt.Errorf("解压Stex失败: %v", err)
	}
	cmd = exec.Command("chmod", "+x", StexBin)
	if _, err := cmd.CombinedOutput(); err != nil {
		_ = utils.RemoveDir(stexDir)
		logger.Logger.Errorf("添加Stex执行权限失败: %v", err)
		return images, fmt.Errorf("添加Stex执行权限失败: %v", err)
	}
	cmd = exec.Command(StexBin, "-v")
	if _, err := cmd.CombinedOutput(); err != nil {
		_ = utils.RemoveDir(stexDir)
		logger.Logger.Errorf("stex执行失败: %v", err)
		return images, fmt.Errorf("stex执行失败: %v", err)
	}

	return images, nil
}

// 解压游戏tex文件
func unzipDstImages(string) ([]models.DstImage, error) {
	var images []models.DstImage

	// 幂等
	_ = utils.RemoveDir(gameImagesPath)

	_ = utils.EnsureDirExists(gameImagesPath)
	cmdArgs := []string{
		"dst/data/databundles/images.zip",
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

// 游戏tex文件转为png文件 dmp_files/stex/bin/stex unpack -i dmp_files/dst_images/images/inventoryimages[1 2 3 4].xml -o dmp_files/dst_images
func texToPng(string) ([]models.DstImage, error) {
	var images []models.DstImage

	inventoryimages := []string{
		"inventoryimages1",
		"inventoryimages2",
		"inventoryimages3",
		"inventoryimages4",
	}
	for _, inventoryimage := range inventoryimages {
		cmdArgs := []string{
			"unpack",
			"-i",
			fmt.Sprintf("%s/images/%s.xml", gameImagesPath, inventoryimage),
			"-o",
			gameImagesPath,
		}
		cmd := exec.Command(StexBin, cmdArgs...)
		if _, err := cmd.CombinedOutput(); err != nil {
			for _, dir := range inventoryimages {
				_ = utils.RemoveDir(path.Join(gameImagesPath, dir))
			}
			_ = utils.BashCMD(fmt.Sprintf("rm -f %s/*.png", gameImagesPath))
			logger.Logger.Errorf("游戏tex: %s 转png失败: %v", inventoryimage, err)
			return images, fmt.Errorf("游戏tex: %s 转png失败: %v", inventoryimage, err)
		}
		err := utils.BashCMD(fmt.Sprintf("cp %s/%s/*.png %s/", gameImagesPath, inventoryimage, gameImagesPath))
		if err != nil {
			for _, dir := range inventoryimages {
				_ = utils.RemoveDir(path.Join(gameImagesPath, dir))
			}
			_ = utils.BashCMD(fmt.Sprintf("rm -f %s/*.png", gameImagesPath))
			logger.Logger.Errorf("整理png文件失败: %v", err)
			return images, fmt.Errorf("整理png文件失败: %v", err)
		}
		_ = utils.RemoveDir(fmt.Sprintf("%s/%s/*", gameImagesPath, inventoryimage))
	}
	_ = utils.RemoveDir(fmt.Sprintf("%s/images", gameImagesPath))

	return images, nil
}

// 安装TMIR 与 获取 翻译文件
func installTMIR(string) ([]models.DstImage, error) {
	var images []models.DstImage

	modDir := filepath.Join(db.CurrentDir, utils.PluginTmiPath, "tmir")
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

	scriptPath := filepath.Join("dst", "data", "databundles", "scripts.zip")
	tmpDir := filepath.Join(utils.PluginTmiPath, "tmp")
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
	L := lua.NewState()
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
