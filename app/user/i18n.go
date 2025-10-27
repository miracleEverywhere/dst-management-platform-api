package user

import "github.com/gin-gonic/gin"

func message(c *gin.Context, message string) string {
	zh := map[string]string{
		"bad request":      "请求参数错误",
		"register success": "注册成功",
		"register fail":    "注册失败",
		"user exist":       "请勿重复注册",
		"create fail":      "创建失败",
		"create success":   "创建成功",
		"login fail":       "登录失败",
		"login success":    "登录成功",
		"wrong password":   "密码错误",
		"user not exist":   "用户不存在",
		"disabled":         "用户已被禁用",
	}
	en := map[string]string{
		"bad request":      "Bad Request",
		"register success": "Register Success",
		"register fail":    "Register Fail",
		"user exist":       "User Existed",
		"create fail":      "Create Fail",
		"create success":   "Create Success",
		"login fail":       "Login Fail",
		"login success":    "Login Success",
		"wrong password":   "Wrong Password",
		"user not exist":   "User Not Exist",
		"disabled":         "User is Disabled",
	}

	switch c.Request.Header.Get("X-I18n-Lang") {
	case "zh":
		return zh[message]
	case "en":
		return en[message]
	default:
		return zh[message]
	}
}
