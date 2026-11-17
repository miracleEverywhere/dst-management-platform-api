package i18n

// Diagnosis 是 AI 诊断模块的词条表：基础文案 + 本模块词条。
var Diagnosis = NewExtendedI18n(map[string]Text{
	"diagnosis not found":           {ZH: "诊断记录不存在或已被清理", EN: "Diagnosis not found"},
	"diagnosis expired":             {ZH: "诊断已过期，请重新诊断", EN: "Diagnosis expired, please run it again"},
	"diagnosis not ready":           {ZH: "诊断尚未完成，请稍后再试", EN: "Diagnosis is not ready yet"},
	"diagnosis ai config missing":   {ZH: "尚未配置可用的大模型，请先在 AI 诊断或 AI 对话设置中配置", EN: "No AI model is configured. Configure the AI diagnosis or AI chat settings first"},
	"diagnosis ai config invalid":   {ZH: "诊断大模型配置不完整或不合法", EN: "The diagnosis AI configuration is incomplete or invalid"},
	"diagnosis busy":                {ZH: "诊断任务正在进行中，请稍后再试", EN: "A diagnosis is already running, please try again later"},
	"diagnosis action invalid":      {ZH: "动作不在本次诊断的候选列表中", EN: "The action is not part of this diagnosis"},
	"diagnosis action precondition": {ZH: "动作参数或当前服务器状态不满足执行条件", EN: "The action parameters or current server state do not meet the requirements"},
	"diagnosis confirm required":    {ZH: "高风险动作需要二次确认，请输入正确的确认文本", EN: "This action requires confirmation with the correct text"},
	"diagnosis apply busy":          {ZH: "该房间已有修复任务正在执行", EN: "A repair task is already running for this room"},
	"diagnosis ai fail":             {ZH: "AI 调用失败，请检查模型配置或稍后重试", EN: "AI request failed. Check the model settings or retry later"},
	"diagnosis apply fail":          {ZH: "修复执行失败", EN: "Repair failed"},
	"diagnosis verify fail":         {ZH: "修复执行后复检未通过", EN: "Verification failed after the repair"},

	"diagnosis action start world":      {ZH: "启动世界", EN: "Start world"},
	"diagnosis action restart world":    {ZH: "重启世界", EN: "Restart world"},
	"diagnosis action restart room":     {ZH: "重启房间", EN: "Restart room"},
	"diagnosis action clean logs":       {ZH: "清理日志", EN: "Clean logs"},
	"diagnosis action predownload mods": {ZH: "预下载模组", EN: "Pre-download mods"},
	"diagnosis action update game":      {ZH: "更新游戏", EN: "Update game"},
	"diagnosis action disable mod":      {ZH: "禁用模组", EN: "Disable mod"},
	"diagnosis action enable mod":       {ZH: "启用模组", EN: "Enable mod"},
	"diagnosis action delete snapshot":  {ZH: "删除存档快照", EN: "Delete snapshot"},
	"diagnosis action restore backup":   {ZH: "恢复备份", EN: "Restore backup"},
	"diagnosis action change port":      {ZH: "修改端口", EN: "Change port"},
})
