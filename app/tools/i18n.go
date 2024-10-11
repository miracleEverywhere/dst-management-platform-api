package tools

func Success(message string, lang string) string {
	successZH := map[string]string{
		"installing": "正在安装中。。。",
	}
	successEN := map[string]string{
		"installing": "Installing...",
	}

	if lang == "zh" {
		return successZH[message]
	} else {
		return successEN[message]
	}
}
