package cloud

import (
	"context"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
)

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
	if err = s.saveLinkTokens(link); err != nil {
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
	if err = s.saveLinkTokens(link); err != nil {
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
	if err = s.saveLinkTokens(link); err != nil {
		logger.Logger.Errorf("保存云账号 token 失败, err: %v", err)
	}

	return service.Success(result)
}
