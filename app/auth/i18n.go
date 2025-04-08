package auth

func Response(message string, lang string) string {
	zh := map[string]string{
		"loginSuccess":   "登录成功",
		"updatePassword": "密码修改成功",
		"userExist":      "用户名已存在",
		"createSuccess":  "创建成功",
	}
	en := map[string]string{
		"loginSuccess":   "Login Response",
		"updatePassword": "Update Password Response",
		"userExist":      "Username already exist",
		"createSuccess":  "Create Success",
	}

	if lang == "zh" {
		return zh[message]
	} else {
		return en[message]
	}
}
