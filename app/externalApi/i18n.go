package externalApi

func response(message string, lang string) string {
	successZH := map[string]string{
		"getVersionFail":        "饥荒版本获取失败",
		"getConnectionCodeFail": "直连代码获取失败",
		"getModInfoFail":        "获取模组信息失败",
	}
	successEN := map[string]string{
		"getVersionFail":        "get DST version fail",
		"getConnectionCodeFail": "get connection code fail",
		"getModInfoFail":        "get mods info fail",
	}
	if lang == "zh" {
		return successZH[message]
	} else {
		return successEN[message]
	}
}
