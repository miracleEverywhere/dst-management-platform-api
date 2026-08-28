package models

import "gorm.io/datatypes"

type ModInfo struct {
	ID              int                        `json:"id" gorm:"column:id;primaryKey"`
	Language        string                     `json:"-" gorm:"column:language"`
	Name            string                     `json:"name" gorm:"column:name"`
	Size            string                     `json:"size" gorm:"column:size"`
	Tags            datatypes.JSONType[[]Tags] `json:"tags" gorm:"column:tags;type:json"`
	PreviewUrl      string                     `json:"preview_url" gorm:"column:preview_url"`
	FileDescription string                     `json:"file_description" gorm:"column:file_description"`
	FileUrl         string                     `json:"file_url" gorm:"column:file_url"`
	VoteData        VoteData                   `json:"vote_data" gorm:"embedded"`
	DownloadedReady bool                       `json:"downloadedReady" gorm:"column:downloaded_ready"`
	TimeCreated     int                        `json:"time_created" gorm:"column:time_created"`
	TimeUpdated     int                        `json:"time_updated" gorm:"column:time_updated"`
	Subscriptions   int                        `json:"subscriptions" gorm:"column:subscriptions"`
	CacheTimestamp  int64                      `json:"-" gorm:"column:cache_timestamp"`
}

type Tags struct {
	Tag         string `json:"tag"`
	DisplayName string `json:"display_name"`
}

type VoteData struct {
	Score     float64 `json:"score" gorm:"column:score"`
	VotesUp   int     `json:"votes_up" gorm:"column:votes_up"`
	VotesDown int     `json:"votes_down" gorm:"column:votes_down"`
}

func (ModInfo) TableName() string {
	return "mod_info"
}
