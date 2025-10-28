package user

import "dst-management-platform-api/utils"

type ExtendedI18n struct {
	utils.BaseI18n
}

func NewExtendedI18n() *ExtendedI18n {
	e := &ExtendedI18n{
		BaseI18n: utils.BaseI18n{
			ZH: make(map[string]string),
			EN: make(map[string]string),
		},
	}

	utils.I18nMutex.Lock()
	defer utils.I18nMutex.Unlock()

	// 复制基础翻译
	for k, v := range utils.I18n.ZH {
		e.ZH[k] = v
	}
	for k, v := range utils.I18n.EN {
		e.EN[k] = v
	}

	// 添加扩展翻译
	e.ZH["register success"] = "注册成功"
	e.ZH["register fail"] = "注册失败"
	e.ZH["user exist"] = "请勿重复注册"
	e.ZH["create fail"] = "创建失败"
	e.ZH["create success"] = "创建成功"
	e.ZH["login fail"] = "登录失败"
	e.ZH["login success"] = "登录成功"
	e.ZH["wrong password"] = "密码错误"
	e.ZH["user not exist"] = "用户不存在"
	e.ZH["disabled"] = "用户已被禁用"

	e.EN["register success"] = "Register Success"
	e.EN["register fail"] = "Register Fail"
	e.EN["user exist"] = "User Existed"
	e.EN["create fail"] = "Create Fail"
	e.EN["create success"] = "Create Success"
	e.EN["login fail"] = "Login Fail"
	e.EN["login success"] = "Login Success"
	e.EN["wrong password"] = "Wrong Password"
	e.EN["user not exist"] = "User Not Exist"
	e.EN["disabled"] = "User is Disabled"

	return e
}

var message = NewExtendedI18n()
