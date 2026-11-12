package dao

import (
	"dst-management-platform-api/database/models"
	"errors"

	"gorm.io/gorm"
)

type CloudLinkDAO struct {
	BaseDAO[models.CloudLink]
}

func NewCloudLinkDAO(db *gorm.DB) *CloudLinkDAO {
	return &CloudLinkDAO{
		BaseDAO: *NewBaseDAO[models.CloudLink](db),
	}
}

// GetByUsername 查询本地用户的云绑定，未绑定时返回空结构体与 nil 错误。
func (d *CloudLinkDAO) GetByUsername(username string) (*models.CloudLink, error) {
	var link models.CloudLink
	err := d.db.Where("username = ?", username).First(&link).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &link, nil
	}

	return &link, err
}

// GetByCloudAccountID 按云账号 ID 查询绑定，用于云账号登录。
func (d *CloudLinkDAO) GetByCloudAccountID(accountID int64) (*models.CloudLink, error) {
	var link models.CloudLink
	err := d.db.Where("cloud_account_id = ?", accountID).First(&link).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &link, nil
	}

	return &link, err
}

// GetByIdentity 按云用户名、云邮箱或本地用户名查询绑定，用于云服务不可用时的本地回退。
func (d *CloudLinkDAO) GetByIdentity(identity string) (*models.CloudLink, error) {
	var link models.CloudLink
	err := d.db.
		Where("cloud_username = ? OR LOWER(cloud_email) = LOWER(?) OR username = ?", identity, identity, identity).
		First(&link).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return &link, nil
	}

	return &link, err
}

// SaveLink 新增或更新绑定关系，以本地用户名为唯一键。
// 已存在的绑定只做 UPDATE：绑定被并发解绑删除时 UPDATE 影响 0 行即忽略，
// 避免 GORM Save 在影响 0 行时回落为 INSERT，把旧 token 与云账号凭据重新写回。
func (d *CloudLinkDAO) SaveLink(link *models.CloudLink) error {
	if link.ID == 0 {
		existing, err := d.GetByUsername(link.Username)
		if err != nil {
			return err
		}
		if existing.ID != 0 {
			link.ID = existing.ID
		}
	}

	if link.ID != 0 {
		return d.db.
			Model(&models.CloudLink{}).
			Where("id = ?", link.ID).
			Select("*").
			Omit("id").
			Updates(link).Error
	}

	return d.db.Create(link).Error
}

func (d *CloudLinkDAO) DeleteByUsername(username string) error {
	return d.db.Where("username = ?", username).Delete(&models.CloudLink{}).Error
}
