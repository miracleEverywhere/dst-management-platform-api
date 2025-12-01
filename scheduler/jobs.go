package scheduler

import (
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
)

var jobs []JobConfig

func initJobs() {
	var globalSetting models.GlobalSetting
	err := DBHandler.globalSettingDao.GetGlobalSetting(&globalSetting)
	if err != nil {
		logger.Logger.Error("初始化定时任务失败", "err", err)
		panic("初始化定时任务失败")
	}

	// 全局定时任务
	// players online
	jobs = append(jobs, JobConfig{
		Name:     "onlinePlayerGet",
		Func:     onlinePlayerGet,
		Args:     nil,
		TimeType: "second",
		Interval: globalSetting.PlayerGetFrequency,
		DayAt:    "",
	})

}
