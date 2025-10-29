package room

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

	i.ZH["room name exist"] = "房间名重复"
	i.EN["room name exist"] = "Room Name Already Existed"

	return i
}

var message = NewExtendedI18n()
