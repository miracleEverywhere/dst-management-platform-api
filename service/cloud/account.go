package cloud

import (
	"context"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
)

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

	// 绑定/切换必须与同步、解绑串行，且读取-修改-写回在同一临界区内完成。
	unlock := lockUserSync(username)
	defer unlock()

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
