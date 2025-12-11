package models

type GlobalSetting struct {
	ID                 int    `gorm:"primaryKey;not null;column:id" json:"id"`
	PlayerGetFrequency int    `gorm:"column:player_get_frequency" json:"playerGetFrequency"`
	UIDMaintainEnable  bool   `gorm:"column:uid_maintain_enable" json:"UIDMaintainEnable"`
	UIDMaintainSetting int    `gorm:"column:uid_maintain_setting" json:"UIDMaintainSetting"`
	SysMetricsEnable   bool   `gorm:"column:sys_metrics_enable" json:"sysMetricsEnable"`
	SysMetricsSetting  int    `gorm:"column:sys_metrics_setting" json:"sysMetricsSetting"`
	AutoUpdateEnable   bool   `gorm:"column:auto_update_enable" json:"autoUpdateEnable"`
	AutoUpdateSetting  string `gorm:"column:auto_update_setting" json:"autoUpdateSetting"`
}

func (GlobalSetting) TableName() string {
	return "global_settings"
}
