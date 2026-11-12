package dashboard

import (
	"dst-management-platform-api/i18n"
	dashboardsvc "dst-management-platform-api/service/dashboard"
)

// message 是本模块词条表；文案数据集中在 i18n 包，这里只做包内作用域引用，
// 避免在 handler 里写错成其他模块的表。
var message = i18n.Dashboard

// Handler 控制台模块 HTTP 层，业务逻辑全部由 service 承担。
// 统一响应用 message.Respond，身份提取用 middleware.Actor。
type Handler struct {
	svc *dashboardsvc.Service
}

// NewHandler 创建控制台模块 handler
func NewHandler(svc *dashboardsvc.Service) *Handler {
	return &Handler{svc: svc}
}
