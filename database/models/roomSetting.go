package models

type RoomSetting struct {
	RoomName                  string `gorm:"primaryKey;not null" json:"roomName"`
	BackupEnable              bool   `json:"backupEnable"`
	BackupSetting             string `json:"backupSetting"`
	BackupCleanEnable         bool   `json:"backupCleanEnable"`
	BackupCleanSetting        bool   `json:"backupCleanSetting"`
	RestartEnable             bool   `json:"restartEnable"`
	RestartSetting            string `json:"restartSetting"`
	AnnounceEnable            bool   `json:"announceEnable"`
	AnnounceSetting           string `json:"announceSetting"`
	KeepaliveEnable           bool   `json:"keepaliveEnable"`
	KeepaliveSetting          string `json:"keepaliveSetting"`
	ScheduledStartStopEnable  bool   `json:"scheduledStartStopEnable"`
	ScheduledStartStopSetting string `json:"scheduledStartStopSetting"`
	TickRate                  int    `json:"tickRate"`
	Bit64                     int    `json:"bit64"`
}

func (RoomSetting) TableName() string {
	return "room_settings"
}
