// Package service 提供 app 层与业务实现之间的公共契约。
//
// 约定：
//   - service 层不依赖 gin，也不直接写 HTTP 响应；业务结果统一用 Result 返回。
//   - 错误用 `constant` 里的业务错误码标识（一个原因一个码），文案由 i18n 包按码反查词条，
//     service 不需要知道任何文案 key。
//   - 成功（HTTP/业务码 200）如果需要自定义文案，用 Msg/MsgF 传词条 key。
//   - 少量必须覆盖文案的场景（云端透传、token 撤销）用 FailMsg/FailKey。
//   - lite 版不引用本包，保持独立实现。
package service

// Actor 是一次请求的身份上下文。
// handler 从 JWT 中间件写入的 gin.Context 中提取后传给 service，
// service 层据此做权限判断与语言相关的底层调用，避免反向依赖 gin。
type Actor struct {
	Username string
	Nickname string
	Role     string
	// Lang 对应请求头 X-I18n-Lang，用于下传给 dst 等会本地化的底层组件
	Lang string
	// Path 当前请求路径，仅用于 service 层保持与原实现一致的日志输出
	Path string
}

// IsAdmin 是否为管理员
func (a Actor) IsAdmin() bool {
	return a.Role == "admin"
}

// Result 是 service 层统一返回给 handler 的业务结果。
//
//	Code       业务码。错误场景必须是 constant 里的唯一错误码；成功固定 200
//	Status     HTTP 状态码，0 表示 http.StatusOK
//	Msg        字面文案，优先于其他文案来源（云端透传、个别历史接口）
//	MsgKey     词条 key。成功文案用它；少量错误需要覆盖文案时也用它
//	MsgArgs    词条格式化参数，对应 i18n.GetF
//	MsgBuilder 需要把文案与其他字符串拼接时使用，handler 传入翻译函数后由 service 决定拼装顺序
//	Data       响应体的 data 字段
type Result struct {
	Code       int
	Status     int
	Msg        string
	MsgKey     string
	MsgArgs    []any
	MsgBuilder func(tr func(msgKey string, args ...any) string) string
	Data       any
}

// Success 返回 200 且文案为字面量 "success"，对应原有直接写死 "success" 的接口
func Success(data any) Result {
	return Result{Code: 200, Msg: "success", Data: data}
}

// Msg 返回 200 并使用 i18n 词条作为成功文案
func Msg(msgKey string, data any) Result {
	return Result{Code: 200, MsgKey: msgKey, Data: data}
}

// MsgF 返回 200 并使用带格式化参数的 i18n 词条
func MsgF(msgKey string, args []any, data any) Result {
	return Result{Code: 200, MsgKey: msgKey, MsgArgs: args, Data: data}
}

// Fail 返回一个错误码，文案由注册表里的词条 key 决定
func Fail(code int) Result {
	return Result{Code: code}
}

// FailF 返回一个带格式化参数的错误码
func FailF(code int, args ...any) Result {
	return Result{Code: code, MsgArgs: args}
}

// FailData 返回一个带 data 的错误码（如分页失败时保留零值分页结构）
func FailData(code int, data any) Result {
	return Result{Code: code, Data: data}
}

// FailMsg 返回一个错误码并直接使用字面文案，不再经过词条翻译
// （用于云端透传的已本地化文案、以及需要带上底层错误细节的场景）
func FailMsg(code int, msg string) Result {
	return Result{Code: code, Msg: msg}
}

// FailKey 返回一个错误码但覆盖文案词条，用于必须区分措辞的少数场景
func FailKey(code int, msgKey string) Result {
	return Result{Code: code, MsgKey: msgKey}
}
