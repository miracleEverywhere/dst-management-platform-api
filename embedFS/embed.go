package embedFS

import (
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed dist/*
//go:embed dist/assets/*
var Dist embed.FS

//go:embed luajit/*
var LuaJit embed.FS

//go:embed shell/*
var Shell embed.FS

// CopyEmbeddedFiles err := CopyEmbeddedFiles(embedFS.Shell, "shell", "./output/shell", "start.sh", "stop.sh")
func CopyEmbeddedFiles(sourceFS embed.FS, sourceRoot, targetDir string, includeFiles ...string) error {
	// 直接构建完整的源文件路径
	for _, filename := range includeFiles {
		sourcePath := filepath.Join(sourceRoot, filename)

		// 读取嵌入文件
		data, err := sourceFS.ReadFile(sourcePath)
		if err != nil {
			return fmt.Errorf("读取嵌入文件 %s 失败: %w", sourcePath, err)
		}

		// 构建目标路径
		targetPath := filepath.Join(targetDir, filename)

		// 确保目标目录存在
		dir := filepath.Dir(targetPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录 %s 失败: %w", dir, err)
		}

		// 写入文件
		if err := os.WriteFile(targetPath, data, 0644); err != nil {
			return fmt.Errorf("写入文件 %s 失败: %w", targetPath, err)
		}
	}
	return nil
}

func GenerateDefaultFile() {
	var err error
	// luajit
	err = utils.EnsureDirExists(fmt.Sprintf("%s/luajit", utils.DmpFiles))
	if err != nil {
		logger.Logger.Error("创建dmp_files/luajit失败", "err", err)
		return
	}
	err = CopyEmbeddedFiles(LuaJit, "luajit", fmt.Sprintf("%s/luajit/", utils.DmpFiles), "liblua.so", "libluajit.so", "libpreload.so")
	if err != nil {
		logger.Logger.Error("生成luajit依赖失败", "err", err)
		return
	}

	// install 脚本
	err = CopyEmbeddedFiles(Shell, "shell", "./", "manual_install.sh")
	if err != nil {
		logger.Logger.Error("生成手动安装脚本失败", "err", err)
		return
	}

	err = utils.ChangeFileMode("./manual_install.sh", 0755)
	if err != nil {
		logger.Logger.Error("手动安装脚本添加权限失败", "err", err)
		return
	}

	// update 脚本
	err = CopyEmbeddedFiles(Shell, "shell", "./", "manual_update.sh")
	if err != nil {
		logger.Logger.Error("生成手动更新脚本失败", "err", err)
		return
	}

	err = utils.ChangeFileMode("./manual_update.sh", 0755)
	if err != nil {
		logger.Logger.Error("手动更新脚本添加权限失败", "err", err)
		return
	}

	// 删除Windows的换行符
	_ = utils.BashCMD("sed -i 's/\\r$//' manual_install.sh")
	_ = utils.BashCMD("sed -i 's/\\r$//' manual_update.sh")
}
