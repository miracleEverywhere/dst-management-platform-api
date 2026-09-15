package logs

import (
	"dst-management-platform-api/i18n"
)

// logs 目前只使用 utils 的基础文案，保留独立词条表便于后续扩展
var message = i18n.NewExtendedI18n(map[string]i18n.Text{})
