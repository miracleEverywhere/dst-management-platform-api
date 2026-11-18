package i18n

import (
	"fmt"
	"net/http"

	"dst-management-platform-api/constant"
	"dst-management-platform-api/service"

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

// Respond 把 service 层的业务结果翻译成本地化响应，保持 {"code","message","data"} 结构。
// 它是 app/ 与 middleware 写响应的唯一入口：
//   - 文案优先级：Result.Msg（已是最终文案，不再翻译）> Result.MsgBuilder（需要与其他字符串拼接）
//     > MsgKey/MsgArgs（显式词条）> 错误码注册表反查出的词条
//   - 注册表反查出的词条按该码所属模块的词条表翻译，跨模块透传的码（如云存档导入失败时
//     透传房间模块的存档解析错误码）也能取到文案
//   - 词条缺失时兜底 unknown error，避免前端拿到空 message
//   - Result.Status 为 0 时使用 HTTP 200
func (b *BaseI18n) Respond(c *gin.Context, r service.Result) {
	status := r.Status
	if status == 0 {
		status = http.StatusOK
	}

	msg := r.Msg
	if msg == "" && r.MsgBuilder != nil {
		msg = r.MsgBuilder(func(msgKey string, args ...any) string {
			return b.get(c, msgKey, args...)
		})
	}
	if msg == "" {
		table := b
		msgKey := r.MsgKey
		if msgKey == "" {
			if entry, ok := constant.LookupCode(r.Code); ok {
				msgKey = entry.TextKey
				if codeTable, ok := codeTables[entry.Module]; ok {
					table = codeTable
				}
			} else {
				msgKey = "unknown error"
			}
		}
		msg = table.get(c, msgKey, r.MsgArgs...)
	}
	if msg == "" {
		// 码已注册但对应模块的词条表里没有该 key 时不能把空串写出去
		msg = b.Get(c, "unknown error")
	}

	c.JSON(status, gin.H{"code": r.Code, "message": msg, "data": r.Data})
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

// get 按请求语言取词条，带格式化参数时走 GetF
func (b *BaseI18n) get(c *gin.Context, msgKey string, args ...any) string {
	if len(args) > 0 {
		return b.GetF(c, msgKey, args...)
	}
	return b.Get(c, msgKey)
}

// codeTables 错误码所属模块（constant.Entry.Module）到该模块词条表的映射。
// 错误码的文案由注册表指向某个模块的词条，与当前请求由哪个模块写出响应无关：
// 例如云存档导入失败会透传房间模块的存档解析错误码，文案必须去房间模块的表里取，
// 否则只查调用方模块的表会命中不到词条、返回空 message。
var codeTables = map[string]*BaseI18n{
	"framework": &I18n,
	"user":      &User.BaseI18n,
	"room":      &Room.BaseI18n,
	"mod":       &Mod.BaseI18n,
	"dashboard": &Dashboard.BaseI18n,
	"platform":  &Platform.BaseI18n,
	"logs":      &Logs.BaseI18n,
	"tools":     &Tools.BaseI18n,
	"player":    &Player.BaseI18n,
	"cloud":     &Cloud.BaseI18n,
	"diagnosis": &Diagnosis.BaseI18n,
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
		"unknown error":     "服务器内部错误",
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
		"unknown error":     "Internal Server Error",
	},
}
