package tools

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

	i.ZH["get backup fail"] = "获取备份文件失败"
	i.ZH["create backup fail"] = "创建备份文件失败"
	i.ZH["create backup success"] = "创建成功"
	i.ZH["restore fail"] = "恢复失败"
	i.ZH["restore success"] = "恢复成功"
	i.ZH["get setting fail"] = "获取定时通知设置失败"
	i.ZH["generate map fail"] = "生成地图失败"
	i.ZH["get snapshot fail"] = "获取备份文件失败"
	i.ZH["tex convert fail"] = "物品图片转换失败"
	i.ZH["parse item list fail"] = "解析物品列表失败"
	i.ZH["no such category"] = "没有发现种类: %s"
	i.ZH["exec fail"] = "执行失败"
	i.ZH["exec success"] = "执行成功"

	i.EN["get backup fail"] = "get backup fail"
	i.EN["create backup fail"] = "create backup fail"
	i.EN["create backup success"] = "create success"
	i.EN["restore fail"] = "restore fail"
	i.EN["restore success"] = "restore success"
	i.EN["get setting fail"] = "Get Announce Settings Fail"
	i.EN["generate map fail"] = "generate map fail"
	i.EN["get snapshot fail"] = "get snapshot fail"
	i.EN["tex convert fail"] = "Item Image Conversion Fail"
	i.EN["parse item list fail"] = "Parse Item List Fail"
	i.EN["no such category"] = "no such category: %s"
	i.EN["exec fail"] = "Execute Fail"
	i.EN["exec success"] = "Execute Success"

	return i
}

var message = NewExtendedI18n()
