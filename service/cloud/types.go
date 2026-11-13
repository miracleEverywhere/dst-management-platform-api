package cloud

import (
	"encoding/json"
)

type apiEnvelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

type User struct {
	ID        int64           `json:"id"`
	Username  string          `json:"username"`
	Email     string          `json:"email"`
	Role      string          `json:"role"`
	IsActive  bool            `json:"isActive"`
	CreatedAt json.RawMessage `json:"createdAt"`
}

type Account struct {
	ID            int64           `json:"id"`
	Username      string          `json:"username"`
	Email         string          `json:"email"`
	Role          string          `json:"role"`
	Plan          string          `json:"plan"`
	Status        string          `json:"status"`
	FavoriteQuota int             `json:"favoriteQuota"`
	CreatedAt     json.RawMessage `json:"createdAt"`
	LastLoginAt   json.RawMessage `json:"lastLoginAt"`
}

type Plan struct {
	Code           string `json:"code"`
	Name           string `json:"name"`
	Price          int    `json:"price"`
	FavoriteQuota  int    `json:"favoriteQuota"`
	TokenQuota     int    `json:"tokenQuota"`
	GameAdminQuota int    `json:"gameAdminQuota"`
	GameSlotQuota  int    `json:"gameSlotQuota"`
	ArchiveQuota   int    `json:"archiveQuota"`
}

type TokenPair struct {
	AccessToken           string `json:"accessToken"`
	RefreshToken          string `json:"refreshToken"`
	AccessTokenExpiresAt  int64  `json:"accessTokenExpiresAt"`
	RefreshTokenExpiresAt int64  `json:"refreshTokenExpiresAt"`
}

type AuthResult struct {
	User           User      `json:"user"`
	Account        Account   `json:"account"`
	Plan           Plan      `json:"plan"`
	Tokens         TokenPair `json:"tokens"`
	CloudActivated bool      `json:"cloudActivated"`
	FavoriteQuota  int       `json:"favoriteQuota"`
	FavoriteCount  int       `json:"favoriteCount"`
	TokenCount     int       `json:"tokenCount"`
}

type MeResult struct {
	Account       Account `json:"account"`
	Plan          Plan    `json:"plan"`
	FavoriteQuota int     `json:"favoriteQuota"`
	FavoriteCount int     `json:"favoriteCount"`
	TokenCount    int     `json:"tokenCount"`
}

type KleiToken struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Token     string `json:"token"`
	Remark    string `json:"remark"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type TokenList struct {
	Items      []KleiToken `json:"items"`
	TokenQuota int         `json:"tokenQuota"`
	Used       int         `json:"used"`
}

type GameEntry struct {
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	UID       string `json:"uid"`
	Nickname  string `json:"nickname"`
	Remark    string `json:"remark"`
	CreatedAt int64  `json:"createdAt"`
	UpdatedAt int64  `json:"updatedAt"`
}

type GameEntryList struct {
	Items []GameEntry `json:"items"`
	Kind  string      `json:"kind"`
	Quota int         `json:"quota"`
	Used  int         `json:"used"`
}

type Archive struct {
	ID               string `json:"id"`
	Name             string `json:"name"`
	FileName         string `json:"fileName"`
	Remark           string `json:"remark"`
	SizeBytes        int64  `json:"sizeBytes"`
	Checksum         string `json:"checksum"`
	CreatedAt        int64  `json:"createdAt"`
	UpdatedAt        int64  `json:"updatedAt"`
	LastDownloadedAt *int64 `json:"lastDownloadedAt"`
	DownloadCount    int    `json:"downloadCount"`
}

type ArchiveList struct {
	Items    []Archive `json:"items"`
	Quota    int       `json:"quota"`
	Used     int       `json:"used"`
	MaxBytes int64     `json:"maxBytes"`
}

type FavoriteMeta struct {
	RoomName   string `json:"roomName"`
	GameMode   string `json:"gameMode"`
	WorldCount int    `json:"worldCount"`
	ModCount   int    `json:"modCount"`
}

type Favorite struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	Remark          string          `json:"remark"`
	Snapshot        json.RawMessage `json:"snapshot"`
	LocalFavoriteID string          `json:"localFavoriteId"`
	Source          string          `json:"source"`
	CreatedAt       int64           `json:"createdAt"`
	UpdatedAt       int64           `json:"updatedAt"`
	Deleted         bool            `json:"deleted"`
	Meta            FavoriteMeta    `json:"meta"`
}

type FavoritePage struct {
	Items      []Favorite `json:"items"`
	Page       int        `json:"page"`
	PageSize   int        `json:"pageSize"`
	Total      int64      `json:"total"`
	TotalPages int        `json:"totalPages"`
}

type SyncItem struct {
	CloudID   string          `json:"cloudId,omitempty"`
	LocalID   string          `json:"localId,omitempty"`
	Name      string          `json:"name,omitempty"`
	Remark    string          `json:"remark,omitempty"`
	Snapshot  json.RawMessage `json:"snapshot,omitempty"`
	CreatedAt int64           `json:"createdAt,omitempty"`
	UpdatedAt int64           `json:"updatedAt,omitempty"`
	Deleted   bool            `json:"deleted,omitempty"`
}

type SyncRequest struct {
	DeviceID  string     `json:"deviceId"`
	Since     int64      `json:"since"`
	Favorites []SyncItem `json:"favorites"`
}

type SyncSkip struct {
	LocalID string `json:"localId"`
	Name    string `json:"name"`
	Reason  string `json:"reason"`
}

type DeletedRef struct {
	ID        string `json:"id"`
	LocalID   string `json:"localId"`
	Name      string `json:"name"`
	UpdatedAt int64  `json:"updatedAt"`
}

type SyncSummary struct {
	Uploaded      int `json:"uploaded"`
	Updated       int `json:"updated"`
	Deleted       int `json:"deleted"`
	Skipped       int `json:"skipped"`
	QuotaExceeded int `json:"quotaExceeded"`
	Conflicts     int `json:"conflicts"`
	Downloaded    int `json:"downloaded"`
}

type FavoriteQuotaInfo struct {
	Limit int `json:"limit"`
	Used  int `json:"used"`
}

type SyncResult struct {
	ServerTime    int64             `json:"serverTime"`
	Favorites     []Favorite        `json:"favorites"`
	Deleted       []DeletedRef      `json:"deleted"`
	Skipped       []SyncSkip        `json:"skipped"`
	Summary       SyncSummary       `json:"summary"`
	FavoriteQuota FavoriteQuotaInfo `json:"favoriteQuota"`
}
