package utils

var (
	Platform            string
	Registered          bool
	HomeDir             string
	STATISTICS          = make(map[string][]Statistics) // 玩家统计
	SYSMETRICS          []SysMetrics                    // 系统监控
	UserCache           = make(map[string]User)
	UpdateModID         string // 玩家通过聊天窗口进行mod更新的校验ID
	UpdateModProcessing bool   // 确定更新后会sleep一段时间，该值为true
)
