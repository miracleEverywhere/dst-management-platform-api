package mod

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"strconv"
	"strings"
)

// DeleteMultiResult 批量删除结果
type DeleteMultiResult struct {
	SuccessIDs []int `json:"successIDs"`
	FailedIDs  []int `json:"failedIDs"`
}

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

	if err = game.DeleteUserCachedMod(actor.Username, id); err != nil {
		logger.Logger.Warnf("清理用户模组缓存失败, username: %s, modID: %d, err: %v", actor.Username, id, err)
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

	successIDs := make([]int, 0, len(mods))
	failedIDs := make([]int, 0)

	for _, mod := range mods {
		if err = game.ModDelete(mod.ID, mod.FileUrl); err != nil {
			logger.Logger.Errorf("删除模组失败, username: %s, modID: %d, err: %v", actor.Username, mod.ID, err)
			failedIDs = append(failedIDs, mod.ID)
			continue
		}

		if err = game.DeleteUserCachedMod(actor.Username, mod.ID); err != nil {
			logger.Logger.Errorf("清理用户模组缓存失败, username: %s, modID: %d, err: %v", actor.Username, mod.ID, err)
			failedIDs = append(failedIDs, mod.ID)
			continue
		}

		successIDs = append(successIDs, mod.ID)

		err = cache.ModDownloadStatus.Delete(roomID, mod.ID)
		if err != nil {
			logger.Logger.Errorf("删除模组下载状态缓存失败: %v", err)
		}
	}

	data := DeleteMultiResult{
		SuccessIDs: successIDs,
		FailedIDs:  failedIDs,
	}

	if len(failedIDs) == 0 {
		return service.Msg("delete success", nil)
	}
	if len(successIDs) == 0 {
		return service.FailData(constant.CodeModDeleteFail, data)
	}

	return service.MsgF("delete partial success", []any{len(successIDs), len(failedIDs), joinModIDs(failedIDs)}, data)
}

func joinModIDs(ids []int) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = strconv.Itoa(id)
	}

	return strings.Join(parts, ", ")
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
