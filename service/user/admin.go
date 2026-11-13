package user

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
)

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
