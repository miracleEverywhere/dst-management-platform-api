package cloud

import (
	"bytes"
	"crypto/sha256"
	"dst-management-platform-api/app/room"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"encoding/hex"
	"encoding/json"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

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

// syncWithLink 执行一次云同步。同一用户的同步串行执行，避免多个请求互相覆盖。
func (h *Handler) syncWithLink(c *gin.Context, link *models.CloudLink, direction string) (*SyncResult, int64, error) {
	unlock := lockUserSync(link.Username)
	defer unlock()

	if link.DeviceID == "" {
		link.DeviceID = uuid.NewString()
	}

	maps, err := h.cloudMapDao.ListByUsername(link.Username)
	if err != nil {
		return nil, 0, err
	}
	mapByLocal := make(map[string]*models.CloudFavoriteMap, len(maps))
	mapByCloud := make(map[string]*models.CloudFavoriteMap, len(maps))
	for i := range maps {
		mapByLocal[maps[i].LocalFavoriteID] = &maps[i]
		mapByCloud[maps[i].CloudFavoriteID] = &maps[i]
	}

	localFavorites, err := h.favoriteDao.ListAllFavorites(link.Username, "dst", true)
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

	result, err := h.client.Sync(c.Request.Context(), link, request)
	if err != nil {
		return nil, 0, err
	}

	if direction != "upload" {
		if err = h.applyCloudResult(link.Username, result, mapByCloud, mapByLocal, localByID, localByFingerprint); err != nil {
			return nil, 0, err
		}
	} else {
		h.ensureUploadMappings(link.Username, result, mapByCloud, localByID)
	}

	link.LastSyncAt = result.ServerTime
	if result.FavoriteQuota.Limit > 0 {
		link.FavoriteQuota = result.FavoriteQuota.Limit
	}
	// 同步后顺带刷新一次套餐，用户在云站点升级后无需等到下次登录。
	if me, meErr := h.client.Me(c.Request.Context(), link); meErr == nil {
		applyMeResult(link, me)
	}

	if err = h.saveLink(link); err != nil {
		return nil, 0, err
	}

	localCount := h.localFavoriteCount(link.Username)

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

		var data room.XRoomTotalInfo
		unmarshalErr := json.Unmarshal([]byte(favorite.Data), &data)
		if unmarshalErr == nil {
			item.Name = strings.TrimSpace(data.RoomData.GameName)
		}
		if favorite.DeletedAt == 0 {
			if item.Name == "" || unmarshalErr != nil || len(data.WorldData) == 0 {
				logger.Logger.Warnf("收藏快照缺少房间名或世界，跳过上传, favoriteID: %s", favorite.ID)
				continue
			}
			if msgKey, arg := room.ValidateFavoriteData(&data); msgKey != "" {
				logger.Logger.Warnf("收藏快照校验失败，跳过上传, favoriteID: %s, reason: %s, arg: %s", favorite.ID, msgKey, arg)
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

func (h *Handler) applyCloudResult(username string, result *SyncResult, mapByCloud, mapByLocal map[string]*models.CloudFavoriteMap, localByID map[string]*models.Favorite, localByFingerprint map[string]string) error {
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
			local, err := h.createLocalFavoriteFromCloud(username, remote)
			if err != nil {
				return err
			}
			if local != nil {
				localByID[local.ID] = local
				if fingerprint := snapshotFingerprint(local.Data); fingerprint != "" {
					localByFingerprint[fingerprint] = local.ID
				}
				h.upsertMapping(username, local.ID, remote, mapByLocal, mapByCloud)
			}
			continue
		}

		local := localByID[localID]
		if local == nil {
			// 映射指向的本地收藏已不存在（例如手工清理过数据库），重新创建并重建映射。
			created, err := h.createLocalFavoriteFromCloud(username, remote)
			if err != nil {
				return err
			}
			if created != nil {
				localByID[created.ID] = created
				if fingerprint := snapshotFingerprint(created.Data); fingerprint != "" {
					localByFingerprint[fingerprint] = created.ID
				}
				h.upsertMapping(username, created.ID, remote, mapByLocal, mapByCloud)
			}
			continue
		}

		remoteUpdatedAt := toLocalMillis(remote.UpdatedAt)
		if remoteUpdatedAt > local.UpdatedAt || (remoteUpdatedAt == local.UpdatedAt && local.DeletedAt != 0) {
			snapshot, ok := prepareRemoteSnapshot(remote)
			if !ok {
				continue
			}
			updatedAt := remoteUpdatedAt
			if err := h.favoriteDao.UpdateFavoriteFromCloud(local.ID, username, snapshot, remote.Remark, updatedAt); err != nil {
				return err
			}
			local.Data = snapshot
			local.Remark = remote.Remark
			local.UpdatedAt = updatedAt
			local.DeletedAt = 0
		}
		h.upsertMapping(username, local.ID, remote, mapByLocal, mapByCloud)
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
		if err := h.favoriteDao.SoftDeleteFavoriteAt(local.ID, username, deletedAt); err != nil {
			return err
		}
		local.DeletedAt = deletedAt
		local.UpdatedAt = deletedAt
	}

	return nil
}

func (h *Handler) ensureUploadMappings(username string, result *SyncResult, mapByCloud map[string]*models.CloudFavoriteMap, localByID map[string]*models.Favorite) {
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
		mapping := &models.CloudFavoriteMap{
			Username:        username,
			LocalFavoriteID: remote.LocalFavoriteID,
			CloudFavoriteID: remote.ID,
			LastCloudUpdate: remote.UpdatedAt,
			CreatedAt:       nowSeconds(),
			UpdatedAt:       nowSeconds(),
		}
		if err := h.cloudMapDao.Upsert(mapping); err != nil {
			logger.Logger.Warnf("保存云收藏映射失败, username: %s, localID: %s, err: %v", username, remote.LocalFavoriteID, err)
		}
	}
}

func (h *Handler) createLocalFavoriteFromCloud(username string, remote *Favorite) (*models.Favorite, error) {
	snapshot, ok := prepareRemoteSnapshot(remote)
	if !ok {
		return nil, nil
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
	if err := h.favoriteDao.CreateFavorite(favorite); err != nil {
		return nil, err
	}

	return favorite, nil
}

func (h *Handler) upsertMapping(username, localID string, remote *Favorite, mapByLocal, mapByCloud map[string]*models.CloudFavoriteMap) {
	mapping := mapByLocal[localID]
	if mapping == nil {
		mapping = &models.CloudFavoriteMap{
			Username:        username,
			LocalFavoriteID: localID,
			CloudFavoriteID: remote.ID,
			LastCloudUpdate: remote.UpdatedAt,
			CreatedAt:       nowSeconds(),
			UpdatedAt:       nowSeconds(),
		}
	} else {
		mapping.CloudFavoriteID = remote.ID
		mapping.LastCloudUpdate = remote.UpdatedAt
		mapping.UpdatedAt = nowSeconds()
	}
	if err := h.cloudMapDao.Upsert(mapping); err != nil {
		logger.Logger.Warnf("保存云收藏映射失败, username: %s, localID: %s, err: %v", username, localID, err)
		return
	}
	mapByLocal[localID] = mapping
	mapByCloud[remote.ID] = mapping
}

// prepareRemoteSnapshot 校验云端快照并清空自定义启动命令，返回入库 JSON。
func prepareRemoteSnapshot(remote *Favorite) (string, bool) {
	if len(remote.Snapshot) == 0 || string(remote.Snapshot) == "null" {
		logger.Logger.Warnf("云端收藏快照为空，已跳过, cloudID: %s", remote.ID)
		return "", false
	}
	var data room.XRoomTotalInfo
	if err := json.Unmarshal(remote.Snapshot, &data); err != nil {
		logger.Logger.Warnf("云端收藏快照解析失败，已跳过, cloudID: %s, err: %v", remote.ID, err)
		return "", false
	}
	if msgKey, arg := room.ValidateFavoriteData(&data); msgKey != "" {
		logger.Logger.Warnf("云端收藏快照校验失败，已跳过, cloudID: %s, reason: %s, arg: %s", remote.ID, msgKey, arg)
		return "", false
	}
	if room.ClearFavoriteCustomStartupCmd(&data) {
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
