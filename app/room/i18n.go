package room

import (
	"dst-management-platform-api/i18n"
)

var message = i18n.NewExtendedI18n(map[string]i18n.Text{
	"upload save fail":                    {ZH: "上传文件保存失败", EN: "file save fail"},
	"unzip fail":                          {ZH: "解压失败", EN: "unzip file fail"},
	"find cluster home fail":              {ZH: "查询存档主目录失败", EN: "find DST main path fail"},
	"cluster.ini file not found":          {ZH: "cluster.ini文件不存在", EN: "cluster.ini file not found"},
	"read cluster.ini file fail":          {ZH: "读取cluster.ini文件失败", EN: "read cluster.ini file fail"},
	"cluster.ini cluster_name not found":  {ZH: "cluster.ini中未发现[cluster_name]字段", EN: "cluster_name not found in cluster.ini"},
	"cluster.ini game_mode not found":     {ZH: "cluster.ini中未发现[game_mode]字段", EN: "game_mode not found in cluster.ini"},
	"get worlds path fail":                {ZH: "获取世界目录失败", EN: "get worlds path fail"},
	"server.ini file not found":           {ZH: "server.ini文件不存在", EN: "server.ini file not found"},
	"read server.ini file fail":           {ZH: "读取server.ini文件失败", EN: "read server.ini file fail"},
	"server.ini is_master not found":      {ZH: "server.ini中未发现[is_master]字段", EN: "is_master not found in server.ini"},
	"read is_master from server.ini fail": {ZH: "读取server.ini[is_master]字段失败", EN: "read server.ini[is_master] fail"},
	"level data not found":                {ZH: "未发现世界配置", EN: "world level data not found"},
	"no available worlds found":           {ZH: "存档文件中没有发现可用的世界", EN: "no available worlds found"},
	"number of worlds does not match":     {ZH: "上传存档世界个数与当前房间世界个数不相等", EN: "the number of worlds does not match"},
	"write file fail":                     {ZH: "写入文件失败", EN: "write file fail"},
	"upload success":                      {ZH: "上传成功", EN: "upload success"},
	"deactivate success":                  {ZH: "关闭成功", EN: "Deactivate Success"},
	"activate fail":                       {ZH: "激活成功", EN: "Activate Fail"},
	"activate success":                    {ZH: "激活成功", EN: "Activate Success"},
	"port conflict":                       {ZH: "端口 [%d] 冲突", EN: "Port [%d] Conflict"},
	"webhook test fail":                   {ZH: "Webhook 测试失败: %s", EN: "Webhook Test Failed: %s"},
	"webhook test success":                {ZH: "Webhook 测试成功", EN: "Webhook Test Success"},
})
