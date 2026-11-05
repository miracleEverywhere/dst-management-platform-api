package models

type Favorite struct {
	ID        string `gorm:"primaryKey;column:id" json:"id"`
	Username  string `gorm:"column:username;not null;index:idx_favorites_username_type,priority:1" json:"username"`
	Type      string `gorm:"column:type;not null;default:dst;index:idx_favorites_username_type,priority:2" json:"type"`
	Data      string `gorm:"column:data;not null" json:"-"`
	Remark    string `gorm:"column:remark" json:"remark"`
	CreatedAt int64  `gorm:"column:created_at;not null;default:0;autoCreateTime:false" json:"createdAt"`
	UpdatedAt int64  `gorm:"column:updated_at;not null;default:0;autoUpdateTime:false" json:"updatedAt"`
	// DeletedAt 软删除时间戳，0 表示未删除；云同步需要把本地删除作为 tombstone 上传
	DeletedAt int64 `gorm:"column:deleted_at;not null;default:0" json:"-"`
}

func (Favorite) TableName() string {
	return "favorites"
}
