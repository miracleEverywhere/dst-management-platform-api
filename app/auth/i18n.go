package auth

func Success(message string, lang string) string {
	successZH := map[string]string{
		"loginSuccess":   "登录成功",
		"updatePassword": "密码修改成功",
	}
	successEN := map[string]string{
		"loginSuccess":   "Login Success",
		"updatePassword": "Update Password Success",
	}

	if lang == "zh" {
		return successZH[message]
	} else {
		return successEN[message]
	}
}
