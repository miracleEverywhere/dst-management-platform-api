package scheduler

import (
	"github.com/go-co-op/gocron"
	"time"
)

var Scheduler = gocron.NewScheduler(time.UTC)

// InitTasks 初始化定时任务
func InitTasks() {
	// 每10秒执行一次任务
	_, err := Scheduler.Every(15).Seconds().Do(setPlayer2DB)
	if err != nil {
		return
	}
}
