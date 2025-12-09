package dashboard

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
	i.ZH["startup game success"] = "启动成功"
	i.ZH["shutdown game fail"] = "关闭失败"
	i.ZH["shutdown game success"] = "关闭成功"
	i.ZH["restart game fail"] = "重启失败"
	i.ZH["restart game success"] = "重启成功"
	i.ZH["updating"] = "更新中，请耐心等待"
	i.ZH["reset game fail"] = "重置失败"
	i.ZH["reset game success"] = "重置成功"
	i.ZH["delete game fail"] = "清空世界失败"
	i.ZH["delete game success"] = "清空世界成功"
	i.ZH["announce fail"] = "宣告失败"
	i.ZH["announce success"] = "宣告成功"
	i.ZH["exec fail"] = "执行失败"
	i.ZH["exec success"] = "执行成功"
	i.ZH["connection code fail"] = "直连代码获取失败"
	i.ZH["check lobby fail"] = "检查世界失败"

	i.EN["startup game fail"] = "Startup Fail"
	i.EN["startup game success"] = "Startup Success"
	i.EN["shutdown game fail"] = "Shutdown Fail"
	i.EN["shutdown game success"] = "Shutdown Success"
	i.EN["restart game fail"] = "Restart Fail"
	i.EN["restart game success"] = "Restart Success"
	i.EN["updating"] = "Updating, please wait patiently"
	i.EN["reset game fail"] = "Reset Fail"
	i.EN["reset game success"] = "Reset Success"
	i.EN["delete game fail"] = "Delete Fail"
	i.EN["delete game success"] = "Delete Success"
	i.EN["announce fail"] = "Announce Fail"
	i.EN["announce success"] = "Announce Success"
	i.EN["exec fail"] = "Execute Fail"
	i.EN["exec success"] = "Execute Success"
	i.EN["connection code fail"] = "Get Connection Code Fail"
	i.EN["check lobby fail"] = "Check Lobby Fail"

	return i
}

var message = NewExtendedI18n()
