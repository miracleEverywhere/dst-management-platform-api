package utils

/* 全局变量，除了数据库缓存落盘，其余的都只在内存中 */
var (
	Platform            string                              // 运行的平台类型
	Registered          bool                                // 是否已经完成注册
	HomeDir             string                              // 当前用户的$HOME目录绝对路径
	STATISTICS          = make(map[string][]Statistics)     // 玩家统计
	SYSMETRICS          []SysMetrics                        // 系统监控
	PlayTimeCount       = make(map[string]map[string]int64) // 玩家游戏时长统计
	UserCache           = make(map[string]User)             // 玩家信息缓存
	UpdateModProcessing bool                                // 玩家确定mod更新后会sleep一段时间，此时为true
	InContainer         bool                                // DMP是否由容器启动
	DstUpdating         bool                                // 饥荒是否正在更新
	DstInstalling       bool                                // 饥荒是否正在安装
	DBCache             Config                              // 数据库缓存
)
