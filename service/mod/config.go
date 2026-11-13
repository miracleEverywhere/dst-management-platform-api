package mod

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"strconv"
)

// SettingModConfigStruct 模组配置结构
func (s *Service) SettingModConfigStruct(actor service.Actor, roomID, worldID, id int, fileURL string) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	options, err := game.GetModConfigureOptions(worldID, id, fileURL == "")
	if err != nil {
		logger.Logger.Error("获取模组设置失败")
		return service.Result{Code: constant.CodeModConfigOptionsError, Data: options}
	}

	return service.Success(options)
}

// SettingModConfigValue 模组配置值
func (s *Service) SettingModConfigValue(actor service.Actor, roomID, worldID, id int, fileURL string) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	options, err := game.GetModConfigureOptionsValues(worldID, id, fileURL == "")
	if err != nil {
		logger.Logger.Error("获取模组设置失败")
		return service.Result{Code: constant.CodeModConfigValuesError, Data: options}
	}

	return service.Success(options)
}

// SettingModConfigValuePut 修改模组配置值
func (s *Service) SettingModConfigValuePut(actor service.Actor, roomID, worldID, id int, modORConfig dst.ModORConfig) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	err = game.ModConfigureOptionsValuesChange(worldID, id, &modORConfig)
	if err != nil {
		logger.Logger.Error("修改模组设置失败")
		return service.Fail(constant.CodeModModifyConfigError)
	}

	if err = s.roomDao.UpdateRoom(room); err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	if err = s.worldDao.UpdateWorlds(worlds); err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	return service.Msg("modify mod configuration values success", nil)
}

// AddEnable 启用模组
func (s *Service) AddEnable(actor service.Actor, roomID, worldID, id int, fileURL string) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	err = game.ModEnable(worldID, id, fileURL == "")
	if err != nil {
		logger.Logger.Errorf("模组启用失败, err: %v", err)
		return service.Fail(constant.CodeModEnableFail)
	}

	if err = s.roomDao.UpdateRoom(room); err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	if err = s.worldDao.UpdateWorlds(worlds); err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	return service.Msg("mod enable success", nil)
}

// AddDisable 禁用模组
func (s *Service) AddDisable(actor service.Actor, roomID, worldID, id int) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	err = game.ModDisable(id)
	if err != nil {
		logger.Logger.Errorf("模组禁用失败, err: %v", err)
		return service.Fail(constant.CodeModDisableFail)
	}

	if err = s.roomDao.UpdateRoom(room); err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	if err = s.worldDao.UpdateWorlds(worlds); err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	return service.Msg("mod disable success", nil)
}

// EnabledMods 已启用模组列表
func (s *Service) EnabledMods(actor service.Actor, roomID, worldID int) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	modsID, err := game.GetEnabledMods(worldID)
	if err != nil {
		logger.Logger.Error("获取模组设置失败")
		return service.Result{Code: constant.CodeModGetEnabledFail, Data: modsID}
	}

	if err = s.addDownloadedModInfo(&modsID, actor.Lang); err != nil {
		logger.Logger.Error("添加模组额外信息失败")
	}

	return service.Success(modsID)
}

// ModInfo 批量查询模组信息
func (s *Service) ModInfo(actor service.Actor, inputIDs []int) service.Result {
	modIDs := make([]int, 0, len(inputIDs))
	seen := make(map[int]struct{}, len(inputIDs))
	for _, modID := range inputIDs {
		if modID <= 0 {
			continue
		}
		if _, ok := seen[modID]; ok {
			continue
		}

		seen[modID] = struct{}{}
		modIDs = append(modIDs, modID)
	}

	mods := make([]dst.DownloadedMod, 0, len(modIDs))
	for _, modID := range modIDs {
		mods = append(mods, dst.DownloadedMod{ID: modID})
	}

	if len(mods) > 0 {
		err := s.addDownloadedModInfo(&mods, actor.Lang)
		if err != nil {
			logger.Logger.Errorf("获取模组信息失败, err: %v", err)
			return service.Fail(constant.CodeModSearchFail)
		}
	}

	return service.Success(mods)
}
