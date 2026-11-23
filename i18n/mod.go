package i18n

// Mod 是模组模块的词条表：基础文案 + 本模块词条。
var Mod = NewExtendedI18n(map[string]Text{
	"downloading":                             {ZH: "%s 正在下载中", EN: "Downloading %s"},
	"downloaded":                              {ZH: "%s 下载完成", EN: "%s Downloaded"},
	"server error":                            {ZH: "服务器内部错误", EN: "Internal Server Error"},
	"mod configuration options error":         {ZH: "获取模组配置信息失败", EN: "Generate Mod Configuration Options Error"},
	"mod configuration values error":          {ZH: "获取模组配置失败", EN: "Generate Mod Configurations Error"},
	"modify mod configuration values error":   {ZH: "修改模组配置失败", EN: "Modify Mod Configuration Error"},
	"modify mod configuration values success": {ZH: "修改模组配置成功", EN: "Modify Mod Configuration Success"},
	"mod enable fail":                         {ZH: "模组启用失败", EN: "Mod Enable Fail"},
	"mod enable success":                      {ZH: "模组启用成功", EN: "Mod Enable Success"},
	"mod enable success but globally banned":  {ZH: "模组启用成功，但该模组已被全局屏蔽，服务器不会加载它", EN: "Mod enabled, but it is globally banned and the server will not load it"},
	"mod disable fail":                        {ZH: "模组禁用失败", EN: "Mod Disable Fail"},
	"mod disable success":                     {ZH: "模组禁用成功", EN: "Mod Disable Success"},
	"get enabled mod fail":                    {ZH: "获取启用模组失败", EN: "Get Enabled Mods Fail"},
	"get banned mods fail":                    {ZH: "获取全局屏蔽模组列表失败", EN: "Get Globally Banned Mods Fail"},
	"write banned mods file fail":             {ZH: "写入全局屏蔽模组文件失败", EN: "Write Banned Mods File Fail"},
	"save banned mods success":                {ZH: "保存全局屏蔽模组列表成功", EN: "Save Globally Banned Mods Success"},
	"search fail":                             {ZH: "获取模组信息失败", EN: "Search Mod Fail"},
	"delete partial success":                  {ZH: "删除完成：成功 %d 个，失败 %d 个（失败 ID：%s）", EN: "Delete finished: %d succeeded, %d failed (failed IDs: %s)"},
})
