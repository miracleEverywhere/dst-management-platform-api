package models

// CloudLink 保存本地用户与 DMP Cloud 账号的绑定关系。
// 一个本地用户最多绑定一个云账号；同一台 DMP 上一个云账号也只能映射一个本地用户。
// 云账号密码使用 bcrypt 保存，仅用于云服务不可用时的本地回退校验。
type CloudLink struct {
	ID               uint   `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Username         string `gorm:"column:username;not null;uniqueIndex:uk_cloud_links_username" json:"username"`
	CloudAccountID   int64  `gorm:"column:cloud_account_id;not null;uniqueIndex:uk_cloud_links_account" json:"cloudAccountId"`
	CloudUsername    string `gorm:"column:cloud_username;not null" json:"cloudUsername"`
	CloudEmail       string `gorm:"column:cloud_email;not null;default:''" json:"cloudEmail"`
	CloudPassword    string `gorm:"column:cloud_password;type:text" json:"-"`
	Plan             string `gorm:"column:plan;not null;default:basic" json:"plan"`
	FavoriteQuota    int    `gorm:"column:favorite_quota;not null;default:1" json:"favoriteQuota"`
	TokenQuota       int    `gorm:"column:token_quota;not null;default:1" json:"tokenQuota"`
	GameAdminQuota   int    `gorm:"column:game_admin_quota;not null;default:1" json:"gameAdminQuota"`
	GameSlotQuota    int    `gorm:"column:game_slot_quota;not null;default:1" json:"gameSlotQuota"`
	ArchiveQuota     int    `gorm:"column:archive_quota;not null;default:0" json:"archiveQuota"`
	AccessToken      string `gorm:"column:access_token;type:text" json:"-"`
	RefreshToken     string `gorm:"column:refresh_token;type:text" json:"-"`
	AccessExpiresAt  int64  `gorm:"column:access_expires_at;not null;default:0" json:"-"`
	RefreshExpiresAt int64  `gorm:"column:refresh_expires_at;not null;default:0" json:"-"`
	DeviceID         string `gorm:"column:device_id;not null;default:''" json:"deviceId"`
	LastSyncAt       int64  `gorm:"column:last_sync_at;not null;default:0" json:"lastSyncAt"`
	LinkedAt         int64  `gorm:"column:linked_at;not null;default:0" json:"linkedAt"`
	UpdatedAt        int64  `gorm:"column:updated_at;not null;default:0" json:"updatedAt"`
}

func (CloudLink) TableName() string {
	return "cloud_links"
}

// CloudFavoriteMap 记录本地收藏与云端收藏的映射，避免同一条收藏在云端产生重复副本。
type CloudFavoriteMap struct {
	ID              uint   `gorm:"primaryKey;autoIncrement;column:id" json:"id"`
	Username        string `gorm:"column:username;not null;index:idx_cloud_map_user" json:"username"`
	LocalFavoriteID string `gorm:"column:local_favorite_id;not null;uniqueIndex:uk_cloud_map_local" json:"localFavoriteId"`
	CloudFavoriteID string `gorm:"column:cloud_favorite_id;not null;index:idx_cloud_map_cloud" json:"cloudFavoriteId"`
	LastCloudUpdate int64  `gorm:"column:last_cloud_update;not null;default:0" json:"lastCloudUpdate"`
	CreatedAt       int64  `gorm:"column:created_at;not null;default:0" json:"createdAt"`
	UpdatedAt       int64  `gorm:"column:updated_at;not null;default:0" json:"updatedAt"`
}

func (CloudFavoriteMap) TableName() string {
	return "cloud_favorite_maps"
}
