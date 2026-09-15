package mod

import (
	"dst-management-platform-api/i18n"
)

var message = i18n.NewExtendedI18n(map[string]i18n.Text{
	"downloading":                             {ZH: "%s 正在下载中", EN: "Downloading %s"},
	"downloaded":                              {ZH: "%s 下载完成", EN: "%s Downloaded"},
	"server error":                            {ZH: "服务器内部错误", EN: "Internal Server Error"},
	"mod configuration options error":         {ZH: "获取模组配置信息失败", EN: "Generate Mod Configuration Options Error"},
	"mod configuration values error":          {ZH: "获取模组配置失败", EN: "Generate Mod Configurations Error"},
	"modify mod configuration values error":   {ZH: "修改模组配置失败", EN: "Modify Mod Configuration Error"},
	"modify mod configuration values success": {ZH: "修改模组配置成功", EN: "Modify Mod Configuration Success"},
	"mod enable fail":                         {ZH: "模组启用失败", EN: "Mod Enable Fail"},
	"mod enable success":                      {ZH: "模组启用成功", EN: "Mod Enable Success"},
	"mod disable fail":                        {ZH: "模组禁用失败", EN: "Mod Disable Fail"},
	"mod disable success":                     {ZH: "模组禁用成功", EN: "Mod Disable Success"},
	"get enabled mod fail":                    {ZH: "获取启用模组失败", EN: "Get Enabled Mods Fail"},
	"search fail":                             {ZH: "获取模组信息失败", EN: "Search Mod Fail"},
})
