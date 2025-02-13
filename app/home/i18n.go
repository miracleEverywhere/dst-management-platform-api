package home

func response(message string, lang string) string {
	successZH := map[string]string{
		"rollbackSuccess": "回档成功",
		"restartSuccess":  "重启成功",
		"restartFail":     "重启失败",
		"shutdownSuccess": "关闭成功",
		"startupSuccess":  "开启成功",
		"startupFail":     "开启失败",
		"updating":        "正在更新中，请耐心等待",
		"announceSuccess": "宣告成功",
		"announceFail":    "宣告失败",
		"execSuccess":     "执行成功",
		"execFail":        "执行失败",
		"resetSuccess":    "重置成功",
		"deleteSuccess":   "删除成功",
		"deleteFail":      "删除失败",
	}
	successEN := map[string]string{
		"rollbackSuccess": "Rollback Success",
		"restartSuccess":  "Restart Success",
		"restartFail":     "Restart Fail",
		"shutdownSuccess": "Shutdown Success",
		"startupSuccess":  "Startup Success",
		"startupFail":     "Startup Fail",
		"updating":        "Updating, please wait patiently",
		"announceSuccess": "Announce Success",
		"announceFail":    "Announce Failed",
		"execSuccess":     "Execute Success",
		"execFail":        "Execute Failed",
		"resetSuccess":    "Reset Success",
		"deleteSuccess":   "Delete Success",
		"deleteFail":      "Delete Failed",
	}

	if lang == "zh" {
		return successZH[message]
	} else {
		return successEN[message]
	}
}
