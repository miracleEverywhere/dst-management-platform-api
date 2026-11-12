// Package cloud 云账号、云收藏同步与云存档的业务实现，不依赖 gin。
package cloud

import (
	"context"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	roomsvc "dst-management-platform-api/service/room"
	"dst-management-platform-api/utils"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
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

// ===== 云账号与本地用户 =====

func (s *Service) getLink(username string) (*models.CloudLink, error) {
	return s.cloudLinkDao.GetByUsername(username)
}

func (s *Service) saveLink(link *models.CloudLink) error {
	link.UpdatedAt = nowSeconds()
	if link.DeviceID == "" {
		link.DeviceID = uuid.NewString()
	}
	if link.LinkedAt == 0 {
		link.LinkedAt = nowSeconds()
	}

	return s.cloudLinkDao.SaveLink(link)
}

func (s *Service) getLocalUser(username string) (*models.User, error) {
	return s.userDao.GetUserByUsername(username)
}

// applyAuthResult 把云登录 / 云绑定返回的账号信息写入本地绑定表。
func (s *Service) applyAuthResult(link *models.CloudLink, result *AuthResult, password string) error {
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

func (s *Service) refreshProfile(ctx context.Context, link *models.CloudLink) error {
	me, err := s.client.Me(ctx, link)
	if err != nil {
		return err
	}
	applyMeResult(link, me)
	if err = s.saveLink(link); err != nil {
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
func (s *Service) refreshProfileQuietly(ctx context.Context, link *models.CloudLink) {
	if err := s.refreshProfile(ctx, link); err != nil {
		logger.Logger.Warnf("刷新云套餐失败, username: %s, err: %v", link.Username, err)
	}
}

func (s *Service) issueLocalToken(user *models.User) (string, error) {
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

// ===== 云账号登录与绑定 =====

// Login 云账号登录。
// 数据库无用户时创建首个本地管理员；已有用户时必须先绑定云账号。
// 云服务网络错误/超时/5xx 时回退到本地账号校验。
func (s *Service) Login(ctx context.Context, actor service.Actor, identity, password string) service.Result {
	if identity == "" || password == "" {
		return service.Fail(constant.CodeCloudBadRequest)
	}

	userCount, err := s.userDao.Count(nil)
	if err != nil {
		logger.Logger.Errorf("查询用户数量失败, err: %v", err)
		return service.Fail(constant.CodeCloudDBError)
	}

	result, err := s.client.Login(ctx, identity, password)
	if err != nil {
		if IsUnavailable(err) {
			return s.fallbackLocalLogin(actor, identity, password)
		}
		return failCloud(actor, err)
	}

	if userCount == 0 {
		return s.bootstrapCloudUser(ctx, actor, result, password)
	}

	link, err := s.cloudLinkDao.GetByCloudAccountID(result.Account.ID)
	if err != nil {
		logger.Logger.Errorf("查询云账号绑定失败, err: %v", err)
		return service.Fail(constant.CodeCloudDBError)
	}
	if link.ID == 0 {
		return service.Fail(constant.CodeCloudNotLinked)
	}

	user, err := s.getLocalUser(link.Username)
	if err != nil {
		logger.Logger.Errorf("查询本地用户失败, err: %v", err)
		return service.Fail(constant.CodeCloudDBError)
	}
	if user.Username == "" {
		return service.Fail(constant.CodeCloudLocalUserMissing)
	}
	if user.Disabled {
		return service.Fail(constant.CodeCloudLocalUserDisabled)
	}

	if err = s.applyAuthResult(link, result, password); err != nil {
		logger.Logger.Errorf("保存云账号信息失败, err: %v", err)
		return service.Fail(constant.CodeCloudDBError)
	}
	if err = s.saveLink(link); err != nil {
		logger.Logger.Errorf("保存云账号绑定失败, err: %v", err)
		return service.Fail(constant.CodeCloudDBError)
	}

	// 登录后只刷新套餐与配额，不再自动同步收藏。
	s.refreshProfileQuietly(ctx, link)

	token, err := s.issueLocalToken(user)
	if err != nil {
		return service.Fail(constant.CodeCloudLoginFail)
	}

	view := cloudLoginView{
		Token: token, Username: user.Username, Plan: link.Plan, FavoriteQuota: link.FavoriteQuota,
	}

	return service.Msg("cloud login success", view)
}

// bootstrapCloudUser 数据库无用户时，用云账号创建首个本地管理员。
func (s *Service) bootstrapCloudUser(ctx context.Context, actor service.Actor, result *AuthResult, password string) service.Result {
	username := result.User.Username
	if username == "" {
		username = result.Account.Username
	}
	if username == "" {
		return service.Fail(constant.CodeCloudLoginFail)
	}

	passwordHash, passwordVersion, err := hashCloudPassword(password)
	if err != nil {
		logger.Logger.Errorf("创建bcrypt密码失败, err: %v", err)
		return service.Fail(constant.CodeCloudCreateFail)
	}

	user := models.User{
		Username:        username,
		Nickname:        username,
		Role:            "admin",
		Avatar:          "1",
		Password:        passwordHash,
		PasswordVersion: passwordVersion,
		Disabled:        false,
		RoomCreation:    true,
	}
	created := true
	var link *models.CloudLink
	if err = s.userDao.Create(&user); err != nil {
		// 并发首次登录时可能已被另一个请求创建，重新读取并确认云账号归属。
		existing, getErr := s.getLocalUser(username)
		if getErr != nil || existing.Username == "" {
			logger.Logger.Errorf("创建本地用户失败, err: %v", err)
			return service.Fail(constant.CodeCloudCreateFail)
		}
		existingLink, linkErr := s.cloudLinkDao.GetByCloudAccountID(result.Account.ID)
		if linkErr != nil {
			logger.Logger.Errorf("查询云账号绑定失败, err: %v", linkErr)
			return service.Fail(constant.CodeCloudDBError)
		}
		if existingLink.ID == 0 || existingLink.Username != existing.Username {
			// 本地账号已被另一个云账号或本地注册占用，必须走绑定流程。
			return service.Fail(constant.CodeCloudNotLinked)
		}
		user = *existing
		link = existingLink
		created = false
	}

	if link == nil {
		link = &models.CloudLink{Username: user.Username}
	}
	if err = s.applyAuthResult(link, result, password); err != nil {
		if created {
			_ = s.userDao.DeleteUser(user.Username)
		}
		logger.Logger.Errorf("保存云账号信息失败, err: %v", err)
		return service.Fail(constant.CodeCloudDBError)
	}
	if err = s.saveLink(link); err != nil {
		if created {
			_ = s.userDao.DeleteUser(user.Username)
		}
		logger.Logger.Errorf("保存云账号绑定失败, err: %v", err)
		return service.Fail(constant.CodeCloudDBError)
	}

	// 首次云登录只刷新套餐与配额，收藏由用户手动同步。
	s.refreshProfileQuietly(ctx, link)

	token, err := s.issueLocalToken(&user)
	if err != nil {
		return service.Fail(constant.CodeCloudLoginFail)
	}

	view := cloudLoginView{
		Token: token, Username: user.Username, Plan: link.Plan, FavoriteQuota: link.FavoriteQuota,
	}

	return service.Msg("cloud login success", view)
}

// fallbackLocalLogin 云服务不可用时，按云用户名/邮箱或本地用户名回退到本地校验。
func (s *Service) fallbackLocalLogin(actor service.Actor, identity, password string) service.Result {
	link, err := s.cloudLinkDao.GetByIdentity(identity)
	if err != nil {
		logger.Logger.Errorf("查询云账号绑定失败, err: %v", err)
		return service.Fail(constant.CodeCloudDBError)
	}

	var user *models.User
	if link.ID != 0 {
		user, err = s.getLocalUser(link.Username)
		if err != nil {
			logger.Logger.Errorf("查询本地用户失败, err: %v", err)
			return service.Fail(constant.CodeCloudDBError)
		}
		if user.Username == "" {
			return service.Fail(constant.CodeCloudLocalUserMissing)
		}
		if !validateCloudPassword(password, link.CloudPassword) && !utils.ValidatePassword(password, user.Password) {
			return service.Fail(constant.CodeCloudPasswordWrong)
		}
	} else {
		user, err = s.getLocalUser(identity)
		if err != nil {
			logger.Logger.Errorf("查询本地用户失败, err: %v", err)
			return service.Fail(constant.CodeCloudDBError)
		}
		if user.Username == "" || !utils.ValidatePassword(password, user.Password) {
			return service.Fail(constant.CodeCloudUnavailable)
		}
	}

	if user.Disabled {
		return service.Fail(constant.CodeCloudLocalUserDisabled)
	}

	token, err := s.issueLocalToken(user)
	if err != nil {
		return service.Fail(constant.CodeCloudLoginFail)
	}

	plan := "basic"
	favoriteQuota := 1
	if link.ID != 0 {
		plan = link.Plan
		favoriteQuota = link.FavoriteQuota
	}
	view := cloudLoginView{
		Token: token, Username: user.Username, Fallback: true, Plan: plan, FavoriteQuota: favoriteQuota,
	}

	return service.Msg("cloud local fallback success", view)
}

// Link 把当前本地账号绑定到云账号。绑定不修改本地用户名和本地密码。
func (s *Service) Link(ctx context.Context, actor service.Actor, identity, password string) service.Result {
	if identity == "" || password == "" {
		return service.Fail(constant.CodeCloudBadRequest)
	}

	username := actor.Username
	user, err := s.getLocalUser(username)
	if err != nil {
		logger.Logger.Errorf("查询本地用户失败, err: %v", err)
		return service.Fail(constant.CodeCloudDBError)
	}
	if user.Username == "" {
		return service.Fail(constant.CodeCloudLocalUserMissing)
	}

	result, err := s.client.Login(ctx, identity, password)
	if err != nil {
		return failCloud(actor, err)
	}

	existing, err := s.cloudLinkDao.GetByCloudAccountID(result.Account.ID)
	if err != nil {
		logger.Logger.Errorf("查询云账号绑定失败, err: %v", err)
		return service.Fail(constant.CodeCloudDBError)
	}
	if existing.ID != 0 && existing.Username != username {
		return service.Fail(constant.CodeCloudAlreadyLinked)
	}

	link, err := s.cloudLinkDao.GetByUsername(username)
	if err != nil {
		logger.Logger.Errorf("查询云账号绑定失败, err: %v", err)
		return service.Fail(constant.CodeCloudDBError)
	}
	if link.ID == 0 {
		link = &models.CloudLink{Username: username}
	} else if link.CloudAccountID != 0 && link.CloudAccountID != result.Account.ID {
		// 切换云账号时清除旧账号的收藏映射，避免把新账号的收藏关联到旧映射。
		if err = s.cloudMapDao.DeleteByUsername(username); err != nil {
			logger.Logger.Errorf("清理云收藏映射失败, err: %v", err)
			return service.Fail(constant.CodeCloudDBError)
		}
	}

	if err = s.applyAuthResult(link, result, password); err != nil {
		logger.Logger.Errorf("保存云账号信息失败, err: %v", err)
		return service.Fail(constant.CodeCloudDBError)
	}
	if err = s.saveLink(link); err != nil {
		logger.Logger.Errorf("保存云账号绑定失败, err: %v", err)
		return service.Fail(constant.CodeCloudDBError)
	}

	view := linkView(link)
	view["favoriteCount"] = s.localFavoriteCount(username)

	return service.Msg("cloud link success", view)
}

// Unlink 解除云账号绑定。需要本地密码或云账号密码确认，防止会话被劫持后直接解绑。
func (s *Service) Unlink(actor service.Actor, password string) service.Result {
	username := actor.Username
	unlock := lockUserSync(username)
	defer unlock()

	user, err := s.getLocalUser(username)
	if err != nil {
		logger.Logger.Errorf("查询本地用户失败, err: %v", err)
		return service.Fail(constant.CodeCloudDBError)
	}
	link, err := s.cloudLinkDao.GetByUsername(username)
	if err != nil {
		logger.Logger.Errorf("查询云账号绑定失败, err: %v", err)
		return service.Fail(constant.CodeCloudDBError)
	}
	if link.ID == 0 {
		return service.Fail(constant.CodeCloudLinkNotFound)
	}

	passwordOK := validateCloudPassword(password, link.CloudPassword)
	if user.Username != "" {
		passwordOK = passwordOK || utils.ValidatePassword(password, user.Password)
	}
	if !passwordOK {
		return service.Fail(constant.CodeCloudPasswordWrong)
	}

	if err = s.cloudLinkDao.DeleteByUsername(username); err != nil {
		logger.Logger.Errorf("解除云账号绑定失败, err: %v", err)
		return service.Fail(constant.CodeCloudDBError)
	}
	if err = s.cloudMapDao.DeleteByUsername(username); err != nil {
		logger.Logger.Errorf("清理云收藏映射失败, err: %v", err)
	}

	return service.Msg("cloud unlink success", map[string]any{"linked": false})
}

// Status 返回当前本地账号的云绑定状态。
func (s *Service) Status(actor service.Actor) service.Result {
	link, err := s.getLink(actor.Username)
	if err != nil {
		logger.Logger.Errorf("查询云账号绑定失败, err: %v", err)
		return service.Fail(constant.CodeCloudDBError)
	}

	view := linkView(link)
	view["favoriteCount"] = s.localFavoriteCount(actor.Username)

	return service.Success(view)
}

// Favorites 云收藏分页列表
func (s *Service) Favorites(ctx context.Context, actor service.Actor, page, pageSize int, keyword string) service.Result {
	link, failed := s.requireLink(actor)
	if failed != nil {
		return *failed
	}

	result, err := s.client.ListFavorites(ctx, link, page, pageSize, keyword)
	if err != nil {
		return failCloud(actor, err)
	}
	if err = s.saveLink(link); err != nil {
		logger.Logger.Errorf("保存云账号 token 失败, err: %v", err)
	}

	return service.Success(result)
}

// Tokens 云端 Klei 令牌列表
func (s *Service) Tokens(ctx context.Context, actor service.Actor) service.Result {
	link, failed := s.requireLink(actor)
	if failed != nil {
		return *failed
	}

	result, err := s.client.ListTokens(ctx, link)
	if err != nil {
		return failCloud(actor, err)
	}
	if err = s.saveLink(link); err != nil {
		logger.Logger.Errorf("保存云账号 token 失败, err: %v", err)
	}

	return service.Success(result)
}

// GameEntries 云端管理员（kind=admin）或预留位（kind=slot）列表
func (s *Service) GameEntries(ctx context.Context, actor service.Actor, kind string) service.Result {
	link, failed := s.requireLink(actor)
	if failed != nil {
		return *failed
	}

	result, err := s.client.ListGameEntries(ctx, link, kind)
	if err != nil {
		return failCloud(actor, err)
	}
	if err = s.saveLink(link); err != nil {
		logger.Logger.Errorf("保存云账号 token 失败, err: %v", err)
	}

	return service.Success(result)
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
