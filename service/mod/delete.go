package mod

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"strconv"
)

// Delete 删除单个模组
func (s *Service) Delete(actor service.Actor, roomID, id int, fileURL string) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	err = game.ModDelete(id, fileURL)
	if err != nil {
		logger.Logger.Errorf("删除模组失败, err: %v", err)
		return service.Fail(constant.CodeModDeleteFail)
	}

	err = cache.ModDownloadStatus.Delete(roomID, id)
	if err != nil {
		logger.Logger.Errorf("删除模组下载状态缓存失败: %v", err)
	}

	return service.Msg("delete success", nil)
}

// DeleteMulti 批量删除模组
func (s *Service) DeleteMulti(actor service.Actor, roomID int, mods []dst.DeleteModInfo) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	err = game.ModDeleteMulti(mods)
	if err != nil {
		logger.Logger.Errorf("删除模组失败, err: %v", err)
		return service.Fail(constant.CodeModDeleteFail)
	}

	for _, mod := range mods {
		err = cache.ModDownloadStatus.Delete(roomID, mod.ID)
		if err != nil {
			logger.Logger.Errorf("删除模组下载状态缓存失败: %v", err)
		}
	}

	return service.Msg("delete success", nil)
}

// DeleteAcf 删除 acf 文件
func (s *Service) DeleteAcf(actor service.Actor, roomID int) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	err = game.DeleteAcf()
	if err != nil {
		logger.Logger.Errorf("删除acf文件失败, err: %v", err)
		return service.Fail(constant.CodeModDeleteFail)
	}

	err = cache.ModDownloadStatus.DeleteByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("删除模组下载状态缓存失败: %v", err)
	}

	return service.Msg("delete success", nil)
}
