package dao

import (
	"dst-management-platform-api/database/models"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CloudFavoriteMapDAO struct {
	BaseDAO[models.CloudFavoriteMap]
}

func NewCloudFavoriteMapDAO(db *gorm.DB) *CloudFavoriteMapDAO {
	return &CloudFavoriteMapDAO{
		BaseDAO: *NewBaseDAO[models.CloudFavoriteMap](db),
	}
}

func (d *CloudFavoriteMapDAO) GetByLocalID(username, localID string) (*models.CloudFavoriteMap, error) {
	var item models.CloudFavoriteMap
	err := d.db.Where("username = ? AND local_favorite_id = ?", username, localID).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &item, nil
	}

	return &item, err
}

func (d *CloudFavoriteMapDAO) GetByCloudID(username, cloudID string) (*models.CloudFavoriteMap, error) {
	var item models.CloudFavoriteMap
	err := d.db.Where("username = ? AND cloud_favorite_id = ?", username, cloudID).First(&item).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &item, nil
	}

	return &item, err
}

func (d *CloudFavoriteMapDAO) ListByUsername(username string) ([]models.CloudFavoriteMap, error) {
	var items []models.CloudFavoriteMap
	err := d.db.Where("username = ?", username).Find(&items).Error

	return items, err
}

// upsertCloudFavoriteMap 以本地收藏 ID 为唯一键写入映射，可在事务内复用。
func upsertCloudFavoriteMap(tx *gorm.DB, item *models.CloudFavoriteMap) error {
	return tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "local_favorite_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"username", "cloud_favorite_id", "last_cloud_update", "updated_at"}),
	}).Create(item).Error
}

// Upsert 以本地收藏 ID 为唯一键写入映射，云端 ID 变化时覆盖。
func (d *CloudFavoriteMapDAO) Upsert(item *models.CloudFavoriteMap) error {
	return upsertCloudFavoriteMap(d.db, item)
}

func (d *CloudFavoriteMapDAO) DeleteByUsername(username string) error {
	return d.db.Where("username = ?", username).Delete(&models.CloudFavoriteMap{}).Error
}
