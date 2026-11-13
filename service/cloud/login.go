package cloud

import (
	"context"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"sync"
)

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
		return s.bootstrapCloudUser(ctx, result, password)
	}

	return s.loginLinkedCloudUser(ctx, result, password)
}

// loginLinkedCloudUser 完成已绑定云账号的登录；绑定在用户锁内重新读取并确认。
func (s *Service) loginLinkedCloudUser(ctx context.Context, result *AuthResult, password string) service.Result {
	link, user, failed := s.authenticateLinkedCloudUser(result, password)
	if failed != nil {
		return *failed
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

// authenticateLinkedCloudUser 在用户锁内按云账号重新读取绑定、校验本地用户并保存登录结果。
// 锁覆盖读取、BCrypt 与写库，避免登录过程中绑定被解绑或切换后又被旧结果覆盖。
func (s *Service) authenticateLinkedCloudUser(result *AuthResult, password string) (*models.CloudLink, *models.User, *service.Result) {
	link, err := s.cloudLinkDao.GetByCloudAccountID(result.Account.ID)
	if err != nil {
		logger.Logger.Errorf("查询云账号绑定失败, err: %v", err)
		failed := service.Fail(constant.CodeCloudDBError)
		return nil, nil, &failed
	}
	if link.ID == 0 {
		failed := service.Fail(constant.CodeCloudNotLinked)
		return nil, nil, &failed
	}

	unlock := lockUserSync(link.Username)
	defer unlock()

	// 等锁期间绑定可能已被解绑或切换到其它云账号，必须重新读取。
	current, err := s.cloudLinkDao.GetByUsername(link.Username)
	if err != nil {
		logger.Logger.Errorf("查询云账号绑定失败, err: %v", err)
		failed := service.Fail(constant.CodeCloudDBError)
		return nil, nil, &failed
	}
	if current.ID == 0 || current.CloudAccountID != result.Account.ID {
		failed := service.Fail(constant.CodeCloudNotLinked)
		return nil, nil, &failed
	}
	link = current

	user, err := s.getLocalUser(link.Username)
	if err != nil {
		logger.Logger.Errorf("查询本地用户失败, err: %v", err)
		failed := service.Fail(constant.CodeCloudDBError)
		return nil, nil, &failed
	}
	if user.Username == "" {
		failed := service.Fail(constant.CodeCloudLocalUserMissing)
		return nil, nil, &failed
	}
	if user.Disabled {
		failed := service.Fail(constant.CodeCloudLocalUserDisabled)
		return nil, nil, &failed
	}

	if err = s.applyAuthResult(link, result, password); err != nil {
		logger.Logger.Errorf("保存云账号信息失败, err: %v", err)
		failed := service.Fail(constant.CodeCloudDBError)
		return nil, nil, &failed
	}
	if err = s.saveLink(link); err != nil {
		logger.Logger.Errorf("保存云账号绑定失败, err: %v", err)
		failed := service.Fail(constant.CodeCloudDBError)
		return nil, nil, &failed
	}

	return link, user, nil
}

// bootstrapMu 串行化首次管理员创建，避免多个云账号并发读到 userCount=0 后各建一个管理员。
var bootstrapMu sync.Mutex

// bootstrapCloudUser 数据库无用户时，用云账号创建首个本地管理员。
func (s *Service) bootstrapCloudUser(ctx context.Context, result *AuthResult, password string) service.Result {
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

	var (
		user   *models.User
		link   *models.CloudLink
		failed *service.Result
	)

	// 临界区只覆盖“复查用户数 + 创建用户 + 创建绑定 + 保存绑定”，
	// 网络刷新与签发 token 放在锁外。
	func() {
		bootstrapMu.Lock()
		defer bootstrapMu.Unlock()

		userCount, countErr := s.userDao.Count(nil)
		if countErr != nil {
			logger.Logger.Errorf("查询用户数量失败, err: %v", countErr)
			failedResult := service.Fail(constant.CodeCloudDBError)
			failed = &failedResult
			return
		}
		if userCount > 0 {
			// 另一个请求已完成首次初始化，转入正常的已绑定登录流程。
			// 这样同一个云账号的并发首次登录不会被误判为未绑定。
			return
		}

		candidate := models.User{
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
		if createErr := s.userDao.Create(&candidate); createErr != nil {
			// 并发首次登录时可能已被另一个请求创建，重新读取并确认云账号归属。
			existing, getErr := s.getLocalUser(username)
			if getErr != nil || existing.Username == "" {
				logger.Logger.Errorf("创建本地用户失败, err: %v", createErr)
				failedResult := service.Fail(constant.CodeCloudCreateFail)
				failed = &failedResult
				return
			}
			existingLink, linkErr := s.cloudLinkDao.GetByCloudAccountID(result.Account.ID)
			if linkErr != nil {
				logger.Logger.Errorf("查询云账号绑定失败, err: %v", linkErr)
				failedResult := service.Fail(constant.CodeCloudDBError)
				failed = &failedResult
				return
			}
			if existingLink.ID == 0 || existingLink.Username != existing.Username {
				// 本地账号已被另一个云账号或本地注册占用，必须走绑定流程。
				failedResult := service.Fail(constant.CodeCloudNotLinked)
				failed = &failedResult
				return
			}
			candidate = *existing
			link = existingLink
			created = false
		}

		if link == nil {
			link = &models.CloudLink{Username: candidate.Username}
		}
		// passwordHash 已在锁外计算，传入空密码避免在临界区内重复做 BCrypt。
		link.CloudPassword = passwordHash
		if applyErr := s.applyAuthResult(link, result, ""); applyErr != nil {
			if created {
				_ = s.userDao.DeleteUser(candidate.Username)
			}
			logger.Logger.Errorf("保存云账号信息失败, err: %v", applyErr)
			failedResult := service.Fail(constant.CodeCloudDBError)
			failed = &failedResult
			return
		}
		if saveErr := s.saveLink(link); saveErr != nil {
			if created {
				_ = s.userDao.DeleteUser(candidate.Username)
			}
			logger.Logger.Errorf("保存云账号绑定失败, err: %v", saveErr)
			failedResult := service.Fail(constant.CodeCloudDBError)
			failed = &failedResult
			return
		}

		user = &candidate
	}()

	if failed != nil {
		return *failed
	}
	if user == nil {
		return s.loginLinkedCloudUser(ctx, result, password)
	}

	// 首次云登录只刷新套餐与配额，收藏由用户手动同步。
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
