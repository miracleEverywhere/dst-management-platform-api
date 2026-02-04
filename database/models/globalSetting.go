package models

type GlobalSetting struct {
	ID                 int    `gorm:"primaryKey;not null;column:id" json:"id"`
	PlayerGetFrequency int    `gorm:"column:player_get_frequency" json:"playerGetFrequency"`
	UIDMaintainEnable  bool   `gorm:"column:uid_maintain_enable" json:"UIDMaintainEnable"`
	SysMetricsEnable   bool   `gorm:"column:sys_metrics_enable" json:"sysMetricsEnable"`
	SysMetricsSetting  int    `gorm:"column:sys_metrics_setting" json:"sysMetricsSetting"`
	AutoUpdateEnable   bool   `gorm:"column:auto_update_enable" json:"autoUpdateEnable"`   // 自动更新是否开启
	AutoUpdateSetting  string `gorm:"column:auto_update_setting" json:"autoUpdateSetting"` // 自动更新时间设置
	AutoUpdateRestart  bool   `gorm:"column:auto_update_restart" json:"autoUpdateRestart"` // 自动更新后是否重启，按理说要加在Setting中，但是太麻烦了
}

func (GlobalSetting) TableName() string {
	return "global_settings"
}
