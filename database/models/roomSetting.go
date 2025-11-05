package models

type RoomSetting struct {
	RoomID                    int    `gorm:"primaryKey;not null" json:"roomID"`
	BackupEnable              bool   `json:"backupEnable"`
	BackupSetting             string `json:"backupSetting"`
	BackupCleanEnable         bool   `json:"backupCleanEnable"`
	BackupCleanSetting        int    `json:"backupCleanSetting"`
	RestartEnable             bool   `json:"restartEnable"`
	RestartSetting            string `json:"restartSetting"`
	AnnounceEnable            bool   `json:"announceEnable"`
	AnnounceSetting           string `json:"announceSetting"`
	KeepaliveEnable           bool   `json:"keepaliveEnable"`
	KeepaliveSetting          int    `json:"keepaliveSetting"`
	ScheduledStartStopEnable  bool   `json:"scheduledStartStopEnable"`
	ScheduledStartStopSetting string `json:"scheduledStartStopSetting"`
	TickRate                  int    `json:"tickRate"`
	StartType                 string `json:"startType"`
}

func (RoomSetting) TableName() string {
	return "room_settings"
}
