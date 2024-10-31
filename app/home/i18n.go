package home

func Success(message string, lang string) string {
	successZH := map[string]string{
		"rollbackSuccess": "回档成功",
		"restartSuccess":  "重启成功",
		"shutdownSuccess": "关闭成功",
		"startupSuccess":  "开启成功",
		"updating":        "正在更新中，请耐心等待",
		"announceSuccess": "宣告成功",
		"announceFail":    "宣告失败",
		"execSuccess":     "执行成功",
		"execFail":        "执行失败",
		"resetSuccess":    "重置成功",
	}
	successEN := map[string]string{
		"rollbackSuccess": "Rollback Success",
		"restartSuccess":  "Restart Success",
		"shutdownSuccess": "Shutdown Success",
		"startupSuccess":  "Startup Success",
		"updating":        "Updating, please wait patiently",
		"announceSuccess": "Announce Success",
		"announceFail":    "Announce Failed",
		"execSuccess":     "Execute Success",
		"execFail":        "Execute Failed",
		"resetSuccess":    "Reset Success",
	}

	if lang == "zh" {
		return successZH[message]
	} else {
		return successEN[message]
	}
}
