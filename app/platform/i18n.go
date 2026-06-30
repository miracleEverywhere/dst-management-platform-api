package platform

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

	// 复制基础翻译
	for k, v := range utils.I18n.ZH {
		i.ZH[k] = v
	}
	for k, v := range utils.I18n.EN {
		i.EN[k] = v
	}

	// 添加扩展翻译
	i.ZH["get os info fail"] = "获取系统信息失败"
	i.ZH["get screens fail"] = "获取Screens失败"
	i.ZH["kill screen fail"] = "关闭Screens失败"
	i.ZH["kill screen success"] = "关闭Screens成功"
	i.ZH["webhook test fail"] = "Webhook 测试失败: %s"
	i.ZH["webhook test success"] = "Webhook 测试成功"
	i.ZH["setting no change"] = "配置未修改"

	i.EN["get os info fail"] = "Get OS Info Fail"
	i.EN["get screens fail"] = "Get Screens Fail"
	i.EN["kill screen fail"] = "Kill Screens Fail"
	i.EN["kill screen success"] = "Kill Screens Success"
	i.EN["webhook test fail"] = "Webhook Test Failed: %s"
	i.EN["webhook test success"] = "Webhook Test Success"
	i.EN["setting no change"] = "Setting no change"

	return i
}

var message = NewExtendedI18n()
