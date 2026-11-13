// Package cloud 云账号、云收藏同步与云存档的业务实现，不依赖 gin。
package cloud

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	roomsvc "dst-management-platform-api/service/room"
	"errors"
	"fmt"
	"strings"
	"time"
)

// ===== Service 与依赖 =====
// Service 云模块业务实现，不依赖 gin
type Service struct {
	userDao      *dao.UserDAO
	cloudLinkDao *dao.CloudLinkDAO
	cloudMapDao  *dao.CloudFavoriteMapDAO
	favoriteDao  *dao.FavoriteDAO
	client       *Client
	roomSvc      *roomsvc.Service
}

// New 创建云模块 service；roomSvc 供云存档导入复用房间模块的导入实现
func New(userDao *dao.UserDAO, cloudLinkDao *dao.CloudLinkDAO, cloudMapDao *dao.CloudFavoriteMapDAO, favoriteDao *dao.FavoriteDAO, roomSvc *roomsvc.Service) *Service {
	return &Service{
		userDao:      userDao,
		cloudLinkDao: cloudLinkDao,
		cloudMapDao:  cloudMapDao,
		favoriteDao:  favoriteDao,
		client:       NewClient(),
		roomSvc:      roomSvc,
	}
}

// ===== 响应与通用工具 =====
// errLinkChanged 表示云绑定在请求等待期间已被解绑或切换，旧请求不能再回写。
var errLinkChanged = errors.New("cloud link changed")

// failCloud 把云客户端错误转换为 DMP 业务结果，HTTP 状态保持 200。
func failCloud(actor service.Actor, err error) service.Result {
	switch {
	case errors.Is(err, ErrRelink):
		return service.Fail(constant.CodeCloudRelinkRequired)
	case IsUnavailable(err):
		logger.Logger.Warnf("云服务不可用, api: %s, err: %v", actor.Path, err)
		return service.Fail(constant.CodeCloudUnavailable)
	default:
		if msg := MessageOf(err); msg != "" {
			logger.Logger.Warnf("云服务请求失败, api: %s, err: %v", actor.Path, err)
			return service.FailMsg(constant.CodeCloudRemoteError, msg)
		}
		logger.Logger.Errorf("云服务请求失败, api: %s, err: %v", actor.Path, err)
		return service.Fail(constant.CodeCloudUnavailable)
	}
}

func nowSeconds() int64 {
	return time.Now().Unix()
}

// ===== 视图 =====
type cloudLoginView struct {
	Token         string `json:"token"`
	Username      string `json:"username"`
	Fallback      bool   `json:"fallback"`
	Plan          string `json:"plan"`
	FavoriteQuota int    `json:"favoriteQuota"`
}

func linkView(link *models.CloudLink) map[string]any {
	if link == nil || link.ID == 0 {
		return map[string]any{"linked": false}
	}

	return map[string]any{
		"linked":         true,
		"username":       link.Username,
		"cloudAccountId": link.CloudAccountID,
		"cloudUsername":  link.CloudUsername,
		"cloudEmail":     link.CloudEmail,
		"plan":           link.Plan,
		"planName":       PlanName(link.Plan),
		"favoriteQuota":  link.FavoriteQuota,
		"tokenQuota":     link.TokenQuota,
		"gameAdminQuota": link.GameAdminQuota,
		"gameSlotQuota":  link.GameSlotQuota,
		"archiveQuota":   link.ArchiveQuota,
		"lastSyncAt":     link.LastSyncAt,
		"linkedAt":       link.LinkedAt,
		"updatedAt":      link.UpdatedAt,
	}
}

type syncSummaryView struct {
	Direction     string     `json:"direction"`
	ServerTime    int64      `json:"serverTime"`
	Uploaded      int        `json:"uploaded"`
	Updated       int        `json:"updated"`
	Deleted       int        `json:"deleted"`
	Skipped       int        `json:"skipped"`
	QuotaExceeded int        `json:"quotaExceeded"`
	Conflicts     int        `json:"conflicts"`
	Downloaded    int        `json:"downloaded"`
	FavoriteQuota int        `json:"favoriteQuota"`
	FavoriteUsed  int        `json:"favoriteUsed"`
	SkippedItems  []SyncSkip `json:"skippedItems"`
	LocalCount    int64      `json:"localCount"`
}

func newSyncSummaryView(direction string, result *SyncResult, localCount int64) syncSummaryView {
	skipped := result.Skipped
	if skipped == nil {
		skipped = []SyncSkip{}
	}

	return syncSummaryView{
		Direction:     direction,
		ServerTime:    result.ServerTime,
		Uploaded:      result.Summary.Uploaded,
		Updated:       result.Summary.Updated,
		Deleted:       result.Summary.Deleted,
		Skipped:       result.Summary.Skipped,
		QuotaExceeded: result.Summary.QuotaExceeded,
		Conflicts:     result.Summary.Conflicts,
		Downloaded:    result.Summary.Downloaded,
		FavoriteQuota: result.FavoriteQuota.Limit,
		FavoriteUsed:  result.FavoriteQuota.Used,
		SkippedItems:  skipped,
		LocalCount:    localCount,
	}
}

func (s *Service) localFavoriteCount(username string) int64 {
	count, err := s.favoriteDao.Count("username = ? AND type = ? AND deleted_at = 0", username, "dst")
	if err != nil {
		logger.Logger.Errorf("统计本地收藏失败, username: %s, err: %v", username, err)
		return 0
	}

	return count
}

// ===== 权限与校验 =====
func (s *Service) canCreateRoom(actor service.Actor) bool {
	if actor.IsAdmin() {
		return true
	}
	user, err := s.getLocalUser(actor.Username)
	if err != nil || user.Username == "" {
		return false
	}

	return user.RoomCreation
}

func (s *Service) hasRoomPermission(actor service.Actor, roomID int) bool {
	if actor.IsAdmin() {
		return true
	}
	user, err := s.getLocalUser(actor.Username)
	if err != nil || user.Username == "" {
		return false
	}
	for _, id := range strings.Split(user.Rooms, ",") {
		if id == fmt.Sprintf("%d", roomID) {
			return true
		}
	}

	return false
}

// requireLink 读取当前本地账号的云绑定，未绑定时返回失败结果
func (s *Service) requireLink(actor service.Actor) (*models.CloudLink, *service.Result) {
	link, err := s.getLink(actor.Username)
	if err != nil {
		logger.Logger.Errorf("查询云账号绑定失败, err: %v", err)
		result := service.Fail(constant.CodeCloudDBError)
		return nil, &result
	}
	if link.ID == 0 {
		result := service.Fail(constant.CodeCloudLinkNotFound)
		return nil, &result
	}

	return link, nil
}

// QuotasForPlan 按套餐返回各配额
func QuotasForPlan(plan string) (favorite, token, admin, slot, archive int) {
	switch plan {
	case "ultimate":
		return 500, 500, 500, 500, 5
	case "pro":
		return 20, 20, 20, 20, 0
	default:
		return 1, 1, 1, 1, 0
	}
}

// PlanName 套餐展示名，保持与原实现一致的中文文案
func PlanName(plan string) string {
	switch plan {
	case "ultimate":
		return "旗舰版"
	case "pro":
		return "高级版"
	default:
		return "基础版"
	}
}
