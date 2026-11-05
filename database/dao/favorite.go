package dao

import (
	"dst-management-platform-api/database/models"

	"github.com/google/uuid"
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

// CreateFavorite 创建收藏，未指定 ID 时生成 UUID
func (d *FavoriteDAO) CreateFavorite(favorite *models.Favorite) error {
	if favorite.ID == "" {
		favorite.ID = uuid.NewString()
	}

	return d.db.Create(favorite).Error
}

func (d *FavoriteDAO) UpdateFavorite(favorite *models.Favorite) error {
	return d.db.Save(favorite).Error
}

// GetFavoriteByID 查询未删除的收藏
func (d *FavoriteDAO) GetFavoriteByID(id, username, favoriteType string) (*models.Favorite, error) {
	var favorite models.Favorite
	err := d.db.
		Where("id = ? AND username = ? AND type = ? AND deleted_at = 0", id, username, favoriteType).
		First(&favorite).Error

	return &favorite, err
}

// GetFavoriteByIDIncludingDeleted 查询收藏，包含已软删除的记录，供云同步使用
func (d *FavoriteDAO) GetFavoriteByIDIncludingDeleted(id, username, favoriteType string) (*models.Favorite, error) {
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
		Where("username = ? AND type = ? AND deleted_at = 0", username, favoriteType)

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

// ListAllFavorites 查询指定用户的全部收藏，不分页，供云同步使用。
func (d *FavoriteDAO) ListAllFavorites(username, favoriteType string, includeDeleted bool) ([]models.Favorite, error) {
	var favorites []models.Favorite
	query := d.db.Where("username = ? AND type = ?", username, favoriteType)
	if !includeDeleted {
		query = query.Where("deleted_at = 0")
	}

	err := query.Order("updated_at DESC, id DESC").Find(&favorites).Error

	return favorites, err
}

// DeleteFavorite 软删除指定用户的收藏，返回更新行数。
// 软删除用于向云端传播删除；本地列表会过滤 deleted_at != 0 的记录。
func (d *FavoriteDAO) DeleteFavorite(id, username, favoriteType string, deletedAt int64) (int64, error) {
	result := d.db.Model(&models.Favorite{}).
		Where("id = ? AND username = ? AND type = ? AND deleted_at = 0", id, username, favoriteType).
		Updates(map[string]any{
			"deleted_at": deletedAt,
			"updated_at": deletedAt,
		})

	return result.RowsAffected, result.Error
}

// UpdateFavoriteFromCloud 用云端快照覆盖本地收藏，并恢复软删除状态。
func (d *FavoriteDAO) UpdateFavoriteFromCloud(id, username, data, remark string, updatedAt int64) error {
	result := d.db.Model(&models.Favorite{}).
		Where("id = ? AND username = ?", id, username).
		Updates(map[string]any{
			"data":       data,
			"remark":     remark,
			"updated_at": updatedAt,
			"deleted_at": 0,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// SoftDeleteFavoriteAt 按云端 tombstone 的时间戳软删除本地收藏。
func (d *FavoriteDAO) SoftDeleteFavoriteAt(id, username string, deletedAt int64) error {
	result := d.db.Model(&models.Favorite{}).
		Where("id = ? AND username = ? AND deleted_at = 0", id, username).
		Updates(map[string]any{
			"deleted_at": deletedAt,
			"updated_at": deletedAt,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
