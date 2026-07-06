package dao

import (
	"dst-management-platform-api/database/models"

	"gorm.io/gorm"
)

type DstImageDAO struct {
	BaseDAO[models.DstImage]
}

func NewDstImageDAO(db *gorm.DB) *DstImageDAO {
	return &DstImageDAO{
		BaseDAO: *NewBaseDAO[models.DstImage](db),
	}
}

func (d *DstImageDAO) Categories() ([]string, error) {
	var categories []string
	err := d.db.Model(&models.DstImage{}).Distinct("category").Pluck("category", &categories).Error

	return categories, err
}

func (d *DstImageDAO) List(category string, page, pageSize int) (*PaginatedResult[models.DstImage], error) {
	images, err := d.Query(page, pageSize, "category = ?", category)

	return images, err
}

func (d *DstImageDAO) UpdateImage(img *models.DstImage) error {
	err := d.db.Save(img).Error

	return err
}

func (d *DstImageDAO) DeleteNoName() error {
	return d.db.Where("name_zh = ''").Delete(&models.DstImage{}).Error
}

func (d *DstImageDAO) DeleteAll() error {
	return d.db.Where("1 = 1").Delete(&models.DstImage{}).Error
}

func (d *DstImageDAO) InitImages(images []models.DstImage) error {
	if err := d.DeleteAll(); err != nil {
		return err
	}
	if len(images) == 0 {
		return nil
	}
	return d.db.Create(&images).Error
}
