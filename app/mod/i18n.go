package mod

func response(message string, lang string) string {
	responseZH := map[string]string{
		"needDownload": "请先下载模组",
	}
	responseEN := map[string]string{
		"needDownload": "Please download MOD first",
	}

	if lang == "zh" {
		return responseZH[message]
	} else {
		return responseEN[message]
	}
}
