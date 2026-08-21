package cache

import "sync"

type SysMetrics struct {
	Timestamp   int64   `json:"timestamp"`
	Cpu         float64 `json:"cpu"`
	Memory      float64 `json:"memory"`
	NetUplink   float64 `json:"netUplink"`
	NetDownlink float64 `json:"netDownlink"`
	Disk        float64 `json:"disk"`
}

var (
	// SystemMetrics 系统监控数据
	SystemMetrics []SysMetrics
	// SystemMetricsMutex 系统监控数据锁
	SystemMetricsMutex sync.RWMutex
)
