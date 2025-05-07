package utils

var (
	Platform   string
	Registered bool
	HomeDir    string
	STATISTICS = make(map[string][]Statistics) // 玩家统计
	SYSMETRICS []SysMetrics                    // 系统监控
	UserCache  = make(map[string]User)
)
