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
	i.ZH["upload save fail"] = "上传文件保存失败"
	i.ZH["unzip fail"] = "解压失败"
	i.ZH["find cluster home fail"] = "查询存档主目录失败"
	i.ZH["cluster.ini file not found"] = "cluster.ini文件不存在"
	i.ZH["read cluster.ini file fail"] = "读取cluster.ini文件失败"
	i.ZH["cluster.ini cluster_name not found"] = "cluster.ini中未发现[cluster_name]字段"
	i.ZH["cluster.ini game_mode not found"] = "cluster.ini中未发现[game_mode]字段"
	i.ZH["get worlds path fail"] = "获取世界目录失败"
	i.ZH["server.ini file not found"] = "server.ini文件不存在"
	i.ZH["read server.ini file fail"] = "读取server.ini文件失败"
	i.ZH["server.ini is_master not found"] = "server.ini中未发现[is_master]字段"
	i.ZH["read is_master from server.ini fail"] = "读取server.ini[is_master]字段失败"
	i.ZH["server.ini name not found"] = "server.ini中未发现[name]字段"
	i.ZH["level data not found"] = "未发现世界配置"
	i.ZH["no available worlds found"] = "存档文件中没有发现可用的世界"
	i.ZH["number of worlds does not match"] = "上传存档世界个数与当前房间世界个数不相等"
	i.ZH["write file fail"] = "写入文件失败"
	i.ZH["upload success"] = "上传成功"
	i.ZH["deactivate success"] = "关闭成功"
	i.ZH["activate fail"] = "激活成功"
	i.ZH["activate success"] = "激活成功"

	i.EN["room name exist"] = "Room Name Already Existed"
	i.EN["upload save fail"] = "file save fail"
	i.EN["unzip fail"] = "unzip file fail"
	i.EN["find cluster home fail"] = "find DST main path fail"
	i.EN["cluster.ini file not found"] = "cluster.ini file not found"
	i.EN["read cluster.ini file fail"] = "read cluster.ini file fail"
	i.EN["cluster.ini cluster_name not found"] = "cluster_name not found in cluster.ini"
	i.EN["cluster.ini game_mode not found"] = "game_mode not found in cluster.ini"
	i.EN["get worlds path fail"] = "get worlds path fail"
	i.EN["server.ini file not found"] = "server.ini file not found"
	i.EN["read server.ini file fail"] = "read server.ini file fail"
	i.EN["server.ini is_master not found"] = "is_master not found in server.ini"
	i.EN["read is_master from server.ini fail"] = "read server.ini[is_master] fail"
	i.EN["server.ini name not found"] = "name not found in server.ini"
	i.EN["level data not found"] = "world level data not found"
	i.EN["no available worlds found"] = "no available worlds found"
	i.EN["number of worlds does not match"] = "the number of worlds does not match"
	i.EN["write file fail"] = "write file fail"
	i.EN["upload success"] = "upload success"
	i.EN["deactivate success"] = "Deactivate Success"
	i.EN["activate fail"] = "Activate Fail"
	i.EN["activate success"] = "Activate Success"

	return i
}

var message = NewExtendedI18n()
