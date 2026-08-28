package dao

import (
	"dst-management-platform-api/database/models"

	"gorm.io/gorm"
)

type ModInfoDAO struct {
	BaseDAO[models.ModInfo]
}

func NewModInfoDAO(db *gorm.DB) *ModInfoDAO {
	return &ModInfoDAO{
		BaseDAO: *NewBaseDAO[models.ModInfo](db),
	}
}

func (d *ModInfoDAO) GetModInfoByID(id int, languages ...string) (*models.ModInfo, error) {
	var modInfo models.ModInfo
	query := d.db.Where("id = ?", id)
	if len(languages) > 0 {
		query = query.Where("language = ?", languages[0])
	}
	err := query.First(&modInfo).Error

	return &modInfo, err
}

func (d *ModInfoDAO) GetModInfoByIDs(ids []int, languages ...string) ([]models.ModInfo, error) {
	var modInfos []models.ModInfo
	if len(ids) == 0 {
		return modInfos, nil
	}
	query := d.db.Where("id IN ?", ids)
	if len(languages) > 0 {
		query = query.Where("language = ?", languages[0])
	}
	err := query.Find(&modInfos).Error

	return modInfos, err
}

func (d *ModInfoDAO) UpdateModInfo(modInfo *models.ModInfo) error {
	err := d.db.Save(modInfo).Error

	return err
}
