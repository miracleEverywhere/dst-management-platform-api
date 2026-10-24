package dao

import (
	"dst-management-platform-api/database/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type MapImageDAO struct {
	BaseDAO[models.MapImage]
}

func NewMapImageDAO(db *gorm.DB) *MapImageDAO {
	return &MapImageDAO{
		BaseDAO: *NewBaseDAO[models.MapImage](db),
	}
}

// GetMany 批量读取某游戏版本下指定 kind 的图片，返回 name -> 记录。
func (d *MapImageDAO) GetMany(kind, gameBuild string, names []string) (map[string]models.MapImage, error) {
	result := make(map[string]models.MapImage, len(names))
	for start := 0; start < len(names); start += 400 {
		end := start + 400
		if end > len(names) {
			end = len(names)
		}
		var rows []models.MapImage
		err := d.db.
			Where("kind = ? AND game_build = ? AND version = ? AND name IN ?", kind, gameBuild, models.MapImageVersion, names[start:end]).
			Find(&rows).Error
		if err != nil {
			return result, err
		}
		for _, row := range rows {
			result[row.Name] = row
		}
	}
	return result, nil
}

func (d *MapImageDAO) Get(name, kind, gameBuild string) (*models.MapImage, error) {
	var image models.MapImage
	err := d.db.Where("name = ? AND kind = ? AND game_build = ? AND version = ?",
		name, kind, gameBuild, models.MapImageVersion).First(&image).Error
	if err != nil {
		return nil, err
	}
	return &image, nil
}

// Upsert 写入或覆盖一张图片。图片内容由存档/游戏资源决定，重新生成时直接覆盖。
func (d *MapImageDAO) Upsert(image *models.MapImage) error {
	return d.UpsertMany([]models.MapImage{*image})
}

// UpsertMany 批量写入/覆盖图片，主要用于一次解包整个图集后的缓存。
func (d *MapImageDAO) UpsertMany(images []models.MapImage) error {
	if len(images) == 0 {
		return nil
	}
	return d.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "name"},
			{Name: "kind"},
			{Name: "game_build"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"version", "image", "width", "height", "sha256", "updated_at",
		}),
	}).CreateInBatches(&images, 50).Error
}
