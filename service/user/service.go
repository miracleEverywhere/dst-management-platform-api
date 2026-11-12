package user

import (
	"crypto/sha512"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"encoding/hex"
)

// Service 用户模块业务实现，不依赖 gin
type Service struct {
	userDao *dao.UserDAO
}

// New 创建用户模块 service
func New(userDao *dao.UserDAO) *Service {
	return &Service{userDao: userDao}
}

// Registered 是否尚未注册任何用户
func (s *Service) Registered() service.Result {
	num, err := s.userDao.Count(nil)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Result{Code: 200, MsgKey: "database error", Data: false}
	}

	return service.Success(num == 0)
}

// Register 首次注册，注册的用户默认拥有最高权限
func (s *Service) Register(user models.User) service.Result {
	num, err := s.userDao.Count(nil)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeUserDBError)
	}

	if num != 0 {
		logger.Logger.Info("创建用户失败，用户已存在")
		return service.Fail(constant.CodeUserExist)
	}

	// 注册的用户默认拥有最高权限
	user.Disabled = false
	user.Role = "admin"

	// 设置密码
	password, err := utils.GenerateBcryptPassword(user.Password)
	if err != nil {
		logger.Logger.Errorf("创建bcrypt密码失败：%v", err)
		return service.Fail(constant.CodeUserCreateFail)
	}
	user.Password = password
	user.PasswordVersion = models.PasswordVersionBcrypt

	if errCreate := s.userDao.Create(&user); errCreate != nil {
		logger.Logger.Errorf("创建用户失败, err: %v", errCreate)
		return service.Fail(constant.CodeUserDBError)
	}

	return service.Msg("register success", nil)
}

// Login 校验密码并签发 token，兼容旧密码版本并在登录成功后升级为 bcrypt
func (s *Service) Login(user models.User) service.Result {
	dbUser, err := s.userDao.GetUserByUsername(user.Username)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeUserDBError)
	}

	if dbUser.Username == "" {
		return service.Fail(constant.CodeUserNotExist)
	}

	if dbUser.Disabled {
		return service.Fail(constant.CodeUserDisabled)
	}

	var (
		validated          bool
		needUpdatePassword bool
	)

	switch dbUser.PasswordVersion {
	case models.PasswordVersionBcrypt:
		validated = utils.ValidatePassword(user.Password, dbUser.Password)
	case models.PasswordVersionBcryptSha256:
		validated = utils.ValidateBcryptSha256Password(user.Password, dbUser.Password)
	case models.PasswordVersionSha512:
		hash := sha512.Sum512([]byte(user.Password))
		password := hex.EncodeToString(hash[:])

		validated = dbUser.Password == password
		if validated {
			needUpdatePassword = true
		}
	default:
		hash := sha512.Sum512([]byte(user.Password))
		password := hex.EncodeToString(hash[:])

		validated = dbUser.Password == password
		if validated {
			needUpdatePassword = true
		}
	}

	defer func() {
		if needUpdatePassword {
			password, err := utils.GenerateBcryptPassword(user.Password)
			if err != nil {
				logger.Logger.Errorf("创建bcrypt密码失败：%v", err)
				return
			}
			dbUser.Password = password
			dbUser.PasswordVersion = models.PasswordVersionBcrypt

			err = s.userDao.UpdateUser(dbUser)
			if err != nil {
				logger.Logger.Errorf("更新数据库失败, err: %v", err)
			} else {
				logger.Logger.Info("密码已升级至bcrypt")
			}
		}
	}()

	if !validated {
		return service.Fail(constant.CodeUserWrongPassword)
	}

	token, err := utils.GenerateJWT(*dbUser, []byte(cache.JwtSecret), constant.JwtExpirationHours)
	if err != nil {
		logger.Logger.Errorf("生成jwt失败, err: %v", err)
		return service.Fail(constant.CodeUserLoginFail)
	}

	// 登录成功后缓存 token 版本号
	cache.SetTokenVersion(dbUser.Username, dbUser.TokenVersion)

	return service.Msg("login success", token)
}

// Menu 当前角色可见的菜单
func (s *Service) Menu(actor service.Actor) service.Result {
	var data []menuItem

	if actor.IsAdmin() {
		data = []menuItem{
			rooms,
			dashboard,
			game,
			tools,
			logs,
			upload,
			install,
			platform,
		}
	} else {
		toolsLess := tools
		toolsLess.Links = []menuItem{
			tools.Links[0],
			tools.Links[1],
			tools.Links[2],
			tools.Links[4],
			tools.Links[5],
			tools.Links[6],
		}
		logsLess := logs
		logsLess.Links = []menuItem{
			logs.Links[0],
			logs.Links[1],
			logs.Links[2],
		}

		data = []menuItem{
			rooms,
			dashboard,
			game,
			toolsLess,
			logsLess,
			upload,
		}
	}

	return service.Success(data)
}

// CreateUser 管理员创建用户
func (s *Service) CreateUser(user models.User) service.Result {
	dbUser, err := s.userDao.GetUserByUsername(user.Username)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeUserDBError)
	}

	if dbUser.Username != "" {
		logger.Logger.Info("创建用户失败，用户已存在")
		return service.Fail(constant.CodeUserExist)
	}

	// 设置密码
	password, err := utils.GenerateBcryptPassword(user.Password)
	if err != nil {
		logger.Logger.Errorf("创建bcrypt密码失败：%v", err)
		return service.Fail(constant.CodeUserCreateFail)
	}
	user.Password = password
	user.PasswordVersion = models.PasswordVersionBcrypt

	if errCreate := s.userDao.Create(&user); errCreate != nil {
		logger.Logger.Errorf("创建用户失败, err: %v", errCreate)
		return service.Fail(constant.CodeUserCreateFail)
	}

	return service.Msg("create success", nil)
}

// Base 当前登录用户的基础信息
func (s *Service) Base(actor service.Actor) service.Result {
	dbUser, err := s.userDao.GetUserByUsername(actor.Username)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeUserDBError)
	}
	dbUser.Password = ""
	dbUser.PasswordVersion = ""

	return service.Success(dbUser)
}

// UpdateUser 管理员更新用户；禁用用户或修改角色时撤销其所有 token
func (s *Service) UpdateUser(user models.User) service.Result {
	dbUser, err := s.userDao.GetUserByUsername(user.Username)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeUserDBError)
	}
	if dbUser.Username == "" {
		return service.Fail(constant.CodeUserNotExist)
	}

	// 如果管理员禁用了用户或修改了角色，撤销其所有 token
	if (!dbUser.Disabled && user.Disabled) || dbUser.Role != user.Role {
		dbUser.TokenVersion = cache.RevokeTokenVersion(dbUser.Username, dbUser.TokenVersion)
	}

	// basePut不涉及密码修改
	user.Password = dbUser.Password
	user.PasswordVersion = dbUser.PasswordVersion
	user.TokenVersion = dbUser.TokenVersion

	err = s.userDao.UpdateUser(&user)
	if err != nil {
		logger.Logger.Errorf("更新数据库失败, err: %v", err)
		return service.Fail(constant.CodeUserUpdateFail)
	}

	return service.Msg("update success", nil)
}

// DeleteUser 管理员删除用户，用户数小于等于 1 时禁止删除
func (s *Service) DeleteUser(user models.User) service.Result {
	// 用户数小于等于1时，禁止删除
	num, err := s.userDao.Count(nil)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeUserDBError)
	}

	if num <= 1 {
		return service.Fail(constant.CodeUserDeleteAllForbidden)
	}

	// 查询用户是否存在
	dbUser, err := s.userDao.GetUserByUsername(user.Username)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeUserDBError)
	}
	if dbUser.Username == "" {
		return service.Fail(constant.CodeUserNotExist)
	}

	// 撤销被删除用户的所有 token
	cache.RevokeTokenVersion(dbUser.Username, dbUser.TokenVersion)

	// 执行删除
	err = s.userDao.Delete(dbUser)
	if err != nil {
		logger.Logger.Errorf("更新数据库失败, err: %v", err)
		return service.Fail(constant.CodeUserDeleteFail)
	}

	return service.Msg("delete success", nil)
}

// UserList 管理员分页查询用户，返回结果不包含密码字段
func (s *Service) UserList(actor service.Actor, q string, page, pageSize int) service.Result {
	var data dao.PaginatedResult[models.User]

	if !actor.IsAdmin() {
		return service.Result{Code: constant.CodeUserNoPermission, Data: data}
	}

	users, err := s.userDao.ListUsers(q, page, pageSize)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Result{Code: constant.CodeUserDBError, Data: data}
	}

	data.Data = []models.User{} // 防止Data为nil
	for _, user := range users.Data {
		user.Password = ""
		user.PasswordVersion = ""
		data.Data = append(data.Data, user)
	}

	data.Page = users.Page
	data.PageSize = users.PageSize
	data.TotalCount = users.TotalCount

	return service.Success(data)
}

// UpdateMyself 当前登录用户修改自己的信息，修改密码后撤销所有旧 token
func (s *Service) UpdateMyself(actor service.Actor, user models.User) service.Result {
	// 使用 JWT token 中的用户名，防止越权修改
	user.Username = actor.Username

	dbUser, err := s.userDao.GetUserByUsername(user.Username)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeUserDBError)
	}

	if user.Password != "" {
		password, err := utils.GenerateBcryptPassword(user.Password)
		if err != nil {
			logger.Logger.Errorf("创建bcrypt密码失败：%v", err)
		} else {
			dbUser.Password = password
			dbUser.PasswordVersion = models.PasswordVersionBcrypt
			// 修改密码后撤销所有旧 token
			dbUser.TokenVersion = cache.RevokeTokenVersion(dbUser.Username, dbUser.TokenVersion)
		}
	}
	if user.Nickname != "" {
		dbUser.Nickname = user.Nickname
	}
	if user.Avatar != "" {
		dbUser.Avatar = user.Avatar
	}

	err = s.userDao.UpdateUser(dbUser)
	if err != nil {
		logger.Logger.Errorf("更新数据库失败, err: %v", err)
		return service.Fail(constant.CodeUserUpdateFail)
	}

	return service.Msg("myself update success", nil)
}

// Revoke 管理员撤销指定用户的所有 token
func (s *Service) Revoke(username string) service.Result {
	dbUser, err := s.userDao.GetUserByUsername(username)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeUserDBError)
	}
	if dbUser.Username == "" {
		return service.Fail(constant.CodeUserNotExist)
	}

	// 递增 token 版本号，使所有旧 token 失效
	dbUser.TokenVersion = cache.RevokeTokenVersion(dbUser.Username, dbUser.TokenVersion)
	err = s.userDao.UpdateUser(dbUser)
	if err != nil {
		logger.Logger.Errorf("更新数据库失败, err: %v", err)
		return service.Fail(constant.CodeUserRevokeFail)
	}

	logger.Logger.Infof("管理员已撤销用户 %s 的所有 token", username)

	return service.Msg("revoke success", nil)
}

// ---- 菜单数据 ----

type menuItem struct {
	ID        int        `json:"id"`
	Type      string     `json:"type"`
	Section   string     `json:"section"`
	Title     string     `json:"title"`
	To        string     `json:"to"`
	Component string     `json:"component"`
	Icon      string     `json:"icon"`
	Links     []menuItem `json:"links"`
}

var rooms = menuItem{
	ID:        1,
	Type:      "link",
	Section:   "",
	Title:     "rooms",
	To:        "/rooms",
	Component: "rooms/index",
	Icon:      "ri-instance-line",
	Links:     nil,
}

var dashboard = menuItem{
	ID:        2,
	Type:      "link",
	Section:   "",
	Title:     "dashboard",
	To:        "/dashboard",
	Component: "dashboard/index",
	Icon:      "ri-function-ai-line",
	Links:     nil,
}

var game = menuItem{
	ID:        3,
	Type:      "group",
	Section:   "",
	Title:     "game",
	To:        "/game",
	Component: "",
	Icon:      "ri-gamepad-line",
	Links: []menuItem{
		{
			ID:        301,
			Type:      "link",
			Section:   "",
			Title:     "gameBase",
			To:        "/game/base",
			Component: "game/base",
			Icon:      "ri-sword-line",
			Links:     nil,
		},
		{
			ID:        302,
			Type:      "link",
			Section:   "",
			Title:     "gameMod",
			To:        "/game/mod",
			Component: "game/mod",
			Icon:      "ri-rocket-2-line",
			Links:     nil,
		},
		{
			ID:        303,
			Type:      "link",
			Section:   "",
			Title:     "gamePlayer",
			To:        "/game/player",
			Component: "game/player",
			Icon:      "ri-ghost-line",
			Links:     nil,
		},
		{
			ID:        304,
			Type:      "link",
			Section:   "",
			Title:     "gameFavorite",
			To:        "/game/favorite",
			Component: "game/favorite",
			Icon:      "ri-bookmark-3-line",
			Links:     nil,
		},
	},
}

var upload = menuItem{
	ID:        4,
	Type:      "link",
	Section:   "",
	Title:     "upload",
	To:        "/upload",
	Component: "upload/index",
	Icon:      "ri-contacts-book-upload-line",
	Links:     nil,
}

var install = menuItem{
	ID:        5,
	Type:      "link",
	Section:   "",
	Title:     "install",
	To:        "/install",
	Component: "install/index",
	Icon:      "ri-import-line",
	Links:     nil,
}

var tools = menuItem{
	ID:        6,
	Type:      "group",
	Section:   "",
	Title:     "tools",
	To:        "/tools",
	Component: "",
	Icon:      "ri-wrench-line",
	Links: []menuItem{
		{
			ID:        601,
			Type:      "link",
			Section:   "",
			Title:     "toolsBackup",
			To:        "/tools/backup",
			Component: "tools/backup",
			Icon:      "ri-save-2-line",
			Links:     nil,
		},
		{
			ID:        602,
			Type:      "link",
			Section:   "",
			Title:     "toolsAnnounce",
			To:        "/tools/announce",
			Component: "tools/announce",
			Icon:      "ri-chat-smile-ai-3-line",
			Links:     nil,
		},
		{
			ID:        603,
			Type:      "link",
			Section:   "",
			Title:     "toolsMap",
			To:        "/tools/map",
			Component: "tools/map",
			Icon:      "ri-road-map-line",
			Links:     nil,
		},
		{
			ID:        604,
			Type:      "link",
			Section:   "",
			Title:     "toolsToken",
			To:        "/tools/token",
			Component: "tools/token",
			Icon:      "ri-coupon-line",
			Links:     nil,
		},
		{
			ID:        605,
			Type:      "link",
			Section:   "",
			Title:     "toolsSnapshot",
			To:        "/tools/snapshot",
			Component: "tools/snapshot",
			Icon:      "ri-flip-horizontal-line",
			Links:     nil,
		},
		{
			ID:        606,
			Type:      "link",
			Section:   "",
			Title:     "toolsTmi",
			To:        "/tools/tmi",
			Component: "tools/tmi",
			Icon:      "ri-ghost-4-line",
			Links:     nil,
		},
		{
			ID:        607,
			Type:      "link",
			Section:   "",
			Title:     "toolsAI",
			To:        "/tools/aichat",
			Component: "tools/aichat",
			Icon:      "ri-chat-smile-ai-3-line",
			Links:     nil,
		},
		{
			ID:        608,
			Type:      "link",
			Section:   "",
			Title:     "toolsRoomMetrics",
			To:        "/tools/room_metrics",
			Component: "tools/room_metrics",
			Icon:      "ri-dashboard-2-line",
			Links:     nil,
		},
	},
}

var logs = menuItem{
	ID:        7,
	Type:      "group",
	Section:   "",
	Title:     "logs",
	To:        "/logs",
	Component: "",
	Icon:      "ri-blogger-line",
	Links: []menuItem{
		{
			ID:        701,
			Type:      "link",
			Section:   "",
			Title:     "logsGame",
			To:        "/logs/game",
			Component: "logs/game",
			Icon:      "ri-game-line",
			Links:     nil,
		},
		{
			ID:        702,
			Type:      "link",
			Section:   "",
			Title:     "logsChat",
			To:        "/logs/chat",
			Component: "logs/chat",
			Icon:      "ri-chat-smile-3-line",
			Links:     nil,
		},
		{
			ID:        703,
			Type:      "link",
			Section:   "",
			Title:     "logsDownload",
			To:        "/logs/download",
			Component: "logs/download",
			Icon:      "ri-download-2-line",
			Links:     nil,
		},
		{
			ID:        704,
			Type:      "link",
			Section:   "",
			Title:     "logsSteam",
			To:        "/logs/steam",
			Component: "logs/steam",
			Icon:      "ri-steam-line",
			Links:     nil,
		},
		{
			ID:        705,
			Type:      "link",
			Section:   "",
			Title:     "logsAccess",
			To:        "/logs/access",
			Component: "logs/access",
			Icon:      "ri-code-box-line",
			Links:     nil,
		},
		{
			ID:        706,
			Type:      "link",
			Section:   "",
			Title:     "logsRuntime",
			To:        "/logs/runtime",
			Component: "logs/runtime",
			Icon:      "ri-terminal-box-line",
			Links:     nil,
		},
		{
			ID:        707,
			Type:      "link",
			Section:   "",
			Title:     "logsClean",
			To:        "/logs/clean",
			Component: "logs/clean",
			Icon:      "ri-file-shred-line",
			Links:     nil,
		},
	},
}

var platform = menuItem{
	ID:        8,
	Type:      "link",
	Section:   "",
	Title:     "platform",
	To:        "/platform",
	Component: "platform/index",
	Icon:      "ri-vip-crown-2-line",
	Links:     nil,
}
