package home

func response(message string, lang string) string {
	successZH := map[string]string{
		"rollbackSuccess": "回档成功",
		"rollbackFail":    "回档失败",
		"restartSuccess":  "重启成功",
		"restartFail":     "重启失败",
		"shutdownSuccess": "关闭成功",
		"shutdownFail":    "关闭失败",
		"startupSuccess":  "启动成功",
		"startupFail":     "启动失败",
		"updating":        "正在更新中，请耐心等待",
		"announceSuccess": "宣告成功",
		"announceFail":    "宣告失败",
		"execSuccess":     "执行成功",
		"execFail":        "执行失败",
		"resetSuccess":    "重置成功",
		"resetFail":       "重置失败",
		"deleteSuccess":   "删除成功",
		"deleteFail":      "删除失败",
		"addSuccess":      "添加成功",
	}
	successEN := map[string]string{
		"rollbackSuccess": "Rollback Success",
		"rollbackFail":    "Rollback Fail",
		"restartSuccess":  "Restart Success",
		"restartFail":     "Restart Fail",
		"shutdownSuccess": "Shutdown Success",
		"shutdownFail":    "Shutdown Fail",
		"startupSuccess":  "Startup Success",
		"startupFail":     "Startup Fail",
		"updating":        "Updating, please wait patiently",
		"announceSuccess": "Announce Success",
		"announceFail":    "Announce Failed",
		"execSuccess":     "Execute Success",
		"execFail":        "Execute Failed",
		"resetSuccess":    "Reset Success",
		"resetFail":       "Reset Fail",
		"deleteSuccess":   "Delete Success",
		"deleteFail":      "Delete Failed",
		"addSuccess":      "Add Success",
	}

	if lang == "zh" {
		return successZH[message]
	} else {
		return successEN[message]
	}
}
