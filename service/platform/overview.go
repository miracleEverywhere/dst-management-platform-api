package platform

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"runtime"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
)

// OverviewData 平台概览响应数据
type OverviewData struct {
	RunningTime int64   `json:"runningTime"`
	Memory      uint64  `json:"memory"`
	RoomCount   int64   `json:"roomCount"`
	WorldCount  int64   `json:"worldCount"`
	UserCount   int64   `json:"userCount"`
	UidCount    int64   `json:"uidCount"`
	MaxCpu      float64 `json:"maxCpu"`
	MaxMemory   float64 `json:"maxMemory"`
	MaxNetUp    float64 `json:"maxNetUp"`
	MaxNetDown  float64 `json:"maxNetDown"`
}

// Overview 平台概览
func (s *Service) Overview() service.Result {
	// 运行时间
	t := time.Since(utils.StartTime).Seconds()
	// 内存占用
	mem := utils.GetRES()
	// 房间数
	roomCount, err := s.roomDao.Count(nil)
	if err != nil {
		logger.Logger.Error("统计房间数失败")
		roomCount = 0
	}
	// 世界数
	worldCount, err := s.worldDao.Count(nil)
	if err != nil {
		logger.Logger.Error("统计世界数失败")
		worldCount = 0
	}
	// 用户数
	userCount, err := s.userDao.Count(nil)
	if err != nil {
		logger.Logger.Error("统计用户数失败")
		userCount = 0
	}
	// uid数
	uidCount, err := s.uidMapDao.Count(nil)
	if err != nil {
		logger.Logger.Error("统计用户数失败")
		uidCount = 0
	}
	// 1小时cpu内存网络最大值
	cache.SystemMetricsMutex.RLock()
	systemMetricsLength := len(cache.SystemMetrics)
	reqLength := 60
	var systemMetricsData []cache.SysMetrics
	if systemMetricsLength > reqLength {
		systemMetricsData = cache.SystemMetrics[systemMetricsLength-reqLength:]
	} else {
		systemMetricsData = cache.SystemMetrics
	}
	cache.SystemMetricsMutex.RUnlock()
	var maxCpu, maxMemory, maxNetUp, maxNetDown float64
	for _, m := range systemMetricsData {
		if m.Cpu > maxCpu {
			maxCpu = m.Cpu
		}
		if m.Memory > maxMemory {
			maxMemory = m.Memory
		}
		if m.NetUplink > maxNetUp {
			maxNetUp = m.NetUplink
		}
		if m.NetDownlink > maxNetDown {
			maxNetDown = m.NetDownlink
		}
	}

	// TODO 玩家数最多的的房间Top3

	data := OverviewData{
		RunningTime: int64(t),
		Memory:      mem,
		RoomCount:   roomCount,
		WorldCount:  worldCount,
		UserCount:   userCount,
		UidCount:    uidCount,
		MaxCpu:      maxCpu,
		MaxMemory:   maxMemory,
		MaxNetUp:    maxNetUp,
		MaxNetDown:  maxNetDown,
	}

	return service.Success(data)
}

// OSInfo 系统信息
type OSInfo struct {
	Architecture    string
	OS              string
	CPUModel        string
	CPUCores        int
	CPUDetail       []cpu.InfoStat
	MemorySize      uint64
	Platform        string
	PlatformVersion string
	Uptime          uint64
}

func getOSInfo() (*OSInfo, error) {
	architecture := runtime.GOARCH

	// 获取CPU信息
	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, err
	}
	cpuModel := cpuInfo[0].ModelName
	cpuCount, _ := cpu.Counts(true)
	cpuCore := cpuCount
	cpuDetail := cpuInfo

	// 获取内存信息
	virtualMemory, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	memorySize := virtualMemory.Total

	// 获取主机信息
	hostInfo, err := host.Info()
	if err != nil {
		return nil, err
	}
	platformVersion := hostInfo.PlatformVersion
	platform := hostInfo.Platform
	uptime := hostInfo.Uptime
	osName := hostInfo.OS
	// 返回系统信息
	return &OSInfo{
		Architecture:    architecture,
		OS:              osName,
		CPUModel:        cpuModel,
		CPUCores:        cpuCore,
		CPUDetail:       cpuDetail,
		MemorySize:      memorySize,
		Platform:        platform,
		Uptime:          uptime,
		PlatformVersion: platformVersion,
	}, nil
}

// OSInfoGet 获取系统信息
func (s *Service) OSInfoGet() service.Result {
	osInfo, err := getOSInfo()
	if err != nil {
		logger.Logger.Errorf("获取系统信息失败, err: %v", err)
		return service.Fail(constant.CodePlatformOSInfoFail)
	}

	return service.Success(osInfo)
}

// Metrics 系统监控数据，timeRange 单位为小时
func (s *Service) Metrics(timeRange int) service.Result {
	cache.SystemMetricsMutex.RLock()
	systemMetricsLength := len(cache.SystemMetrics)
	reqLength := timeRange * 60
	if reqLength <= 0 {
		reqLength = 60 // 默认1小时
	}

	var result service.Result
	if systemMetricsLength > reqLength {
		result = service.Success(cache.SystemMetrics[systemMetricsLength-reqLength:])
	} else {
		result = service.Success(cache.SystemMetrics)
	}
	cache.SystemMetricsMutex.RUnlock()

	return result
}
