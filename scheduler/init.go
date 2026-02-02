package scheduler

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/logger"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-co-op/gocron"
)

// Start 开启定时任务
func Start(roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO, globalSettingDao *dao.GlobalSettingDAO, uidMapDao *dao.UidMapDAO) {
	DBHandler = newDBHandler(roomDao, worldDao, roomSettingDao, globalSettingDao, uidMapDao)
	initJobs()
	registerJobs()
	go Scheduler.StartAsync()
}

// UpdateJob 更新指定任务
func UpdateJob(jobConfig *JobConfig) error {
	jobMutex.Lock()
	defer jobMutex.Unlock()

	// 移除现有任务
	if job, exists := currentJobs[jobConfig.Name]; exists {
		Scheduler.RemoveByReference(job)
		delete(currentJobs, jobConfig.Name)
		logger.Logger.Debug(fmt.Sprintf("发现已存在定时任务[%s]，移除", jobConfig.Name))
	}

	// 添加新任务
	var job *gocron.Job
	var err error

	switch jobConfig.TimeType {
	case SecondType:
		job, err = Scheduler.Every(jobConfig.Interval).Seconds().Do(jobConfig.Func, jobConfig.Args...)
	case MinuteType:
		job, err = Scheduler.Every(jobConfig.Interval).Minutes().Do(jobConfig.Func, jobConfig.Args...)
	case HourType:
		job, err = Scheduler.Every(jobConfig.Interval).Hours().Do(jobConfig.Func, jobConfig.Args...)
	case DayType:
		job, err = Scheduler.Every(1).Day().At(jobConfig.DayAt).Do(jobConfig.Func, jobConfig.Args...)
	default:
		return fmt.Errorf("未知的时间类型: %s, 任务名: %s", jobConfig.TimeType, jobConfig.Name)
	}

	logger.Logger.Debug("正在创建定时任务", "name", jobConfig.Name, "type", jobConfig.TimeType)

	if err != nil {
		return err
	}

	currentJobs[jobConfig.Name] = job
	logger.Logger.Debug(fmt.Sprintf("定时任务[%s]已写入任务池", jobConfig.Name))

	return nil
}

// DeleteJob 删除指定任务
func DeleteJob(jobName string) {
	jobMutex.Lock()
	defer jobMutex.Unlock()

	if job, exists := currentJobs[jobName]; exists {
		Scheduler.RemoveByReference(job)
		delete(currentJobs, jobName)
		logger.Logger.Debug(fmt.Sprintf("删除定时任务[%s]", jobName))
	}
}

// GetJobsByType 根据任务名获取定时任务
func GetJobsByType(roomID int, jobType string) []string {
	jobMutex.Lock()
	defer jobMutex.Unlock()

	var n []string
	for jobName, _ := range currentJobs {
		logger.Logger.Debug("定时任务名", "jobName", jobName, "jobType", jobType)
		if strings.HasSuffix(jobName, jobType) {
			s := strings.Split(jobName, "-")
			if s[0] == strconv.Itoa(roomID) {
				n = append(n, jobName)
			}
		}
	}

	if n == nil {
		return []string{}
	}

	return n
}

// GetJobsByRoomID 根据房间ID获取定时任务
func GetJobsByRoomID(roomID int) []string {
	jobMutex.Lock()
	defer jobMutex.Unlock()

	var n []string
	for jobName, _ := range currentJobs {
		jobNameParts := strings.Split(jobName, "-")
		if jobNameParts[0] == strconv.Itoa(roomID) {
			n = append(n, jobName)
		}
	}

	if n == nil {
		return []string{}
	}

	return n
}
