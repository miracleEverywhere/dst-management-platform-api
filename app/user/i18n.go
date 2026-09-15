package user

import (
	"dst-management-platform-api/i18n"
)

var message = i18n.NewExtendedI18n(map[string]i18n.Text{
	"register success":      {ZH: "注册成功", EN: "Register Success"},
	"user exist":            {ZH: "请勿重复注册", EN: "User Existed"},
	"login fail":            {ZH: "登录失败", EN: "Login Fail"},
	"login success":         {ZH: "登录成功", EN: "Login Success"},
	"wrong password":        {ZH: "密码错误", EN: "Wrong Password"},
	"user not exist":        {ZH: "用户不存在", EN: "User Not Exist"},
	"disabled":              {ZH: "用户已被禁用", EN: "User is Disabled"},
	"myself update success": {ZH: "修改成功，请重新登录", EN: "Update success, please re-login"},
	"delete all users":      {ZH: "禁止删除所有用户", EN: "Prohibit deletion of all users"},
	"revoke success":        {ZH: "Token撤销成功", EN: "Token Revoked Successfully"},
	"revoke fail":           {ZH: "Token撤销失败", EN: "Token Revocation Failed"},
})
