package setting

func response(message string, lang string) string {
	responseZH := map[string]string{
		"saveSuccess":         "保存成功",
		"saveFail":            "保存失败",
		"restartSuccess":      "重启成功",
		"generateSuccess":     "世界生成成功",
		"addAdmin":            "添加管理员成功",
		"addAdminFail":        "添加管理员失败",
		"addBlock":            "添加黑名单成功",
		"addBlockFail":        "添加黑名单失败",
		"addWhite":            "添加白名单成功",
		"addWhiteFail":        "添加白名单失败",
		"deleteAdmin":         "删除管理员成功",
		"deleteAdminFail":     "删除管理员失败",
		"deleteBlock":         "删除黑名单成功",
		"deleteBlockFail":     "删除黑名单失败",
		"deleteWhite":         "删除白名单成功",
		"deleteWhiteFail":     "删除白名单失败",
		"kickSuccess":         "踢出成功",
		"kickFail":            "踢出失败",
		"uploadSuccess":       "上传成功",
		"uploadFail":          "上传失败",
		"wrongUploadFile":     "导入文件未通过校验",
		"deleteOldServerFail": "删除旧服务器文件失败",
		"createNewServerFail": "创建新服务器文件失败",
	}
	responseEN := map[string]string{
		"saveSuccess":         "Save Success",
		"saveFail":            "Save Fail",
		"restartSuccess":      "Restart Success",
		"generateSuccess":     "Generate World Success",
		"addAdmin":            "Successfully added administrator",
		"addAdminFail":        "Failed to add administrator",
		"addBlock":            "Successfully added to blacklist",
		"addBlockFail":        "Failed to add to blacklist",
		"addWhite":            "Successfully added to whitelist",
		"addWhiteFail":        "Failed to add to whitelist",
		"deleteAdmin":         "Successfully deleted administrator",
		"deleteAdminFail":     "Failed to deleted administrator",
		"deleteBlock":         "Successfully deleted to blacklist",
		"deleteBlockFail":     "Failed to deleted to blacklist",
		"deleteWhite":         "Successfully deleted to whitelist",
		"deleteWhiteFail":     "Failed to deleted to whitelist",
		"kickSuccess":         "Kick Success",
		"kickFail":            "Kick Fail",
		"uploadSuccess":       "Upload Success",
		"uploadFail":          "Upload Fail",
		"wrongUploadFile":     "The import file did not pass validation",
		"deleteOldServerFail": "Failed to delete old server files",
		"createNewServerFail": "Failed to create new server files",
	}

	if lang == "zh" {
		return responseZH[message]
	} else {
		return responseEN[message]
	}
}
