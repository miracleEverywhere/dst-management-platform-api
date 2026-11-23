package mod

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
)

// maxBannedModsCount 全局屏蔽名单的条目上限，避免单次请求写入过大的名单
const maxBannedModsCount = 5000

// BannedModsGet 获取全局屏蔽模组名单
func (s *Service) BannedModsGet(actor service.Actor) service.Result {
	modIDs, err := s.bannedModDao.GetModIDs()
	if err != nil {
		logger.Logger.Errorf("获取全局屏蔽模组列表失败, err: %v, api: %s", err, actor.Path)
		return service.Fail(constant.CodeModBannedListGetFail)
	}

	return service.Success(modIDs)
}

// BannedModsPut 覆盖保存全局屏蔽模组名单，并同步投影到 banned_mods.txt
func (s *Service) BannedModsPut(actor service.Actor, modIDs []int) service.Result {
	ids := dst.NormalizeBannedModIDs(modIDs)
	if len(ids) > maxBannedModsCount {
		logger.Logger.Infof("全局屏蔽模组数量超限, count: %d, api: %s", len(ids), actor.Path)
		return service.Fail(constant.CodeModBadRequest)
	}

	if err := s.bannedModDao.Replace(ids); err != nil {
		logger.Logger.Errorf("保存全局屏蔽模组列表失败, err: %v, api: %s", err, actor.Path)
		return service.Fail(constant.CodeModDBError)
	}

	// 数据库是唯一真相源，文件写入失败不回滚：下次保存或服务启动会重新投影
	if err := dst.SaveBannedMods(ids); err != nil {
		logger.Logger.Errorf("写入全局屏蔽模组文件失败, err: %v, api: %s", err, actor.Path)
		return service.Fail(constant.CodeModBannedFileWriteFail)
	}

	return service.Msg("save banned mods success", nil)
}

// isModBanned 判断模组是否在全局屏蔽名单中。
// 查询失败时按未屏蔽处理：提示信息不该阻断启用流程。
func (s *Service) isModBanned(modID int) bool {
	if modID <= 0 {
		return false
	}

	banned, err := s.bannedModDao.Exist(modID)
	if err != nil {
		logger.Logger.Warnf("查询全局屏蔽名单失败, modID: %d, err: %v", modID, err)
		return false
	}

	return banned
}

// SyncBannedModsToFile 把数据库中的屏蔽名单投影到 banned_mods.txt，服务启动时调用。
// 数据库为空时保持文件不变：新部署或未使用该功能时，不清掉服主手工维护的名单。
func SyncBannedModsToFile(bannedModDao *dao.BannedModDAO) {
	modIDs, err := bannedModDao.GetModIDs()
	if err != nil {
		logger.Logger.Errorf("读取全局屏蔽模组列表失败, err: %v", err)
		return
	}
	if len(modIDs) == 0 {
		return
	}

	if err = dst.SaveBannedMods(modIDs); err != nil {
		logger.Logger.Errorf("同步全局屏蔽模组文件失败, err: %v", err)
	}
}
