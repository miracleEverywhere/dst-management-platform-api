package cloud

import (
	"context"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"errors"

	"github.com/google/uuid"
)

// ===== 云账号与本地用户 =====
func (s *Service) getLink(username string) (*models.CloudLink, error) {
	return s.cloudLinkDao.GetByUsername(username)
}

// saveLink 全量写入绑定，只允许在持有用户锁、确认绑定归属后调用。
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

// saveLinkTokens 只回写 token，并以读取时的 CloudAccountID 做条件，
// 防止等待云端响应期间绑定被切换后旧请求覆盖新绑定。
func (s *Service) saveLinkTokens(link *models.CloudLink) error {
	link.UpdatedAt = nowSeconds()
	affected, err := s.cloudLinkDao.UpdateTokensIfAccount(link)
	if err != nil {
		return err
	}
	if affected == 0 {
		return errLinkChanged
	}

	return nil
}

// saveLinkProfile 只回写套餐、配额与 token，并校验绑定仍属于原云账号。
func (s *Service) saveLinkProfile(link *models.CloudLink) error {
	link.UpdatedAt = nowSeconds()
	affected, err := s.cloudLinkDao.UpdateProfileIfAccount(link)
	if err != nil {
		return err
	}
	if affected == 0 {
		return errLinkChanged
	}

	return nil
}

// saveLinkSyncState 回写同步状态、套餐、配额与 token，并校验绑定仍属于原云账号。
func (s *Service) saveLinkSyncState(link *models.CloudLink) error {
	link.UpdatedAt = nowSeconds()
	affected, err := s.cloudLinkDao.UpdateSyncStateIfAccount(link)
	if err != nil {
		return err
	}
	if affected == 0 {
		return errLinkChanged
	}

	return nil
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
	if err = s.saveLinkProfile(link); err != nil {
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
