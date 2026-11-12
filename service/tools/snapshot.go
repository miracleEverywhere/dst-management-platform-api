package tools

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"errors"
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

	// 记录操作前正在运行的世界：无论删除成功还是失败，都恢复到该状态
	runningWorlds := make([]int, 0, len(*worlds))
	for _, world := range *worlds {
		if game.WorldUpStatus(world.ID) {
			runningWorlds = append(runningWorlds, world.ID)
		}
	}
	restoreWorlds := func() error {
		var errs []error
		for _, worldID := range runningWorlds {
			if startErr := game.StartWorld(worldID); startErr != nil {
				errs = append(errs, startErr)
			}
		}
		return errors.Join(errs...)
	}

	// 关闭游戏
	err = game.StopAllWorld()
	if err != nil {
		logger.Logger.Warnf("关闭游戏失败：%v，可能是游戏未运行，跳过", err)
	}

	// 删除存档文件
	deleteErr := game.DeleteSnapshot(name)
	startErr := restoreWorlds()
	if startErr == nil && len(runningWorlds) > 0 {
		// 用 StartWorld 逐个恢复时不会触发 StartAllWorld 内的 webhook，这里补发
		webhook.Snd.Send(webhook.EventGameStart, room.ID, map[string]interface{}{
			"gameID":   room.ID,
			"gameName": room.GameName,
		})
	}
	if deleteErr != nil {
		logger.Logger.Errorf("删除游戏存档文件失败, err: %v", deleteErr)
		if startErr != nil {
			logger.Logger.Errorf("删除失败后恢复游戏运行失败, err: %v", startErr)
		}
		return service.Fail(constant.CodeToolsDeleteFail)
	}
	if startErr != nil {
		logger.Logger.Errorf("快照已删除，但重新启动游戏失败, err: %v", startErr)
		return service.Fail(constant.CodeToolsSnapshotStartFail)
	}

	return service.Msg("delete success", nil)
}
