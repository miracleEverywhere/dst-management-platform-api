package tools

import (
	"dst-management-platform-api/i18n"
)

var message = i18n.NewExtendedI18n(map[string]i18n.Text{
	"get backup fail":                {ZH: "获取备份文件失败", EN: "get backup fail"},
	"create backup fail":             {ZH: "创建备份文件失败", EN: "create backup fail"},
	"create backup success":          {ZH: "创建成功", EN: "create success"},
	"restore fail":                   {ZH: "恢复失败", EN: "restore fail"},
	"restore success":                {ZH: "恢复成功", EN: "restore success"},
	"generate map fail":              {ZH: "生成地图失败", EN: "generate map fail"},
	"get snapshot fail":              {ZH: "获取游戏存档失败", EN: "Get Snapshot Fail"},
	"embedding config incomplete":    {ZH: "嵌入模型配置不完整", EN: "Embedding model configuration is incomplete"},
	"embedding base url invalid":     {ZH: "嵌入模型 Base URL 不合法", EN: "Embedding model Base URL is invalid"},
	"embedding build not running":    {ZH: "当前没有正在进行的嵌入索引构建", EN: "No embedding index build is running"},
	"embedding build cancel success": {ZH: "已取消嵌入索引构建", EN: "Embedding index build cancelled"},
})
