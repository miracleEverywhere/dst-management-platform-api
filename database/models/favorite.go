package models

type Favorite struct {
	ID        int    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Username  string `gorm:"column:username;not null;index:idx_favorites_username_type,priority:1" json:"username"`
	Type      string `gorm:"column:type;not null;default:dst;index:idx_favorites_username_type,priority:2" json:"type"`
	Data      string `gorm:"column:data;not null" json:"-"`
	Remark    string `gorm:"column:remark" json:"remark"`
	CreatedAt int64  `gorm:"column:created_at;not null;default:0;autoCreateTime:false" json:"createdAt"`
	UpdatedAt int64  `gorm:"column:updated_at;not null;default:0;autoUpdateTime:false" json:"updatedAt"`
}

func (Favorite) TableName() string {
	return "favorites"
}
