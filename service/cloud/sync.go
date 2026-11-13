package cloud

import (
	"context"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"errors"
	"strings"
	"sync"

	"github.com/google/uuid"
)

// ===== 云同步 =====
var userSyncLocks = struct {
	sync.Mutex
	locks map[string]*sync.Mutex
}{
	locks: make(map[string]*sync.Mutex),
}

func lockUserSync(username string) func() {
	userSyncLocks.Lock()
	lock, ok := userSyncLocks.locks[username]
	if !ok {
		lock = &sync.Mutex{}
		userSyncLocks.locks[username] = lock
	}
	userSyncLocks.Unlock()

	lock.Lock()

	return lock.Unlock
}

// Sync 执行一次云同步。direction 为 both/upload/download，空值按 both 处理。
func (s *Service) Sync(ctx context.Context, actor service.Actor, direction string) service.Result {
	direction = strings.TrimSpace(direction)
	if direction == "" {
		direction = "both"
	}
	if direction != "both" && direction != "upload" && direction != "download" {
		return service.Fail(constant.CodeCloudSyncInvalidDirection)
	}

	// 先取得用户锁，再读取绑定。Unlink/Link/其它同步都必须经过同一把锁，
	// 避免解绑或切换账号后仍拿旧 link 上传收藏、重建映射。
	unlock := lockUserSync(actor.Username)
	defer unlock()

	link, failed := s.requireLink(actor)
	if failed != nil {
		return *failed
	}

	result, localCount, err := s.syncWithLink(ctx, link, direction)
	if err != nil {
		if errors.Is(err, errLinkChanged) {
			return service.Fail(constant.CodeCloudLinkNotFound)
		}
		if errors.Is(err, ErrRelink) || IsUnavailable(err) || MessageOf(err) != "" {
			return failCloud(actor, err)
		}
		logger.Logger.Errorf("云同步失败, username: %s, err: %v", link.Username, err)
		return service.Fail(constant.CodeCloudSyncFail)
	}

	return service.Msg("cloud sync success", newSyncSummaryView(direction, result, localCount))
}

// syncWithLink 执行一次云同步；调用方必须已经持有该用户的同步锁并重新读取绑定。
func (s *Service) syncWithLink(ctx context.Context, link *models.CloudLink, direction string) (*SyncResult, int64, error) {
	if link.DeviceID == "" {
		link.DeviceID = uuid.NewString()
	}

	maps, err := s.cloudMapDao.ListByUsername(link.Username)
	if err != nil {
		return nil, 0, err
	}
	mapByLocal := make(map[string]*models.CloudFavoriteMap, len(maps))
	mapByCloud := make(map[string]*models.CloudFavoriteMap, len(maps))
	for i := range maps {
		mapByLocal[maps[i].LocalFavoriteID] = &maps[i]
		mapByCloud[maps[i].CloudFavoriteID] = &maps[i]
	}

	localFavorites, err := s.favoriteDao.ListAllFavorites(link.Username, "dst", true)
	if err != nil {
		return nil, 0, err
	}
	localByID := make(map[string]*models.Favorite, len(localFavorites))
	localByFingerprint := make(map[string]string, len(localFavorites))
	for i := range localFavorites {
		localByID[localFavorites[i].ID] = &localFavorites[i]
		if localFavorites[i].DeletedAt == 0 {
			if fingerprint := snapshotFingerprint(localFavorites[i].Data); fingerprint != "" {
				localByFingerprint[fingerprint] = localFavorites[i].ID
			}
		}
	}

	request := SyncRequest{
		DeviceID:  link.DeviceID,
		Since:     0, // 每次拉取全部 tombstone，避免 lastSyncAt 推进后漏掉旧删除
		Favorites: []SyncItem{},
	}
	if direction != "download" {
		request.Favorites = buildSyncItems(link.Username, localFavorites, mapByLocal)
	}

	result, err := s.client.Sync(ctx, link, request)
	if err != nil {
		return nil, 0, err
	}

	if direction != "upload" {
		if err = s.applyCloudResult(link.Username, result, mapByCloud, mapByLocal, localByID, localByFingerprint); err != nil {
			return nil, 0, err
		}
	} else {
		if err = s.ensureUploadMappings(link.Username, result, mapByCloud, localByID); err != nil {
			return nil, 0, err
		}
	}

	link.LastSyncAt = result.ServerTime
	if result.FavoriteQuota.Limit > 0 {
		link.FavoriteQuota = result.FavoriteQuota.Limit
	}
	// 同步后顺带刷新一次套餐，用户在云站点升级后无需等到下次登录。
	if me, meErr := s.client.Me(ctx, link); meErr == nil {
		applyMeResult(link, me)
	}

	if err = s.saveLinkSyncState(link); err != nil {
		return nil, 0, err
	}

	localCount := s.localFavoriteCount(link.Username)

	return result, localCount, nil
}
