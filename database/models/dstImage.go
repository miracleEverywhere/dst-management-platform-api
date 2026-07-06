package models

type DstImage struct {
	ID       int    `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Prefab   string `gorm:"column:prefab" json:"prefab"`
	Category string `gorm:"column:category" json:"category"`
	NameZh   string `gorm:"column:name_zh" json:"name_zh"`
	NameEn   string `gorm:"column:name_en" json:"name_en"`
	Image    string `gorm:"column:image" json:"image,omitempty"`
}

func (DstImage) TableName() string {
	return "dst_images"
}
