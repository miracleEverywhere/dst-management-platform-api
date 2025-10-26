package models

type GlobalSetting struct {
	ID                    int    `gorm:"primaryKey;not null"`
	PlayerGetFrequency    int    `json:"playerGetFrequency"`
	UIDMaintainEnable     bool   `json:"UIDMaintainEnable"`
	UIDMaintainSetting    string `json:"UIDMaintainSetting"`
	SysMetricsEnable      bool   `json:"sysMetricsEnable"`
	SysMetricsSetting     string `json:"sysMetricsSetting"`
	AutoUpdateEnable      bool   `json:"autoUpdateEnable"`
	AutoUpdateSetting     string `json:"autoUpdateSetting"`
	PlayerUpdateModEnable bool   `json:"playerUpdateModEnable"`
}

func (GlobalSetting) TableName() string {
	return "global_settings"
}
