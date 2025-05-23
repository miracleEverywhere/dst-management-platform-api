package auth

func Response(message string, lang string) string {
	zh := map[string]string{
		"loginSuccess":   "登录成功",
		"updatePassword": "密码修改成功",
		"userExist":      "用户名已存在",
		"createSuccess":  "创建成功",
		"updateSuccess":  "更新成功",
		"deleteSuccess":  "删除成功",
	}
	en := map[string]string{
		"loginSuccess":   "Login Success",
		"updatePassword": "Update Password Success",
		"userExist":      "Username already exist",
		"createSuccess":  "Create Success",
		"updateSuccess":  "Update Success",
		"deleteSuccess":  "Delete Success",
	}

	if lang == "zh" {
		return zh[message]
	} else {
		return en[message]
	}
}
