package logs

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

	i.ZH["startup game fail"] = "启动失败"
	i.ZH["download fail"] = "下载失败"

	i.EN["startup game fail"] = "Startup Fail"
	i.EN["download fail"] = "Download Fail"

	return i
}

var message = NewExtendedI18n()
