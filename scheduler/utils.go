package scheduler

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"fmt"
	"github.com/go-co-op/gocron"
	"sync"
	"time"
)

var (
	Scheduler   = gocron.NewScheduler(time.Local)
	jobMutex    sync.RWMutex
	currentJobs = make(map[string]*gocron.Job)
	DBHandler   *Handler
)

type JobConfig struct {
	Name     string
	Func     interface{}
	Args     []interface{}
	TimeType string
	Interval int
	DayAt    string
}

type Handler struct {
	roomDao          *dao.RoomDAO
	worldDao         *dao.WorldDAO
	roomSettingDao   *dao.RoomSettingDAO
	globalSettingDao *dao.GlobalSettingDAO
	uidMapDao        *dao.UidMapDAO
}

func newDBHandler(roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO, globalSettingDao *dao.GlobalSettingDAO, uidMapDao *dao.UidMapDAO) *Handler {
	return &Handler{
		roomDao:          roomDao,
		worldDao:         worldDao,
		roomSettingDao:   roomSettingDao,
		globalSettingDao: globalSettingDao,
		uidMapDao:        uidMapDao,
	}
}

func registerJobs() {
	for _, job := range Jobs {
		err := UpdateJob(&job)
		if err != nil {
			logger.Logger.Error("注册定时任务失败", "err", err)
			panic("注册定时任务失败")
		}
		logger.Logger.Info(fmt.Sprintf("定时任务[%s]注册成功", job.Name))
	}
}

func fetchGameInfo(roomID int) (*models.Room, *[]models.World, *models.RoomSetting, error) {
	room, err := DBHandler.roomDao.GetRoomByID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}
	worlds, err := DBHandler.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}
	roomSetting, err := DBHandler.roomSettingDao.GetRoomSettingsByRoomID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}

	return room, worlds, roomSetting, nil
}
