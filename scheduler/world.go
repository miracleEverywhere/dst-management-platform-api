package scheduler

import (
	"dst-management-platform-api/logger"
	"fmt"
)

func printTest(x int) {
	i, err := DBHandler.roomDao.Count(nil)
	if err != nil {
		return
	}
	logger.Logger.Info(fmt.Sprintf("%d", i))
}
