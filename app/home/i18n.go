package home

func Success(message string, lang string) string {
	successZH := map[string]string{
		"saveSuccess":     "保存成功",
		"restartSuccess":  "重启成功",
		"generateSuccess": "世界生成成功",
	}
	successEN := map[string]string{
		"saveSuccess":     "Save Success",
		"restartSuccess":  "Restart Success",
		"generateSuccess": "Generate World Success",
	}

	if lang == "zh" {
		return successZH[message]
	} else {
		return successEN[message]
	}
}
