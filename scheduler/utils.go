package scheduler

import (
	"dst-management-platform-api/database/models"
	"strconv"
	"strings"
)

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

func GetJobs(roomID int, jobType string) []string {
	var n []string

	for _, job := range Jobs {
		if strings.HasSuffix(job.Name, jobType) {
			s := strings.Split(job.Name, "-")
			if s[0] == strconv.Itoa(roomID) {
				n = append(n, job.Name)
			}
		}
	}

	if n == nil {
		return []string{}
	}

	return n
}
