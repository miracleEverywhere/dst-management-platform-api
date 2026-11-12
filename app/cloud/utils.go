package cloud

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"dst-management-platform-api/app/room"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/utils"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ===== Handler 与依赖 =====

type Handler struct {
	userDao      *dao.UserDAO
	cloudLinkDao *dao.CloudLinkDAO
	cloudMapDao  *dao.CloudFavoriteMapDAO
	favoriteDao  *dao.FavoriteDAO
	client       *Client
	roomHandler  *room.Handler
}

func NewHandler(userDao *dao.UserDAO, cloudLinkDao *dao.CloudLinkDAO, cloudMapDao *dao.CloudFavoriteMapDAO, favoriteDao *dao.FavoriteDAO, roomHandler *room.Handler) *Handler {
	return &Handler{
		userDao:      userDao,
		cloudLinkDao: cloudLinkDao,
		cloudMapDao:  cloudMapDao,
		favoriteDao:  favoriteDao,
		client:       NewClient(),
		roomHandler:  roomHandler,
	}
}

// ===== 响应与通用工具 =====

func ok(c *gin.Context, msg string, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": msg, "data": data})
}

func fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": code, "message": msg, "data": nil})
}

// failCloud 把云客户端错误转换为 DMP 业务响应，HTTP 状态保持 200。
func failCloud(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrRelink):
		fail(c, 201, message.Get(c, "cloud relink required"))
	case IsUnavailable(err):
		logger.Logger.Warnf("云服务不可用, api: %s, err: %v", c.Request.URL.Path, err)
		fail(c, 201, message.Get(c, "cloud unavailable"))
	default:
		if msg := MessageOf(err); msg != "" {
			logger.Logger.Warnf("云服务请求失败, api: %s, err: %v", c.Request.URL.Path, err)
			fail(c, 201, msg)
			return
		}
		logger.Logger.Errorf("云服务请求失败, api: %s, err: %v", c.Request.URL.Path, err)
		fail(c, 201, message.Get(c, "cloud unavailable"))
	}
}

func nowSeconds() int64 {
	return time.Now().Unix()
}

func currentUsername(c *gin.Context) string {
	return c.GetString("username")
}

// ===== 云账号与本地用户 =====

func (h *Handler) getLink(username string) (*models.CloudLink, error) {
	return h.cloudLinkDao.GetByUsername(username)
}

func (h *Handler) saveLink(link *models.CloudLink) error {
	link.UpdatedAt = nowSeconds()
	if link.DeviceID == "" {
		link.DeviceID = uuid.NewString()
	}
	if link.LinkedAt == 0 {
		link.LinkedAt = nowSeconds()
	}

	return h.cloudLinkDao.SaveLink(link)
}

func (h *Handler) getLocalUser(username string) (*models.User, error) {
	return h.userDao.GetUserByUsername(username)
}

// applyAuthResult 把云登录 / 云绑定返回的账号信息写入本地绑定表。
func (h *Handler) applyAuthResult(link *models.CloudLink, result *AuthResult, password string) error {
	plan := result.Account.Plan
	if plan == "" {
		plan = result.Plan.Code
	}
	if plan == "" {
		plan = "basic"
	}
	favoriteQuota, tokenQuota, adminQuota, slotQuota, archiveQuota := QuotasForPlan(plan)

	username := result.User.Username
	if username == "" {
		username = result.Account.Username
	}
	email := result.User.Email
	if email == "" {
		email = result.Account.Email
	}

	passwordHash := link.CloudPassword
	if password != "" {
		hash, _, err := hashCloudPassword(password)
		if err != nil {
			return err
		}
		passwordHash = hash
	}

	link.CloudAccountID = result.Account.ID
	link.CloudUsername = username
	link.CloudEmail = email
	link.CloudPassword = passwordHash
	link.Plan = plan
	link.FavoriteQuota = favoriteQuota
	link.TokenQuota = tokenQuota
	link.GameAdminQuota = adminQuota
	link.GameSlotQuota = slotQuota
	link.ArchiveQuota = archiveQuota
	if link.LinkedAt == 0 {
		link.LinkedAt = nowSeconds()
	}
	link.UpdatedAt = nowSeconds()
	if link.DeviceID == "" {
		link.DeviceID = uuid.NewString()
	}

	return applyTokenPair(link, result.Tokens)
}

// applyMeResult 刷新套餐与配额，不修改 token。
func applyMeResult(link *models.CloudLink, result *MeResult) {
	plan := result.Account.Plan
	if plan == "" {
		plan = result.Plan.Code
	}
	if plan == "" {
		plan = "basic"
	}
	favoriteQuota, tokenQuota, adminQuota, slotQuota, archiveQuota := QuotasForPlan(plan)

	link.Plan = plan
	link.FavoriteQuota = favoriteQuota
	link.TokenQuota = tokenQuota
	link.GameAdminQuota = adminQuota
	link.GameSlotQuota = slotQuota
	link.ArchiveQuota = archiveQuota
	link.UpdatedAt = nowSeconds()
}

func (h *Handler) refreshProfile(c *gin.Context, link *models.CloudLink) error {
	me, err := h.client.Me(c.Request.Context(), link)
	if err != nil {
		return err
	}
	applyMeResult(link, me)
	if err = h.saveLink(link); err != nil {
		return err
	}

	return nil
}

// hashCloudPassword 生成云密码哈希，并返回本地 users 表应使用的密码版本。
// bcrypt 只接受 72 字节以内的密码，超长密码退化为 SHA-256 + bcrypt。
func hashCloudPassword(password string) (string, string, error) {
	if len([]byte(password)) <= 72 {
		hash, err := utils.GenerateBcryptPassword(password)
		return hash, models.PasswordVersionBcrypt, err
	}
	hash, err := utils.GenerateBcryptSha256Password(password)

	return hash, models.PasswordVersionBcryptSha256, err
}

func validateCloudPassword(password, hash string) bool {
	return utils.ValidatePassword(password, hash) || utils.ValidateBcryptSha256Password(password, hash)
}

// refreshProfileQuietly 在云登录后刷新套餐与配额；失败不阻塞登录。
func (h *Handler) refreshProfileQuietly(c *gin.Context, link *models.CloudLink) {
	if err := h.refreshProfile(c, link); err != nil {
		logger.Logger.Warnf("刷新云套餐失败, username: %s, err: %v", link.Username, err)
	}
}

func (h *Handler) issueLocalToken(c *gin.Context, user *models.User) (string, error) {
	if user.Disabled {
		return "", errors.New("disabled")
	}

	token, err := utils.GenerateJWT(*user, []byte(cache.JwtSecret), constant.JwtExpirationHours)
	if err != nil {
		return "", err
	}
	cache.SetTokenVersion(user.Username, user.TokenVersion)

	return token, nil
}

// ===== 视图 =====

func linkView(link *models.CloudLink) gin.H {
	if link == nil || link.ID == 0 {
		return gin.H{"linked": false}
	}

	return gin.H{
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

func (h *Handler) localFavoriteCount(username string) int64 {
	count, err := h.favoriteDao.Count("username = ? AND type = ? AND deleted_at = 0", username, "dst")
	if err != nil {
		logger.Logger.Errorf("统计本地收藏失败, username: %s, err: %v", username, err)
		return 0
	}

	return count
}

// ===== 权限与校验 =====

func (h *Handler) canCreateRoom(c *gin.Context) bool {
	if c.GetString("role") == "admin" {
		return true
	}
	user, err := h.getLocalUser(currentUsername(c))
	if err != nil || user.Username == "" {
		return false
	}

	return user.RoomCreation
}

func (h *Handler) hasRoomPermission(c *gin.Context, roomID int) bool {
	if c.GetString("role") == "admin" {
		return true
	}
	user, err := h.getLocalUser(currentUsername(c))
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

var cloudIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func validCloudID(id string) bool {
	return cloudIDPattern.MatchString(id)
}

// attachmentHeader 生成兼容中文文件名的下载头。
func attachmentHeader(fileName string) string {
	name := strings.TrimSpace(fileName)
	if name == "" {
		name = "archive.zip"
	}
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, `"`, "_")
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return '_'
		}
		return r
	}, name)
	if filepath.Ext(name) == "" {
		name += ".zip"
	}
	fallback := make([]rune, 0, len(name))
	for _, r := range name {
		if r < 0x20 || r > 0x7e || r == '"' || r == '\\' {
			fallback = append(fallback, '_')
		} else {
			fallback = append(fallback, r)
		}
	}

	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, string(fallback), url.PathEscape(name))
}

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}

	return string(runes[:max])
}

// ===== 云存档 =====

var workshopPattern = regexp.MustCompile(`workshop-\d+`)

var errArchiveTooLarge = errors.New("cloud archive too large")

// prepareArchive 复制当前存档，清理日志与旧快照后压缩，返回 zip 路径、remark 和清理函数。
func (h *Handler) prepareArchive(c *gin.Context, roomID int) (string, string, func(), error) {
	_, worlds, _, err := dao.FetchGameInfo(roomID)
	if err != nil {
		return "", "", func() {}, err
	}
	if len(*worlds) == 0 {
		return "", "", func() {}, fmt.Errorf("房间没有世界")
	}

	clusterPath := fmt.Sprintf("%s/Cluster_%d", utils.ClusterPath, roomID)
	if !utils.FileDirectoryExists(clusterPath) {
		return "", "", func() {}, fmt.Errorf("存档目录不存在: %s", clusterPath)
	}

	tempBase := constant.CloudTempPath
	if err = os.MkdirAll(tempBase, 0o700); err != nil {
		return "", "", func() {}, err
	}
	tempDir, err := os.MkdirTemp(tempBase, "archive-*")
	if err != nil {
		return "", "", func() {}, err
	}
	cleanup := func() {
		if err := utils.RemoveDir(tempDir); err != nil {
			logger.Logger.Warnf("清理云存档临时目录失败, dir: %s, err: %v", tempDir, err)
		}
	}

	archiveDir := filepath.Join(tempDir, fmt.Sprintf("Cluster_%d", roomID))
	if err = copyDir(clusterPath, archiveDir); err != nil {
		cleanup()
		return "", "", func() {}, err
	}
	if err = removeLogs(archiveDir); err != nil {
		logger.Logger.Warnf("清理云存档日志目录失败, dir: %s, err: %v", archiveDir, err)
	}
	if err = removeLogFiles(archiveDir); err != nil {
		logger.Logger.Warnf("清理云存档日志文件失败, dir: %s, err: %v", archiveDir, err)
	}
	if err = pruneSnapshots(archiveDir, constant.CloudKeepSnapshots); err != nil {
		logger.Logger.Warnf("清理云存档旧快照失败, dir: %s, err: %v", archiveDir, err)
	}

	zipPath := filepath.Join(tempDir, fmt.Sprintf("Cluster_%d_%d.zip", roomID, time.Now().Unix()))
	if err = utils.Zip(archiveDir, zipPath); err != nil {
		cleanup()
		return "", "", func() {}, err
	}

	fileInfo, err := os.Stat(zipPath)
	if err != nil {
		cleanup()
		return "", "", func() {}, err
	}
	if fileInfo.Size() > constant.CloudArchiveMaxBytes {
		cleanup()
		return "", "", func() {}, errArchiveTooLarge
	}

	//game := dst.NewGameController(roomInfo, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	remark := buildArchiveRemark(currentPublicIP())

	return zipPath, truncateRunes(remark, 255), cleanup, nil
}

// buildArchiveMetadata 生成云存档元数据，内容是与房间收藏同结构的 XRoomTotalInfo。
// 云端要求该字段必填，用于展示房间名称、世界与模组信息。
func buildArchiveMetadata(roomInfo *models.Room, worlds *[]models.World, roomSetting *models.RoomSetting) (string, error) {
	raw, err := json.Marshal(room.RoomSnapshot(roomInfo, worlds, roomSetting))
	if err != nil {
		return "", err
	}

	return string(raw), nil
}

// func buildArchiveRemark(room *models.Room, worlds *[]models.World, session *dst.RoomSessionInfo, publicIP string) string {
func buildArchiveRemark(s string) string {
	//worldParts := make([]string, 0, len(*worlds))
	//for _, world := range *worlds {
	//	master := ""
	//	if world.IsMaster {
	//		master = "，主世界"
	//	}
	//	worldParts = append(worldParts, fmt.Sprintf("%s(直连 %d%s)", world.WorldName, world.ServerPort, master))
	//}
	//
	//return fmt.Sprintf(
	//	"房间：%s；模组：%d 个；天数：%d；公网IP：%s；世界：%s",
	//	room.GameName,
	//	countMods(room, worlds),
	//	session.Cycles,
	//	publicIP,
	//	strings.Join(worldParts, "、"),
	//)

	return fmt.Sprintf("DMP IP: %s", s)
}

func countMods(room *models.Room, worlds *[]models.World) int {
	mods := make(map[string]struct{})
	for _, mod := range workshopPattern.FindAllString(room.ModData, -1) {
		mods[mod] = struct{}{}
	}
	for _, world := range *worlds {
		for _, mod := range workshopPattern.FindAllString(world.ModData, -1) {
			mods[mod] = struct{}{}
		}
	}

	return len(mods)
}

func currentPublicIP() string {
	if cache.InternetIP != "" {
		return cache.InternetIP
	}
	if ip, err := scheduler.GetInternetIP1(); err == nil && ip != "" {
		cache.InternetIP = ip
		return ip
	}
	if ip, err := scheduler.GetInternetIP2(); err == nil && ip != "" {
		cache.InternetIP = ip
		return ip
	}

	return "未知"
}

func copyDir(source, target string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)

		if info.IsDir() {
			mode := info.Mode().Perm() | 0o700
			if err = os.MkdirAll(destination, mode); err != nil {
				return err
			}
			_ = os.Chtimes(destination, info.ModTime(), info.ModTime())
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		if err = os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}

		mode := info.Mode().Perm() | 0o600
		dst, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
		if err != nil {
			_ = src.Close()
			return err
		}
		if _, err = io.Copy(dst, src); err != nil {
			_ = dst.Close()
			_ = src.Close()
			return err
		}
		_ = src.Close()
		if err = dst.Close(); err != nil {
			return err
		}

		_ = os.Chtimes(destination, info.ModTime(), info.ModTime())
		return nil
	})
}

func removeLogs(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		lower := strings.ToLower(info.Name())
		if lower == "server_log" || lower == "server_chat_log" {
			if err = os.RemoveAll(path); err != nil {
				return err
			}
			return filepath.SkipDir
		}

		return nil
	})
}

func pruneSnapshots(root string, keep int) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() || strings.ToLower(info.Name()) != "session" {
			return nil
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if err = pruneSessionDir(filepath.Join(path, entry.Name()), keep); err != nil {
				logger.Logger.Warnf("清理快照失败, dir: %s, err: %v", entry.Name(), err)
			}
		}

		return filepath.SkipDir
	})
}

func pruneSessionDir(dir string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	type snapshot struct {
		name    string
		modTime time.Time
	}
	snapshots := make([]snapshot, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(strings.ToLower(entry.Name()), ".meta") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		snapshots = append(snapshots, snapshot{name: entry.Name(), modTime: info.ModTime()})
	}

	sort.Slice(snapshots, func(i, j int) bool {
		if snapshots[i].modTime.Equal(snapshots[j].modTime) {
			return snapshots[i].name > snapshots[j].name
		}
		return snapshots[i].modTime.After(snapshots[j].modTime)
	})

	for index := keep; index < len(snapshots); index++ {
		name := snapshots[index].name
		if err = os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err = os.Remove(filepath.Join(dir, name+".meta")); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

// 日志清理还会删除根目录下常见的 .log / 含 log 的 .txt 文件。
func removeLogFiles(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		name := strings.ToLower(info.Name())
		if strings.HasSuffix(name, ".log") || (strings.HasSuffix(name, ".txt") && strings.Contains(name, "log")) {
			if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}

		return nil
	})
}

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

// ===== 凭据加密 =====

var (
	cloudKeyOnce sync.Once
	cloudKey     []byte
	cloudKeyErr  error
)

// loadOrCreateKey 读取 dmp_files/cloud.key，不存在时生成 32 字节随机密钥并写入 0600 文件。
func loadOrCreateKey() ([]byte, error) {
	cloudKeyOnce.Do(func() {
		cloudKey, cloudKeyErr = doLoadOrCreateKey(constant.CloudKeyPath)
	})
	if cloudKeyErr != nil {
		return nil, cloudKeyErr
	}

	return cloudKey, nil
}

func doLoadOrCreateKey(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err == nil {
		if len(data) != 32 {
			return nil, fmt.Errorf("云凭据密钥长度不正确: %s", path)
		}
		return data, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, err
	}

	if err = os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}

	key := make([]byte, 32)
	if _, err = rand.Read(key); err != nil {
		return nil, err
	}

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		// 并发创建时另一个请求可能已经写入，重新读取即可。
		if errors.Is(err, os.ErrExist) {
			return doLoadOrCreateKey(path)
		}
		return nil, err
	}
	defer file.Close()

	if _, err = file.Write(key); err != nil {
		_ = os.Remove(path)
		return nil, err
	}

	return key, nil
}

// encryptSecret 使用 AES-256-GCM 加密云凭据，AAD 绑定本地用户名，防止把 A 用户的 token 复制给 B。
func encryptSecret(key []byte, username, plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), []byte(username))

	return base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func decryptSecret(key []byte, username, encoded string) (string, error) {
	if encoded == "" {
		return "", nil
	}

	raw, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("云凭据密文长度不正确")
	}

	plaintext, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], []byte(username))
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// ===== 云服务客户端 =====

// ErrRelink 表示云凭据已失效或无法解密，需要用户重新绑定云账号。
var ErrRelink = errors.New("cloud relink required")

// APIError 是云服务返回的错误。Unavailable 为 true 时表示网络错误、超时或 5xx，
// 只有这类错误允许在云登录时回退到本地账号。
type APIError struct {
	StatusCode  int
	Code        int
	Message     string
	Unavailable bool
	cause       error
}

func (e *APIError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("cloud api %d: %s: %v", e.StatusCode, e.Message, e.cause)
	}

	return fmt.Sprintf("cloud api %d: %s", e.StatusCode, e.Message)
}

func (e *APIError) Unwrap() error {
	return e.cause
}

func IsUnavailable(err error) bool {
	var target *APIError
	return errors.As(err, &target) && target.Unavailable
}

func IsUnauthorized(err error) bool {
	var target *APIError
	return errors.As(err, &target) && target.StatusCode == http.StatusUnauthorized
}

func IsForbidden(err error) bool {
	var target *APIError
	return errors.As(err, &target) && target.StatusCode == http.StatusForbidden
}

func MessageOf(err error) string {
	var target *APIError
	if errors.As(err, &target) && target.Message != "" {
		return target.Message
	}

	return ""
}

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

type Client struct {
	baseURL      string
	httpClient   *http.Client
	syncClient   *http.Client
	uploadClient *http.Client
	streamClient *http.Client
}

func NewClient() *Client {
	return &Client{
		baseURL:      strings.TrimRight(constant.CloudAPIBaseURL, "/"),
		httpClient:   &http.Client{Timeout: constant.CloudHTTPTimeout},
		syncClient:   &http.Client{Timeout: constant.CloudSyncTimeout},
		uploadClient: &http.Client{Timeout: constant.CloudArchiveTimeout},
		streamClient: &http.Client{Timeout: constant.CloudArchiveTimeout},
	}
}

func (c *Client) newRequest(ctx context.Context, method, path, accessToken string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "dmp/"+constant.Version)
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}

	return req, nil
}

func (c *Client) requestJSON(ctx context.Context, client *http.Client, method, path, accessToken string, body any, out any) error {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}

	req, err := c.newRequest(ctx, method, path, accessToken, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
	}

	resp, err := client.Do(req)
	if err != nil {
		return &APIError{StatusCode: 0, Message: "云服务暂时不可用", Unavailable: true, cause: err}
	}
	defer resp.Body.Close()

	return decodeResponse(resp, out)
}

func decodeResponse(resp *http.Response, out any) error {
	data, err := io.ReadAll(io.LimitReader(resp.Body, constant.CloudMaxResponseBytes+1))
	if err != nil {
		return &APIError{StatusCode: resp.StatusCode, Message: "读取云服务响应失败", Unavailable: resp.StatusCode >= 500, cause: err}
	}
	if len(data) > constant.CloudMaxResponseBytes {
		return &APIError{StatusCode: resp.StatusCode, Message: "云服务响应过大"}
	}

	var envelope apiEnvelope
	unmarshalErr := json.Unmarshal(data, &envelope)
	if resp.StatusCode >= 500 {
		message := "云服务暂时不可用"
		if unmarshalErr == nil && envelope.Message != "" {
			message = envelope.Message
		}
		return &APIError{StatusCode: resp.StatusCode, Code: envelope.Code, Message: message, Unavailable: true}
	}
	if unmarshalErr != nil {
		return &APIError{StatusCode: resp.StatusCode, Message: "云服务响应格式不正确", cause: unmarshalErr}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 || (envelope.Code != 200 && envelope.Code != 201) {
		message := envelope.Message
		if message == "" {
			message = "云服务请求失败"
		}
		return &APIError{StatusCode: resp.StatusCode, Code: envelope.Code, Message: message}
	}
	if out != nil && len(envelope.Data) > 0 && string(envelope.Data) != "null" {
		if err = json.Unmarshal(envelope.Data, out); err != nil {
			return &APIError{StatusCode: resp.StatusCode, Message: "解析云服务响应失败", cause: err}
		}
	}

	return nil
}

func (c *Client) accessToken(ctx context.Context, link *models.CloudLink) (string, error) {
	if link.AccessToken == "" || link.RefreshToken == "" {
		return "", ErrRelink
	}
	if link.AccessExpiresAt > time.Now().Unix()+60 {
		return c.decryptToken(link, link.AccessToken)
	}

	return c.refreshLink(ctx, link)
}

func (c *Client) decryptToken(link *models.CloudLink, encrypted string) (string, error) {
	key, err := loadOrCreateKey()
	if err != nil {
		return "", err
	}
	token, err := decryptSecret(key, link.Username, encrypted)
	if err != nil {
		return "", ErrRelink
	}
	if token == "" {
		return "", ErrRelink
	}

	return token, nil
}

func (c *Client) refreshLink(ctx context.Context, link *models.CloudLink) (string, error) {
	if link.RefreshToken == "" {
		return "", ErrRelink
	}
	if link.RefreshExpiresAt > 0 && link.RefreshExpiresAt <= time.Now().Unix() {
		return "", ErrRelink
	}

	refreshToken, err := c.decryptToken(link, link.RefreshToken)
	if err != nil {
		return "", err
	}

	pair, err := c.Refresh(ctx, refreshToken)
	if err != nil {
		if IsUnauthorized(err) || IsForbidden(err) {
			return "", ErrRelink
		}
		return "", err
	}

	if err = applyTokenPair(link, *pair); err != nil {
		return "", err
	}

	return c.decryptToken(link, link.AccessToken)
}

func applyTokenPair(link *models.CloudLink, pair TokenPair) error {
	key, err := loadOrCreateKey()
	if err != nil {
		return err
	}
	access, err := encryptSecret(key, link.Username, pair.AccessToken)
	if err != nil {
		return err
	}
	refresh, err := encryptSecret(key, link.Username, pair.RefreshToken)
	if err != nil {
		return err
	}
	link.AccessToken = access
	link.RefreshToken = refresh
	link.AccessExpiresAt = pair.AccessTokenExpiresAt
	link.RefreshExpiresAt = pair.RefreshTokenExpiresAt

	return nil
}

// doAuthorized 发送需要鉴权的请求。401 时自动刷新一次 token 并重试。
func (c *Client) doAuthorized(ctx context.Context, client *http.Client, link *models.CloudLink, build func(accessToken string) (*http.Request, error)) (*http.Response, error) {
	token, err := c.accessToken(ctx, link)
	if err != nil {
		return nil, err
	}

	req, err := build(token)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, &APIError{Message: "云服务暂时不可用", Unavailable: true, cause: err}
	}
	if resp.StatusCode != http.StatusUnauthorized {
		return resp, nil
	}
	_ = resp.Body.Close()

	token, err = c.refreshLink(ctx, link)
	if err != nil {
		return nil, err
	}
	req, err = build(token)
	if err != nil {
		return nil, err
	}
	resp, err = client.Do(req)
	if err != nil {
		return nil, &APIError{Message: "云服务暂时不可用", Unavailable: true, cause: err}
	}

	return resp, nil
}

func (c *Client) authorizedJSON(ctx context.Context, client *http.Client, link *models.CloudLink, method, path string, body any, out any) error {
	var bodyData []byte
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		bodyData = data
	}

	resp, err := c.doAuthorized(ctx, client, link, func(accessToken string) (*http.Request, error) {
		var reader io.Reader
		if bodyData != nil {
			reader = bytes.NewReader(bodyData)
		}
		req, err := c.newRequest(ctx, method, path, accessToken, reader)
		if err != nil {
			return nil, err
		}
		if bodyData != nil {
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
		}
		return req, nil
	})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return decodeResponse(resp, out)
}

func (c *Client) Login(ctx context.Context, identity, password string) (*AuthResult, error) {
	var result AuthResult
	err := c.requestJSON(ctx, c.httpClient, http.MethodPost, "/auth/login", "", map[string]string{
		"identity": identity,
		"password": password,
	}, &result)

	return &result, err
}

func (c *Client) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	var result struct {
		Tokens TokenPair `json:"tokens"`
	}
	err := c.requestJSON(ctx, c.httpClient, http.MethodPost, "/auth/refresh", "", map[string]string{
		"refreshToken": refreshToken,
	}, &result)

	return &result.Tokens, err
}

func (c *Client) Me(ctx context.Context, link *models.CloudLink) (*MeResult, error) {
	var result MeResult
	err := c.authorizedJSON(ctx, c.httpClient, link, http.MethodGet, "/auth/me", nil, &result)

	return &result, err
}

func (c *Client) ListFavorites(ctx context.Context, link *models.CloudLink, page, pageSize int, keyword string) (*FavoritePage, error) {
	path := fmt.Sprintf("/favorites?page=%d&pageSize=%d&keyword=%s", page, pageSize, url.QueryEscape(keyword))
	var result FavoritePage
	err := c.authorizedJSON(ctx, c.httpClient, link, http.MethodGet, path, nil, &result)

	return &result, err
}

func (c *Client) Sync(ctx context.Context, link *models.CloudLink, in SyncRequest) (*SyncResult, error) {
	var result SyncResult
	err := c.authorizedJSON(ctx, c.syncClient, link, http.MethodPost, "/favorites/sync", in, &result)

	return &result, err
}

func (c *Client) ListTokens(ctx context.Context, link *models.CloudLink) (*TokenList, error) {
	var result TokenList
	err := c.authorizedJSON(ctx, c.httpClient, link, http.MethodGet, "/tokens", nil, &result)

	return &result, err
}

func (c *Client) ListGameEntries(ctx context.Context, link *models.CloudLink, kind string) (*GameEntryList, error) {
	path := "/game-admins"
	if kind == "slot" {
		path = "/game-reserved-slots"
	}
	var result GameEntryList
	err := c.authorizedJSON(ctx, c.httpClient, link, http.MethodGet, path, nil, &result)

	return &result, err
}

func (c *Client) ListArchives(ctx context.Context, link *models.CloudLink) (*ArchiveList, error) {
	var result ArchiveList
	err := c.authorizedJSON(ctx, c.httpClient, link, http.MethodGet, "/archives", nil, &result)

	return &result, err
}

func (c *Client) UploadArchive(ctx context.Context, link *models.CloudLink, filePath, name, remark, metadata string) (*Archive, error) {
	build := func(accessToken string) (*http.Request, error) {
		file, err := os.Open(filePath)
		if err != nil {
			return nil, err
		}

		pipeReader, pipeWriter := io.Pipe()
		writer := multipart.NewWriter(pipeWriter)
		go func() {
			defer file.Close()
			part, err := writer.CreateFormFile("file", filepath.Base(filePath))
			if err != nil {
				_ = pipeWriter.CloseWithError(err)
				return
			}
			if _, err = io.Copy(part, file); err != nil {
				_ = pipeWriter.CloseWithError(err)
				return
			}
			if err = writer.WriteField("name", name); err != nil {
				_ = pipeWriter.CloseWithError(err)
				return
			}
			if err = writer.WriteField("remark", remark); err != nil {
				_ = pipeWriter.CloseWithError(err)
				return
			}
			if err = writer.WriteField("metadata", metadata); err != nil {
				_ = pipeWriter.CloseWithError(err)
				return
			}
			if err = writer.Close(); err != nil {
				_ = pipeWriter.CloseWithError(err)
				return
			}
			_ = pipeWriter.Close()
		}()

		req, err := c.newRequest(ctx, http.MethodPost, "/archives", accessToken, pipeReader)
		if err != nil {
			_ = pipeReader.Close()
			return nil, err
		}
		req.Header.Set("Content-Type", writer.FormDataContentType())

		return req, nil
	}

	resp, err := c.doAuthorized(ctx, c.uploadClient, link, build)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result Archive
	if err = decodeResponse(resp, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *Client) DownloadArchive(ctx context.Context, link *models.CloudLink, id string) (io.ReadCloser, string, int64, error) {
	resp, err := c.doAuthorized(ctx, c.streamClient, link, func(accessToken string) (*http.Request, error) {
		return c.newRequest(ctx, http.MethodGet, "/archives/"+url.PathEscape(id)+"/download", accessToken, nil)
	})
	if err != nil {
		return nil, "", 0, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		if err = decodeResponse(resp, nil); err != nil {
			return nil, "", 0, err
		}
		return nil, "", 0, &APIError{StatusCode: resp.StatusCode, Message: "下载云存档失败"}
	}

	return resp.Body, downloadFileName(resp.Header.Get("Content-Disposition"), id), resp.ContentLength, nil
}

func downloadFileName(header, fallback string) string {
	if header == "" {
		return "archive-" + fallback + ".zip"
	}
	if index := strings.Index(header, "filename*=UTF-8''"); index >= 0 {
		encoded := header[index+len("filename*=UTF-8''"):]
		if end := strings.Index(encoded, ";"); end >= 0 {
			encoded = encoded[:end]
		}
		if decoded, err := url.PathUnescape(strings.TrimSpace(encoded)); err == nil && decoded != "" {
			return decoded
		}
	}
	_, params, err := mime.ParseMediaType(header)
	if err == nil {
		if name := strings.TrimSpace(params["filename"]); name != "" {
			return name
		}
	}

	return "archive-" + fallback + ".zip"
}

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
