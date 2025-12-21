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

	i.EN["get os info fail"] = "Get OS Info Fail"
	i.EN["get screens fail"] = "Get Screens Fail"
	i.EN["kill screen fail"] = "Kill Screens Fail"
	i.EN["kill screen success"] = "Kill Screens Success"

	return i
}

var message = NewExtendedI18n()
