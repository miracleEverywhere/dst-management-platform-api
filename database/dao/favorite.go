package dao

import (
	"dst-management-platform-api/database/models"

	"gorm.io/gorm"
)

type FavoriteDAO struct {
	BaseDAO[models.Favorite]
}

func NewFavoriteDAO(db *gorm.DB) *FavoriteDAO {
	return &FavoriteDAO{
		BaseDAO: *NewBaseDAO[models.Favorite](db),
	}
}

func (d *FavoriteDAO) CreateFavorite(favorite *models.Favorite) error {
	return d.db.Create(favorite).Error
}

func (d *FavoriteDAO) UpdateFavorite(favorite *models.Favorite) error {
	return d.db.Save(favorite).Error
}

func (d *FavoriteDAO) GetFavoriteByID(id int, username, favoriteType string) (*models.Favorite, error) {
	var favorite models.Favorite
	err := d.db.
		Where("id = ? AND username = ? AND type = ?", id, username, favoriteType).
		First(&favorite).Error

	return &favorite, err
}

func (d *FavoriteDAO) ListFavorites(username, favoriteType string, page, pageSize int) (*PaginatedResult[models.Favorite], error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	var (
		favorites []models.Favorite
		total     int64
	)

	query := d.db.Model(&models.Favorite{}).
		Where("username = ? AND type = ?", username, favoriteType)

	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	err := query.
		Order("created_at DESC, id DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Find(&favorites).Error

	return &PaginatedResult[models.Favorite]{
		Data:       favorites,
		Page:       page,
		PageSize:   pageSize,
		TotalCount: total,
	}, err
}

// DeleteFavorite 删除指定用户的收藏，返回删除行数
func (d *FavoriteDAO) DeleteFavorite(id int, username, favoriteType string) (int64, error) {
	result := d.db.
		Where("id = ? AND username = ? AND type = ?", id, username, favoriteType).
		Delete(&models.Favorite{})

	return result.RowsAffected, result.Error
}
