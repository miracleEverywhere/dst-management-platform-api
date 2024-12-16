package tools

func response(message string, lang string) string {
	responseZH := map[string]string{
		"installing":             "正在安装中。。。",
		"duplicatedName":         "名字重复",
		"createSuccess":          "创建成功",
		"deleteSuccess":          "删除成功",
		"deleteFail":             "删除失败",
		"updateSuccess":          "更新成功",
		"updateFail":             "更新失败",
		"restoreFail":            "恢复失败",
		"restoreSuccess":         "恢复成功",
		"fileNotFound":           "文件不存在",
		"fileReadFail":           "读取文件失败",
		"restoreSuccessSaveFail": "恢复成功，写入数据库失败",
		"backupSuccess":          "备份成功",
		"backupFail":             "备份失败",
		"replaceSuccess":         "替换成功",
		"replaceFail":            "替换失败",
		"saveFail":               "保存失败",
		"createTokenSuccess":     "令牌创建成功",
	}
	responseEN := map[string]string{
		"installing":             "Installing...",
		"duplicatedName":         "Duplicated Name",
		"createSuccess":          "Create Success",
		"deleteSuccess":          "Delete Success",
		"deleteFail":             "Delete Failed",
		"updateSuccess":          "Update Success",
		"updateFail":             "Update Failed",
		"restoreFail":            "Restore Fail",
		"restoreSuccess":         "Restore Success",
		"fileNotFound":           "File Not Found",
		"fileReadFail":           "File Read Fail",
		"restoreSuccessSaveFail": "Restore Success, but writing to database failed",
		"backupSuccess":          "Backup Success",
		"backupFail":             "Backup Fail",
		"replaceSuccess":         "Replace Success",
		"replaceFail":            "Replace Fail",
		"saveFail":               "Save Fail",
		"createTokenSuccess":     "Create Token Success",
	}

	if lang == "zh" {
		return responseZH[message]
	} else {
		return responseEN[message]
	}
}
