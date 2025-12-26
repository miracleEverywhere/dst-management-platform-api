package utils

import (
	"sync"

	"github.com/gin-gonic/gin"
)

var I18nMutex sync.Mutex

type BaseI18n struct {
	ZH map[string]string
	EN map[string]string
}

// Get 根据header返回不同的message
func (b *BaseI18n) Get(c *gin.Context, message string) string {
	switch c.Request.Header.Get("X-I18n-Lang") {
	case "zh":
		return b.ZH[message]
	case "en":
		return b.EN[message]
	default:
		return b.ZH[message]
	}
}

// I18n 全局的message，由各个app中的子i18n调用
var I18n = BaseI18n{
	ZH: map[string]string{
		"bad request":       "请求参数错误",
		"database error":    "数据库连接失败",
		"create success":    "创建成功",
		"create fail":       "创建失败",
		"add success":       "添加成功",
		"add fail":          "添加失败",
		"update success":    "更新成功",
		"update fail":       "更新失败",
		"download success":  "下载成功",
		"download fail":     "下载失败",
		"delete success":    "删除成功",
		"delete fail":       "删除失败",
		"permission needed": "权限不足",
		"token fail":        "Token认证失败",
	},
	EN: map[string]string{
		"bad request":       "Bad Request",
		"database error":    "Database Connection Error",
		"create success":    "Create Success",
		"create fail":       "Create Fail",
		"add success":       "Add Success",
		"add fail":          "Add Fail",
		"update success":    "Update Success",
		"update fail":       "Update Fail",
		"download success":  "Download Success",
		"download fail":     "Download Fail",
		"delete success":    "Delete Success",
		"delete fail":       "Delete Fail",
		"permission needed": "Insufficient Permissions",
		"token fail":        "Token Auth Fail",
	},
}
