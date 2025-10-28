package room

import "dst-management-platform-api/utils"

type ExtendedI18n struct {
	utils.BaseI18n
}

func NewExtendedI18n() *ExtendedI18n {
	e := &ExtendedI18n{
		BaseI18n: utils.BaseI18n{
			ZH: make(map[string]string),
			EN: make(map[string]string),
		},
	}

	utils.I18nMutex.Lock()
	defer utils.I18nMutex.Unlock()

	// 复制基础翻译
	for k, v := range utils.I18n.ZH {
		e.ZH[k] = v
	}
	for k, v := range utils.I18n.EN {
		e.EN[k] = v
	}

	// 添加扩展翻译
	e.ZH["welcome"] = "欢迎"
	e.EN["welcome"] = "Welcome"

	return e
}

var message = NewExtendedI18n()
