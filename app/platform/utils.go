package platform

import (
	"dst-management-platform-api/database/dao"
	"github.com/shirou/gopsutil/v3/process"
	"os"
)

type Handler struct {
	userDao   *dao.UserDAO
	systemDao *dao.SystemDAO
}

func NewHandler(userDao *dao.UserDAO, systemDao *dao.SystemDAO) *Handler {
	return &Handler{
		userDao:   userDao,
		systemDao: systemDao,
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
