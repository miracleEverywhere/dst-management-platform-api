package logs

func response(message string, lang string) string {
	responseZH := map[string]string{
		"tarFail":      "打包日志压缩文件失败",
		"fileReadFail": "读取文件失败",
	}
	responseEN := map[string]string{
		"installing":   "Failed to compress log files into a package",
		"fileReadFail": "File Read Fail",
	}

	if lang == "zh" {
		return responseZH[message]
	} else {
		return responseEN[message]
	}
}
