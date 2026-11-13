package platform

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (s *Service) installTmiPlugin(actor service.Actor, proxy string, imageParse bool) service.Result {
	plugin, err := s.pluginDao.GetPluginByPluginName(models.PluginTmi)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	if plugin.Step == 100 {
		return service.Msg("install success", nil)
	}

	var (
		images     []models.DstImage
		installErr error
	)

	if imageParse {
		if !utils.IsValidURL(proxy) && proxy != "" {
			logger.Logger.Warnf("非法代理url已拦截, api: %s, proxy: %s", actor.Path, proxy)
			return service.Fail(constant.CodePlatformBadRequest)
		}

		step := plugin.Step

		step, images, installErr = initTmi(step)
		plugin.Step = step
		if installErr != nil {
			if result, ok := s.updatePlugin(plugin); !ok {
				return result
			}
			return service.FailF(constant.CodePlatformInstallFail, installErr.Error())
		}
	} else {
		images, installErr = installTMIR("")
		if installErr != nil {
			plugin.Step = 5
			if result, ok := s.updatePlugin(plugin); !ok {
				return result
			}
			return service.FailF(constant.CodePlatformInstallFail, installErr.Error())
		}
		plugin.Step = 100
	}

	if err := s.dstImageDao.InitImages(images); err != nil {
		logger.Logger.Errorf("更新数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	plugin.Status = true
	if result, ok := s.updatePlugin(plugin); !ok {
		return result
	}

	if err := s.dstImageDao.DeleteNoName(); err != nil {
		logger.Logger.Warnf("清理异常图片失败: %v", err)
	}

	return service.Msg("install success", nil)
}

// stexDir 是旧版本下载的 Stex 二进制目录，现在改用内置解包，仅保留用于升级清理
const stexDir = constant.PluginTmiPath + "/stex"

const tmirID = constant.TmirID

const gameImagesPath = constant.PluginTmiPath + "/dst_images"

var tmirPath = fmt.Sprintf("%s/tmir/steamapps/workshop/content/322330/%d/scripts/TMIR/itemlist/lists", constant.PluginTmiPath, constant.TmirID)

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
