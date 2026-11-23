package dao

import (
	"dst-management-platform-api/database/models"

	"gorm.io/gorm"
)

// BannedModDAO 全局模组屏蔽名单的数据访问。
// 名单是全局的，不区分房间与世界。
type BannedModDAO struct {
	BaseDAO[models.BannedMod]
}

func NewBannedModDAO(db *gorm.DB) *BannedModDAO {
	return &BannedModDAO{
		BaseDAO: *NewBaseDAO[models.BannedMod](db),
	}
}

// GetModIDs 读取全部被屏蔽的模组 ID，按 ID 升序返回，无记录时返回空切片
func (d *BannedModDAO) GetModIDs() ([]int, error) {
	modIDs := []int{}
	err := d.db.Model(&models.BannedMod{}).Order("mod_id ASC").Pluck("mod_id", &modIDs).Error
	if err != nil {
		return nil, err
	}

	return modIDs, nil
}

// Exist 判断单个模组是否在屏蔽名单中
func (d *BannedModDAO) Exist(modID int) (bool, error) {
	count, err := d.Count("mod_id = ?", modID)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// Replace 用给定 ID 整体覆盖屏蔽名单。
// 清空与写入放在同一事务里，避免中途失败留下半份名单。
func (d *BannedModDAO) Replace(modIDs []int) error {
	return d.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("1 = 1").Delete(&models.BannedMod{}).Error; err != nil {
			return err
		}
		if len(modIDs) == 0 {
			return nil
		}

		rows := make([]models.BannedMod, 0, len(modIDs))
		for _, modID := range modIDs {
			rows = append(rows, models.BannedMod{ModID: modID})
		}

		return tx.Create(&rows).Error
	})
}
