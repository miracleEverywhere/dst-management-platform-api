package tools

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"strconv"
)

// SnapshotGet 获取游戏存档快照列表
func (s *Service) SnapshotGet(actor service.Actor, roomID int) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodeToolsBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeToolsNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeToolsDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	snapshot, err := game.GetSnapshot()
	if err != nil {
		logger.Logger.Errorf("获取游戏存档文件失败, err: %v", err)
		return service.Result{Code: constant.CodeToolsGetSnapshotFail, Data: snapshot}
	}

	return service.Success(snapshot)
}

// SnapshotDelete 删除快照：先停服，删除后重新启动
func (s *Service) SnapshotDelete(actor service.Actor, roomID int, name string) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodeToolsBadRequest)
	}

	if !utils.IsSafePath(name) {
		return service.Fail(constant.CodeToolsBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeToolsNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeToolsDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)

	// 关闭游戏
	err = game.StopAllWorld()
	if err != nil {
		logger.Logger.Warnf("关闭游戏失败：%v，可能是游戏未运行，跳过", err)
	}

	// 删除存档文件
	err = game.DeleteSnapshot(name)
	if err != nil {
		logger.Logger.Errorf("删除游戏存档文件失败, err: %v", err)
		return service.Fail(constant.CodeToolsDeleteFail)
	}

	// 启动游戏
	err = game.StartAllWorld()
	if err != nil {
		logger.Logger.Errorf("启动游戏失败：%v", err)
	}

	return service.Msg("delete success", nil)
}
