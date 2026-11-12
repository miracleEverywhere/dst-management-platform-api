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

// updateIfAccount 按绑定 ID 和预期云账号做条件更新，只写指定字段。
// 查询接口在等待云端响应期间绑定可能已被切换，用预期 CloudAccountID
// 可以阻止旧请求把整个绑定（包括凭据和云账号 ID）写回旧值。
func (d *CloudLinkDAO) updateIfAccount(link *models.CloudLink, fields map[string]any) (int64, error) {
	if link == nil || link.ID == 0 {
		return 0, nil
	}
	result := d.db.
		Model(&models.CloudLink{}).
		Where("id = ? AND cloud_account_id = ?", link.ID, link.CloudAccountID).
		Updates(fields)

	return result.RowsAffected, result.Error
}

// UpdateTokensIfAccount 只更新 token 相关字段，供云查询接口回写刷新后的 token。
func (d *CloudLinkDAO) UpdateTokensIfAccount(link *models.CloudLink) (int64, error) {
	return d.updateIfAccount(link, map[string]any{
		"access_token":       link.AccessToken,
		"refresh_token":      link.RefreshToken,
		"access_expires_at":  link.AccessExpiresAt,
		"refresh_expires_at": link.RefreshExpiresAt,
		"updated_at":         link.UpdatedAt,
	})
}

// UpdateProfileIfAccount 更新套餐、配额与 token，供 /auth/me 刷新使用。
func (d *CloudLinkDAO) UpdateProfileIfAccount(link *models.CloudLink) (int64, error) {
	return d.updateIfAccount(link, map[string]any{
		"plan":               link.Plan,
		"favorite_quota":     link.FavoriteQuota,
		"token_quota":        link.TokenQuota,
		"game_admin_quota":   link.GameAdminQuota,
		"game_slot_quota":    link.GameSlotQuota,
		"archive_quota":      link.ArchiveQuota,
		"access_token":       link.AccessToken,
		"refresh_token":      link.RefreshToken,
		"access_expires_at":  link.AccessExpiresAt,
		"refresh_expires_at": link.RefreshExpiresAt,
		"updated_at":         link.UpdatedAt,
	})
}

// UpdateSyncStateIfAccount 更新同步状态、套餐、配额与 token。
func (d *CloudLinkDAO) UpdateSyncStateIfAccount(link *models.CloudLink) (int64, error) {
	return d.updateIfAccount(link, map[string]any{
		"device_id":          link.DeviceID,
		"last_sync_at":       link.LastSyncAt,
		"plan":               link.Plan,
		"favorite_quota":     link.FavoriteQuota,
		"token_quota":        link.TokenQuota,
		"game_admin_quota":   link.GameAdminQuota,
		"game_slot_quota":    link.GameSlotQuota,
		"archive_quota":      link.ArchiveQuota,
		"access_token":       link.AccessToken,
		"refresh_token":      link.RefreshToken,
		"access_expires_at":  link.AccessExpiresAt,
		"refresh_expires_at": link.RefreshExpiresAt,
		"updated_at":         link.UpdatedAt,
	})
}

func (d *CloudLinkDAO) DeleteByUsername(username string) error {
	return d.db.Where("username = ?", username).Delete(&models.CloudLink{}).Error
}
