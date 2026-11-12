package tools

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
)

// Token 为当前用户签发长期 token，expiration 为 0 时签发 99 年的永久 token
func (s *Service) Token(actor service.Actor, expiration int) service.Result {
	if expiration < 0 {
		logger.Logger.Infof("请求参数错误, api: %s", actor.Path)
		return service.Fail(constant.CodeToolsBadRequest)
	}

	cache.TokenVersionCacheLock.RLock()
	cachedVersion, exists := cache.TokenVersionCache[actor.Username]
	cache.TokenVersionCacheLock.RUnlock()
	if !exists {
		cachedVersion = 0
	}

	user := models.User{
		Username:     actor.Username,
		Nickname:     actor.Nickname,
		Role:         "admin",
		TokenVersion: cachedVersion,
	}

	var expirationHours int

	if expiration == 0 {
		// 生成永久token 99年
		expirationHours = 99 * 365 * 24
	} else {
		expirationHours = expiration
	}

	token, err := utils.GenerateJWT(user, []byte(cache.JwtSecret), expirationHours)
	if err != nil {
		logger.Logger.Errorf("创建token失败, err: %v", err)
		return service.Fail(constant.CodeToolsCreateFail)
	}

	return service.Msg("create success", token)
}
