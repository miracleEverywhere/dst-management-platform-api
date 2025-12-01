package scheduler

import (
	"dst-management-platform-api/database/dao"
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
}

func Start(roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO, globalSettingDao *dao.GlobalSettingDAO) {
	DBHandler = newDBHandler(roomDao, worldDao, roomSettingDao, globalSettingDao)
	initJobs()
	registerJobs()
	go Scheduler.StartAsync()
}

func newDBHandler(roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO, globalSettingDao *dao.GlobalSettingDAO) *Handler {
	return &Handler{
		roomDao:          roomDao,
		worldDao:         worldDao,
		roomSettingDao:   roomSettingDao,
		globalSettingDao: globalSettingDao,
	}
}

func registerJobs() {
	for _, job := range jobs {
		err := UpdateJob(&job)
		if err != nil {
			logger.Logger.Error("注册定时任务失败", "err", err)
			panic("注册定时任务失败")
		}
		logger.Logger.Info(fmt.Sprintf("定时任务[%s]注册成功", job.Name))
	}
}

// UpdateJob 更新特定任务
func UpdateJob(jobConfig *JobConfig) error {
	jobMutex.Lock()
	defer jobMutex.Unlock()

	// 移除现有任务
	if job, exists := currentJobs[jobConfig.Name]; exists {
		Scheduler.RemoveByReference(job)
		delete(currentJobs, jobConfig.Name)
		logger.Logger.Debug(fmt.Sprintf("发现已存在定时任务[%s]，移除。。。", jobConfig.Name))
	}

	// 添加新任务
	var job *gocron.Job
	var err error

	switch jobConfig.TimeType {
	case "second":
		job, err = Scheduler.Every(jobConfig.Interval).Seconds().Do(jobConfig.Func, jobConfig.Args...)
	case "minute":
		job, err = Scheduler.Every(jobConfig.Interval).Minutes().Do(jobConfig.Func, jobConfig.Args...)
	case "hour":
		job, err = Scheduler.Every(jobConfig.Interval).Hours().Do(jobConfig.Func, jobConfig.Args...)
	case "day":
		job, err = Scheduler.Every(1).Day().At(jobConfig.DayAt).Do(jobConfig.Func, jobConfig.Args...)
	default:
		return fmt.Errorf("未知的时间类型: %s, 任务名: %s", jobConfig.TimeType, jobConfig.Name)
	}

	logger.Logger.Debug("正在创建定时任务", "name", jobConfig.Name, "type", jobConfig.TimeType)

	if err != nil {
		return err
	}

	currentJobs[jobConfig.Name] = job
	logger.Logger.Debug(fmt.Sprintf("定时任务[%s]已写入map", jobConfig.Name))

	return nil
}
