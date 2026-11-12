package cloud

import (
	"bytes"
	"context"
	"crypto/sha256"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	roomsvc "dst-management-platform-api/service/room"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

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

func buildSyncItems(username string, favorites []models.Favorite, mapByLocal map[string]*models.CloudFavoriteMap) []SyncItem {
	items := make([]SyncItem, 0, len(favorites))
	for i := range favorites {
		if len(items) >= constant.CloudMaxSyncItems {
			logger.Logger.Warnf("本地收藏超过单次同步上限，仅上传最近更新的 1000 条, username: %s", username)
			break
		}
		favorite := &favorites[i]
		item := SyncItem{
			LocalID:   favorite.ID,
			Remark:    favorite.Remark,
			CreatedAt: toCloudSeconds(favorite.CreatedAt),
			UpdatedAt: toCloudSeconds(favorite.UpdatedAt),
			Deleted:   favorite.DeletedAt != 0,
		}
		if mapping := mapByLocal[favorite.ID]; mapping != nil {
			item.CloudID = mapping.CloudFavoriteID
		}

		var data roomsvc.XRoomTotalInfo
		unmarshalErr := json.Unmarshal([]byte(favorite.Data), &data)
		if unmarshalErr == nil {
			item.Name = strings.TrimSpace(data.RoomData.GameName)
		}
		if favorite.DeletedAt == 0 {
			if item.Name == "" || unmarshalErr != nil || len(data.WorldData) == 0 {
				logger.Logger.Warnf("收藏快照缺少房间名或世界，跳过上传, favoriteID: %s", favorite.ID)
				continue
			}
			if code, arg := roomsvc.ValidateFavoriteData(&data); code != 0 {
				logger.Logger.Warnf("收藏快照校验失败，跳过上传, favoriteID: %s, code: %d, arg: %s", favorite.ID, code, arg)
				continue
			}
			if len(favorite.Data) > 2<<20 {
				logger.Logger.Warnf("收藏快照超过 2MB，跳过上传, favoriteID: %s", favorite.ID)
				continue
			}
			item.Snapshot = json.RawMessage(favorite.Data)
		}

		items = append(items, item)
	}

	return items
}

func (s *Service) applyCloudResult(username string, result *SyncResult, mapByCloud, mapByLocal map[string]*models.CloudFavoriteMap, localByID map[string]*models.Favorite, localByFingerprint map[string]string) error {
	for i := range result.Favorites {
		remote := &result.Favorites[i]
		localID := ""
		if mapping := mapByCloud[remote.ID]; mapping != nil {
			localID = mapping.LocalFavoriteID
		}
		if localID == "" && remote.LocalFavoriteID != "" {
			if _, ok := localByID[remote.LocalFavoriteID]; ok {
				localID = remote.LocalFavoriteID
			}
		}
		if localID == "" && len(remote.Snapshot) > 0 {
			if fingerprint := snapshotFingerprint(string(remote.Snapshot)); fingerprint != "" {
				localID = localByFingerprint[fingerprint]
			}
		}

		if localID == "" {
			local, mapping, err := s.createLocalFavoriteFromCloud(username, remote)
			if err != nil {
				return err
			}
			if local != nil {
				localByID[local.ID] = local
				if fingerprint := snapshotFingerprint(local.Data); fingerprint != "" {
					localByFingerprint[fingerprint] = local.ID
				}
				registerMapping(mapByLocal, mapByCloud, mapping)
			}
			continue
		}

		local := localByID[localID]
		if local == nil {
			// 映射指向的本地收藏已不存在（例如手工清理过数据库），重新创建并重建映射。
			created, mapping, err := s.createLocalFavoriteFromCloud(username, remote)
			if err != nil {
				return err
			}
			if created != nil {
				localByID[created.ID] = created
				if fingerprint := snapshotFingerprint(created.Data); fingerprint != "" {
					localByFingerprint[fingerprint] = created.ID
				}
				registerMapping(mapByLocal, mapByCloud, mapping)
			}
			continue
		}

		remoteUpdatedAt := toLocalMillis(remote.UpdatedAt)
		if remoteUpdatedAt > local.UpdatedAt || (remoteUpdatedAt == local.UpdatedAt && local.DeletedAt != 0) {
			snapshot, ok := prepareRemoteSnapshot(remote)
			if !ok {
				continue
			}
			mapping := mergeCloudMapping(username, local.ID, remote, mapByLocal)
			updatedAt := remoteUpdatedAt
			// 收藏快照与映射必须原子写入，否则映射失败后下次同步无法回连，会重复创建收藏。
			if err := s.favoriteDao.UpdateFavoriteFromCloudWithMapping(local.ID, username, snapshot, remote.Remark, updatedAt, mapping); err != nil {
				return err
			}
			local.Data = snapshot
			local.Remark = remote.Remark
			local.UpdatedAt = updatedAt
			local.DeletedAt = 0
			registerMapping(mapByLocal, mapByCloud, mapping)
			continue
		}
		mapping := mergeCloudMapping(username, local.ID, remote, mapByLocal)
		if err := s.cloudMapDao.Upsert(mapping); err != nil {
			return err
		}
		registerMapping(mapByLocal, mapByCloud, mapping)
	}

	for i := range result.Deleted {
		tombstone := &result.Deleted[i]
		mapping := mapByCloud[tombstone.ID]
		if mapping == nil {
			continue
		}
		local := localByID[mapping.LocalFavoriteID]
		if local == nil || local.DeletedAt != 0 {
			continue
		}
		deletedAt := toLocalMillis(tombstone.UpdatedAt)
		if deletedAt < local.UpdatedAt {
			continue
		}
		if err := s.favoriteDao.SoftDeleteFavoriteAt(local.ID, username, deletedAt); err != nil {
			return err
		}
		local.DeletedAt = deletedAt
		local.UpdatedAt = deletedAt
	}

	return nil
}

func (s *Service) ensureUploadMappings(username string, result *SyncResult, mapByCloud map[string]*models.CloudFavoriteMap, localByID map[string]*models.Favorite) error {
	for i := range result.Favorites {
		remote := &result.Favorites[i]
		if remote.LocalFavoriteID == "" {
			continue
		}
		if _, ok := localByID[remote.LocalFavoriteID]; !ok {
			continue
		}
		if mapByCloud[remote.ID] != nil {
			continue
		}
		mapping := newCloudMapping(username, remote.LocalFavoriteID, remote)
		if err := s.cloudMapDao.Upsert(mapping); err != nil {
			return err
		}
		mapByCloud[remote.ID] = mapping
	}

	return nil
}

func (s *Service) createLocalFavoriteFromCloud(username string, remote *Favorite) (*models.Favorite, *models.CloudFavoriteMap, error) {
	snapshot, ok := prepareRemoteSnapshot(remote)
	if !ok {
		return nil, nil, nil
	}
	updatedAt := toLocalMillis(remote.UpdatedAt)
	if updatedAt == 0 {
		updatedAt = time.Now().UnixMilli()
	}
	createdAt := toLocalMillis(remote.CreatedAt)
	if createdAt == 0 {
		createdAt = updatedAt
	}
	favorite := &models.Favorite{
		Username:  username,
		Type:      "dst",
		Data:      snapshot,
		Remark:    remote.Remark,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		DeletedAt: 0,
	}
	mapping := newCloudMapping(username, "", remote)
	if err := s.favoriteDao.CreateFavoriteWithMapping(favorite, mapping); err != nil {
		return nil, nil, err
	}

	return favorite, mapping, nil
}

func newCloudMapping(username, localID string, remote *Favorite) *models.CloudFavoriteMap {
	now := nowSeconds()

	return &models.CloudFavoriteMap{
		Username:        username,
		LocalFavoriteID: localID,
		CloudFavoriteID: remote.ID,
		LastCloudUpdate: remote.UpdatedAt,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
}

func mergeCloudMapping(username, localID string, remote *Favorite, mapByLocal map[string]*models.CloudFavoriteMap) *models.CloudFavoriteMap {
	mapping := mapByLocal[localID]
	if mapping == nil {
		return newCloudMapping(username, localID, remote)
	}
	mapping.CloudFavoriteID = remote.ID
	mapping.LastCloudUpdate = remote.UpdatedAt
	mapping.UpdatedAt = nowSeconds()

	return mapping
}

func registerMapping(mapByLocal, mapByCloud map[string]*models.CloudFavoriteMap, mapping *models.CloudFavoriteMap) {
	if mapping == nil {
		return
	}
	mapByLocal[mapping.LocalFavoriteID] = mapping
	mapByCloud[mapping.CloudFavoriteID] = mapping
}

// prepareRemoteSnapshot 校验云端快照并清空自定义启动命令，返回入库 JSON。
func prepareRemoteSnapshot(remote *Favorite) (string, bool) {
	if len(remote.Snapshot) == 0 || string(remote.Snapshot) == "null" {
		logger.Logger.Warnf("云端收藏快照为空，已跳过, cloudID: %s", remote.ID)
		return "", false
	}
	var data roomsvc.XRoomTotalInfo
	if err := json.Unmarshal(remote.Snapshot, &data); err != nil {
		logger.Logger.Warnf("云端收藏快照解析失败，已跳过, cloudID: %s, err: %v", remote.ID, err)
		return "", false
	}
	if code, arg := roomsvc.ValidateFavoriteData(&data); code != 0 {
		logger.Logger.Warnf("云端收藏快照校验失败，已跳过, cloudID: %s, code: %d, arg: %s", remote.ID, code, arg)
		return "", false
	}
	if roomsvc.ClearFavoriteCustomStartupCmd(&data) {
		encoded, err := json.Marshal(&data)
		if err != nil {
			logger.Logger.Warnf("云端收藏快照序列化失败，已跳过, cloudID: %s, err: %v", remote.ID, err)
			return "", false
		}
		return string(encoded), true
	}

	return string(remote.Snapshot), true
}

// snapshotFingerprint 与云服务的收藏指纹算法保持一致：压缩后的 JSON 做 SHA-256。
func snapshotFingerprint(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, []byte(raw)); err != nil {
		return ""
	}
	sum := sha256.Sum256(compact.Bytes())

	return hex.EncodeToString(sum[:])
}

// 本地收藏时间戳是毫秒，云服务使用 Unix 秒。
func toCloudSeconds(ts int64) int64 {
	if ts > 100_000_000_000 {
		return ts / 1000
	}

	return ts
}

func toLocalMillis(ts int64) int64 {
	if ts == 0 {
		return 0
	}
	if ts < 100_000_000_000 {
		return ts * 1000
	}

	return ts
}
