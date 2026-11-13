package user

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
)

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
