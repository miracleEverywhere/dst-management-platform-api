package models

import "time"

// MapImageVersion 是地图图片的编码版本。地形贴图缩放/压缩策略变化时递增，
// 旧缓存行不会被查询命中，下一次请求会重新生成并覆盖。
const MapImageVersion = 3

// MapImage 缓存地图用到的图标/地形贴图（base64 PNG），按游戏版本隔离。
// Kind 目前有 icon（minimap 图标）和 tile（地形噪声贴图）两种。
type MapImage struct {
	ID        int       `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Name      string    `gorm:"column:name;size:128;not null;uniqueIndex:idx_map_images_key,priority:1" json:"name"`
	Kind      string    `gorm:"column:kind;size:16;not null;uniqueIndex:idx_map_images_key,priority:2" json:"kind"`
	GameBuild string    `gorm:"column:game_build;size:32;not null;uniqueIndex:idx_map_images_key,priority:3" json:"gameBuild"`
	Version   int       `gorm:"column:version;not null;default:0" json:"-"`
	Image     string    `gorm:"column:image;type:text" json:"image"`
	Width     int       `gorm:"column:width" json:"width"`
	Height    int       `gorm:"column:height" json:"height"`
	SHA256    string    `gorm:"column:sha256;size:64" json:"sha256"`
	CreatedAt time.Time `gorm:"column:created_at" json:"-"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"-"`
}

func (MapImage) TableName() string {
	return "map_images"
}
