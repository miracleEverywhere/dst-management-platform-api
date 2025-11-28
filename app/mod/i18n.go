package mod

import "dst-management-platform-api/utils"

type ExtendedI18n struct {
	utils.BaseI18n
}

func NewExtendedI18n() *ExtendedI18n {
	i := &ExtendedI18n{
		BaseI18n: utils.BaseI18n{
			ZH: make(map[string]string),
			EN: make(map[string]string),
		},
	}

	utils.I18nMutex.Lock()
	defer utils.I18nMutex.Unlock()

	for k, v := range utils.I18n.ZH {
		i.ZH[k] = v
	}
	for k, v := range utils.I18n.EN {
		i.EN[k] = v
	}

	i.ZH["downloading"] = "开始下载模组"
	i.ZH["update completed"] = "更新完成"
	i.ZH["mod configuration options error"] = "获取模组配置信息失败"
	i.ZH["mod configuration values error"] = "获取模组配置失败"
	i.ZH["modify mod configuration values error"] = "修改模组配置失败"
	i.ZH["modify mod configuration values success"] = "修改模组配置成功"
	i.ZH["mod enable fail"] = "模组启用失败"
	i.ZH["mod enable success"] = "模组启用成功"
	i.ZH["mod disable fail"] = "模组禁用失败"
	i.ZH["mod disable success"] = "模组禁用成功"
	i.ZH["get enabled mod fail"] = "获取启用模组失败"

	i.EN["downloading"] = "Downloading Mod"
	i.EN["update completed"] = "Update Completed"
	i.EN["mod configuration options error"] = "Generate Mod Configuration Options Error"
	i.EN["mod configuration values error"] = "Generate Mod Configurations Error"
	i.EN["modify mod configuration values error"] = "Modify Mod Configuration Error"
	i.EN["modify mod configuration values success"] = "Modify Mod Configuration Success"
	i.EN["mod enable fail"] = "Mod Enable Fail"
	i.EN["mod enable success"] = "Mod Enable Success"
	i.EN["mod disable fail"] = "Mod Disable Fail"
	i.EN["mod disable success"] = "Mod Disable Success"
	i.EN["get enabled mod fail"] = "Get Enabled Mods Fail"

	return i
}

var message = NewExtendedI18n()
