package user

import "dst-management-platform-api/utils"

type ExtendedI18n struct {
	utils.BaseI18n
}

func NewExtendedI18n() *ExtendedI18n {
	i := &ExtendedI18n{
		BaseI18n: utils.BaseI18n{
			ZH: make(map[string]string),
			EN: make(map[string]string),
		},
	}

	utils.I18nMutex.Lock()
	defer utils.I18nMutex.Unlock()

	// 复制基础翻译
	for k, v := range utils.I18n.ZH {
		i.ZH[k] = v
	}
	for k, v := range utils.I18n.EN {
		i.EN[k] = v
	}

	// 添加扩展翻译
	i.ZH["register success"] = "注册成功"
	i.ZH["register fail"] = "注册失败"
	i.ZH["user exist"] = "请勿重复注册"
	i.ZH["login fail"] = "登录失败"
	i.ZH["login success"] = "登录成功"
	i.ZH["wrong password"] = "密码错误"
	i.ZH["user not exist"] = "用户不存在"
	i.ZH["disabled"] = "用户已被禁用"
	i.ZH["myself update success"] = "修改成功，请重新登录"
	i.ZH["delete all users"] = "禁止删除所有用户"

	i.EN["register success"] = "Register Success"
	i.EN["register fail"] = "Register Fail"
	i.EN["user exist"] = "User Existed"
	i.EN["login fail"] = "Login Fail"
	i.EN["login success"] = "Login Success"
	i.EN["wrong password"] = "Wrong Password"
	i.EN["user not exist"] = "User Not Exist"
	i.EN["disabled"] = "User is Disabled"
	i.EN["myself update success"] = "Update success, please re-login"
	i.EN["delete all users"] = "Prohibit deletion of all users"

	return i
}

var message = NewExtendedI18n()
