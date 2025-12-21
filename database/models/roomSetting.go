package models

type RoomSetting struct {
	RoomID                    int    `gorm:"primaryKey;not null;column:room_id" json:"roomID"`
	BackupEnable              bool   `gorm:"column:backup_enable" json:"backupEnable"`
	BackupSetting             string `gorm:"column:backup_setting" json:"backupSetting"`
	BackupCleanEnable         bool   `gorm:"column:backup_clean_enable" json:"backupCleanEnable"`
	BackupCleanSetting        int    `gorm:"column:backup_clean_setting" json:"backupCleanSetting"`
	RestartEnable             bool   `gorm:"column:restart_enable" json:"restartEnable"`
	RestartSetting            string `gorm:"column:restart_setting" json:"restartSetting"`
	AnnounceSetting           string `gorm:"column:announce_setting" json:"announceSetting"`
	KeepaliveEnable           bool   `gorm:"column:keepalive_enable" json:"keepaliveEnable"`
	KeepaliveSetting          int    `gorm:"column:keepalive_setting" json:"keepaliveSetting"`
	ScheduledStartStopEnable  bool   `gorm:"column:scheduled_start_stop_enable" json:"scheduledStartStopEnable"`
	ScheduledStartStopSetting string `gorm:"column:scheduled_start_stop_setting" json:"scheduledStartStopSetting"`
	TickRate                  int    `gorm:"column:tick_rate" json:"tickRate"`
	StartType                 string `gorm:"column:start_type" json:"startType"`
	CustomIP                  string `gorm:"column:custom_ip" json:"customIP"`
	CustomPort                int    `gorm:"column:custom_port" json:"customPort"`
}

func (RoomSetting) TableName() string {
	return "room_settings"
}
