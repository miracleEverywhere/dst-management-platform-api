package platform

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"fmt"
	"io"
	nethttp "net/http"
	"os"
	"path/filepath"
	"time"
)

func (s *Service) installAiChatPlugin(actor service.Actor, proxy string) service.Result {
	if proxy != "" && !utils.IsValidURL(proxy) {
		logger.Logger.Warnf("非法代理url已拦截, api: %s, proxy: %s", actor.Path, proxy)
		return service.Fail(constant.CodePlatformBadRequest)
	}

	plugin, err := s.pluginDao.GetPluginByPluginName(models.PluginChat)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	if plugin.Step == 100 {
		return service.Msg("install success", nil)
	}

	original := *plugin

	if err := installAiChatWiki(proxy); err != nil {
		logger.Logger.Errorf("安装 AI Chat 插件失败: %v", err)
		return service.FailF(constant.CodePlatformInstallFail, err.Error())
	}

	plugin.Step = 100
	plugin.Status = true
	if result, ok := s.updatePlugin(plugin); !ok {
		return result
	}

	// 卸载会调用 StopAll，重新安装后必须像启用流程一样恢复房间监听。
	if err := s.aiManager.Start(); err != nil {
		if !original.Status {
			s.aiManager.StopAll()
		}
		plugin.Status = original.Status
		plugin.Step = original.Step
		if restoreErr := s.pluginDao.UpdatePlugin(plugin); restoreErr != nil {
			logger.Logger.Errorf("恢复插件状态失败, plugin: %s, err: %v", models.PluginChat, restoreErr)
		}
		logger.Logger.Errorf("启动 AI 对话服务失败: %v", err)
		return service.FailF(constant.CodePlatformInstallFail, err.Error())
	}

	return service.Msg("install success", nil)
}

const aiChatWikiURL = "https://raw.githubusercontent.com/miracleEverywhere/dst-management-platform-api/master/aichat/wiki/wiki.zip"

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

	client := &nethttp.Client{Timeout: constant.HttpTimeout * time.Second}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("下载 Wiki 压缩包失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != nethttp.StatusOK {
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
