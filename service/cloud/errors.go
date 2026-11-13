package cloud

import (
	"errors"
	"fmt"
	"net/http"
)

// ===== 云服务客户端 =====
// ErrRelink 表示云凭据已失效或无法解密，需要用户重新绑定云账号。
var ErrRelink = errors.New("cloud relink required")

// APIError 是云服务返回的错误。Unavailable 为 true 时表示网络错误、超时或 5xx，
// 只有这类错误允许在云登录时回退到本地账号。
type APIError struct {
	StatusCode  int
	Code        int
	Message     string
	Unavailable bool
	cause       error
}

func (e *APIError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("cloud api %d: %s: %v", e.StatusCode, e.Message, e.cause)
	}

	return fmt.Sprintf("cloud api %d: %s", e.StatusCode, e.Message)
}

func (e *APIError) Unwrap() error {
	return e.cause
}

func IsUnavailable(err error) bool {
	var target *APIError
	return errors.As(err, &target) && target.Unavailable
}

func IsUnauthorized(err error) bool {
	var target *APIError
	return errors.As(err, &target) && target.StatusCode == http.StatusUnauthorized
}

func IsForbidden(err error) bool {
	var target *APIError
	return errors.As(err, &target) && target.StatusCode == http.StatusForbidden
}

func MessageOf(err error) string {
	var target *APIError
	if errors.As(err, &target) && target.Message != "" {
		return target.Message
	}

	return ""
}
