package diagnosis

import (
	"dst-management-platform-api/i18n"
	diagnosissvc "dst-management-platform-api/service/diagnosis"
)

// message 是本模块词条表；文案数据集中在 i18n 包，这里只做包内作用域引用。
var message = i18n.Diagnosis

// Handler 是 AI 诊断模块的 HTTP 层。
type Handler struct {
	svc *diagnosissvc.Service
}

// NewHandler 创建 AI 诊断模块 handler。
func NewHandler(svc *diagnosissvc.Service) *Handler {
	return &Handler{svc: svc}
}
