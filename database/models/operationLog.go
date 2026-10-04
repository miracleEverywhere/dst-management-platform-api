package models

import "time"

// OperationLog 后台操作流水（启停/更新/重置等），由 opmgr 完成时落库
type OperationLog struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	OpID      string     `gorm:"column:op_id;size:64;index" json:"opID"`
	RoomID    int        `gorm:"column:room_id;index" json:"roomID"`
	Type      string     `gorm:"column:type;size:32" json:"type"`
	State     string     `gorm:"column:state;size:16" json:"state"`
	Stage     string     `gorm:"column:stage;size:255" json:"stage"`
	Error     string     `gorm:"column:error;size:512" json:"error"`
	StartedAt time.Time  `gorm:"column:started_at" json:"startedAt"`
	EndedAt   *time.Time `gorm:"column:ended_at" json:"endedAt"`
	By        string     `gorm:"column:by;size:64" json:"by"`
}
