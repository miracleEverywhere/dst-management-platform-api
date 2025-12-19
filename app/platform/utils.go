package platform

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"os"
	"runtime"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

type Handler struct {
	userDao          *dao.UserDAO
	roomDao          *dao.RoomDAO
	worldDao         *dao.WorldDAO
	systemDao        *dao.SystemDAO
	globalSettingDao *dao.GlobalSettingDAO
	uidMapDao        *dao.UidMapDAO
	roomSettingDao   *dao.RoomSettingDAO
}

func NewHandler(userDao *dao.UserDAO, roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, systemDao *dao.SystemDAO, globalSettingDao *dao.GlobalSettingDAO, uidMapDao *dao.UidMapDAO, roomSettingDao *dao.RoomSettingDAO) *Handler {
	return &Handler{
		userDao:          userDao,
		roomDao:          roomDao,
		worldDao:         worldDao,
		systemDao:        systemDao,
		globalSettingDao: globalSettingDao,
		uidMapDao:        uidMapDao,
		roomSettingDao:   roomSettingDao,
	}
}

func getRES() uint64 {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return 0
	}

	memoryInfo, err := p.MemoryInfo()
	if err != nil {
		return 0
	}

	return memoryInfo.RSS
}

type OSInfo struct {
	Architecture    string
	OS              string
	CPUModel        string
	CPUCores        int
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
		MemorySize:      memorySize,
		Platform:        platform,
		Uptime:          uptime,
		PlatformVersion: platformVersion,
	}, nil
}

func (h *Handler) fetchGameInfo(roomID int) (*models.Room, *[]models.World, *models.RoomSetting, error) {
	room, err := h.roomDao.GetRoomByID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}
	worlds, err := h.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}
	roomSetting, err := h.roomSettingDao.GetRoomSettingsByRoomID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}

	return room, worlds, roomSetting, nil
}
