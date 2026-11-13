package platform

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
)

// PluginList 插件列表
func (s *Service) PluginList(page, pageSize int, q string) service.Result {
	plugins, err := s.pluginDao.ListPlugins(q, page, pageSize)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	data := *plugins

	return service.Success(data)
}

// PluginInstall 安装插件，imageParse 仅 TMI 插件使用
func (s *Service) PluginInstall(actor service.Actor, name, proxy string, imageParse *bool) service.Result {
	switch name {
	case models.PluginTmi:
		if imageParse == nil {
			logger.Logger.Infof("请求参数缺少imageParse, api: %s", actor.Path)
			return service.Fail(constant.CodePlatformBadRequest)
		}
		parse := *imageParse
		if cache.OsType == constant.Darwin {
			parse = false
		}
		return s.installTmiPlugin(actor, proxy, parse)
	case models.PluginChat:
		return s.installAiChatPlugin(actor, proxy)
	default:
		logger.Logger.Infof("请求参数错误, api: %s", actor.Path)
		return service.Fail(constant.CodePlatformBadRequest)
	}
}

// PluginAction 插件启用、禁用、更新、卸载
func (s *Service) PluginAction(actor service.Actor, name, action string) service.Result {
	plugin, err := s.pluginDao.GetPluginByPluginName(name)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	// 保存动作前的状态，运行时动作失败时用于回滚数据库与运行时
	original := *plugin
	wasEnabled := plugin.Status

	if result, ok := s.preparePluginAction(actor, plugin, name, action); !ok {
		return result
	}

	if err := s.pluginDao.UpdatePlugin(plugin); err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	if result, ok := s.applyPluginRuntimeAction(name, action); !ok {
		// Start 在已激活时是幂等的，ListEnabled 失败也不会停止现有监听；
		// 仅在原先未启用、可能已启动部分房间时清理运行时。
		if name == models.PluginChat && action == "enable" && !wasEnabled {
			s.aiManager.StopAll()
		}
		plugin.Status = original.Status
		plugin.Step = original.Step
		if restoreErr := s.pluginDao.UpdatePlugin(plugin); restoreErr != nil {
			logger.Logger.Errorf("恢复插件状态失败, plugin: %s, err: %v", name, restoreErr)
		}
		return result
	}

	s.notifyPluginAction(actor, name, action)

	return service.Msg("exec success", nil)
}

func (s *Service) preparePluginAction(actor service.Actor, plugin *models.Plugin, name, action string) (service.Result, bool) {
	switch action {
	case "enable":
		plugin.Status = true
	case "disable":
		plugin.Status = false
	case "update":
		if name == models.PluginChat {
			logger.Logger.Infof("请求参数错误, api: %s", actor.Path)
			return service.Fail(constant.CodePlatformBadRequest), false
		}
		if name == models.PluginTmi {
			if result, ok := s.updateTmiPlugin(); !ok {
				return result, false
			}
			plugin.Status = true
		}
	case "uninstall":
		pluginDir := pluginDirectory(name)
		if pluginDir == "" {
			return service.Fail(constant.CodePlatformBadRequest), false
		}
		if err := utils.RemoveDir(pluginDir); err != nil {
			logger.Logger.Errorf("卸载插件失败: %v", err)
			return service.Fail(constant.CodePlatformUninstallFail), false
		}
		plugin.Step = 0
		plugin.Status = false
	default:
		logger.Logger.Infof("请求参数错误, api: %s", actor.Path)
		return service.Fail(constant.CodePlatformBadRequest), false
	}
	return service.Result{}, true
}

func (s *Service) updateTmiPlugin() (service.Result, bool) {
	images, err := installTMIR("")
	if err != nil {
		logger.Logger.Errorf("更新插件失败: %v", err)
		return service.Fail(constant.CodePlatformUpdateFail), false
	}
	if err = s.dstImageDao.InitImages(images); err != nil {
		logger.Logger.Errorf("更新数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError), false
	}
	if err = s.dstImageDao.DeleteNoName(); err != nil {
		logger.Logger.Warnf("清理异常图片失败: %v", err)
	}
	return service.Result{}, true
}

func pluginDirectory(name string) string {
	switch name {
	case models.PluginTmi:
		return constant.PluginTmiPath
	case models.PluginChat:
		return constant.PluginAiChatPath
	default:
		return ""
	}
}

func (s *Service) applyPluginRuntimeAction(name, action string) (service.Result, bool) {
	if name != models.PluginChat {
		return service.Result{}, true
	}

	switch action {
	case "enable":
		if err := s.aiManager.Start(); err != nil {
			logger.Logger.Errorf("启动 AI 对话服务失败: %v", err)
			return service.Fail(constant.CodePlatformUpdateFail), false
		}
	case "disable", "uninstall":
		s.aiManager.StopAll()
	}
	return service.Result{}, true
}

func (s *Service) notifyPluginAction(actor service.Actor, name, action string) {
	// webhook 通知
	webhook.Snd.Send(webhook.EventPluginUpdated, 0, map[string]interface{}{
		"username":    actor.Username,
		"plugin_name": name,
		"action":      action,
	})
}

// PluginStatus 查询插件状态
func (s *Service) PluginStatus(name string) service.Result {
	plugin, err := s.pluginDao.GetPluginByPluginName(name)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	return service.Success(plugin.Status)
}

// updatePlugin 更新插件记录，失败时返回数据库错误结果
func (s *Service) updatePlugin(plugin *models.Plugin) (service.Result, bool) {
	if err := s.pluginDao.UpdatePlugin(plugin); err != nil {
		logger.Logger.Errorf("更新数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError), false
	}
	return service.Result{}, true
}
