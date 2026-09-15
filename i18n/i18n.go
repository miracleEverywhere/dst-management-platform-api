package i18n

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

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

// GetF 类似 Get，但支持格式化参数，内部调用 fmt.Sprintf
func (b *BaseI18n) GetF(c *gin.Context, message string, args ...interface{}) string {
	msg := b.Get(c, message)
	return fmt.Sprintf(msg, args...)
}

// Text 一条 i18n 词条的中英文案
type Text struct {
	ZH string
	EN string
}

// ExtendedI18n 模块级翻译表：全局基础文案与本模块词条合并后的独立副本
type ExtendedI18n struct {
	BaseI18n
}

// NewExtendedI18n 以全局基础文案为底，叠加模块词条，返回模块私有的翻译表。
// 仅在包初始化阶段调用，返回后不再写入，因此无需加锁。
func NewExtendedI18n(texts map[string]Text) *ExtendedI18n {
	i := &ExtendedI18n{
		BaseI18n: BaseI18n{
			ZH: make(map[string]string, len(I18n.ZH)+len(texts)),
			EN: make(map[string]string, len(I18n.EN)+len(texts)),
		},
	}

	for k, v := range I18n.ZH {
		i.ZH[k] = v
	}
	for k, v := range I18n.EN {
		i.EN[k] = v
	}
	for k, v := range texts {
		i.ZH[k] = v.ZH
		i.EN[k] = v.EN
	}

	return i
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
		"exec success":      "执行成功",
		"exec fail":         "执行失败",
		"permission needed": "权限不足",
		"token fail":        "Token认证失败",
		"token revoked":     "Token已被撤销",
		"too many requests": "请求过于频繁，请稍后再试",
		"invalid url":       "非法URL",
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
		"exec success":      "Execute Success",
		"exec fail":         "Execute Fail",
		"permission needed": "Insufficient Permissions",
		"token fail":        "Token Auth Fail",
		"token revoked":     "Token Revoked",
		"too many requests": "Too Many Requests",
		"invalid url":       "Invalid URL",
	},
}
