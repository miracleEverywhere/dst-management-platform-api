package api

import (
	"encoding/json"
	"errors"
	"net/http"
)

// lite 只提供中文文案，不再按 X-I18n-Lang 做多语言切换。
const (
	msgSuccess         = "成功"
	msgBadRequest      = "请求参数错误"
	msgUpdateSuccess   = "更新成功"
	msgRegisterSuccess = "注册成功"
	msgRegisterFail    = "注册失败"
	msgUserExist       = "请勿重复注册"
	msgLoginSuccess    = "登录成功"
	msgLoginFail       = "登录失败"
	msgWrongPassword   = "密码错误"
	msgUserNotExist    = "用户不存在"
	msgTokenFail       = "Token认证失败"
)

// response 保持主项目 {"code": ..., "message": ..., "data": ...} 的响应结构。
// 业务结果由 code 字段表示，HTTP 状态码固定为 200。
type response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data"`
}

func writeJSON(w http.ResponseWriter, code int, messageText string, data any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(response{Code: code, Message: messageText, Data: data})
}

// bindJSON 只解析 JSON 请求体；lite 不再引入 gin 的多格式绑定。
func bindJSON(r *http.Request, v any) error {
	if r.Body == nil {
		return errors.New("请求体为空")
	}
	return json.NewDecoder(r.Body).Decode(v)
}
