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
