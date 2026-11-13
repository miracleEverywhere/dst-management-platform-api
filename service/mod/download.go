package mod

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"fmt"
	"sort"
	"strconv"
)

// ModDownload 下载模组，sync 为 true 时同步下载，否则后台下载
func (s *Service) ModDownload(actor service.Actor, roomID, id int, fileURL, size, name string, sync bool) service.Result {
	// url https://cdn.steamusercontent.com/ugc/1466437966115152320/8A3E11F0B32FCBFBF308DEB0B5C98A702215374B/
	if ok := checkNotUgcUrl(fileURL); !ok {
		logger.Logger.Warnf("异常模组下载请求已拦截: %s, api: %s", fileURL, actor.Path)
		logger.Logger.Warn("疑似攻击请求，请立即修改用户名密码及重置jwt-secret")
		return service.Fail(constant.CodeModBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	reqSize, err := strconv.Atoi(size)
	if err != nil || reqSize <= 0 || roomID <= 0 || id <= 0 {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, actor.Path)
		return service.Fail(constant.CodeModBadRequest)
	}

	reqSize64 := int64(reqSize)

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	if cache.ModDownloadStatus == nil {
		logger.Logger.Error("模组下载状态缓存未初始化")
		return service.Fail(constant.CodeModServerError)
	}

	if item, err := cache.ModDownloadStatus.Get(roomID, id); err == nil && item.CurrentSize < item.Size {
		return service.Result{
			Code: 200,
			MsgBuilder: func(tr func(msgKey string, args ...any) string) string {
				return name + " " + tr("downloading")
			},
		}
	}

	if err := cache.ModDownloadStatus.Set(roomID, &cache.ModItem{
		ID:          id,
		Size:        reqSize,
		CurrentSize: 0,
	}); err != nil {
		logger.Logger.Errorf("写入模组下载状态失败, err: %v", err)
		return service.Fail(constant.CodeModServerError)
	}

	dl := func() error {
		err, modSize := game.DownloadMod(id, fileURL)
		if err != nil || modSize != reqSize64 {
			logger.Logger.Debugf("模组大小与预期不符, %d, %d, err: %v", modSize, reqSize64, err)
			if deleteErr := cache.ModDownloadStatus.Delete(roomID, id); deleteErr != nil {
				logger.Logger.Debugf("删除模组下载状态失败, err: %v", deleteErr)
			}
			if err == nil {
				err = fmt.Errorf("模组大小与预期不符: 实际 %d, 预期 %d", modSize, reqSize64)
			}
			logger.Logger.Warnf("模组下载失败, roomID: %d, modID: %d, err: %v", roomID, id, err)
			return err
		}

		if setErr := cache.ModDownloadStatus.Set(roomID, &cache.ModItem{
			ID:          id,
			Size:        reqSize,
			CurrentSize: reqSize,
		}); setErr != nil {
			logger.Logger.Errorf("更新模组下载状态失败, err: %v", setErr)
		}

		return nil
	}

	if sync {
		if err := dl(); err != nil {
			return service.Fail(constant.CodeModDownloadFail)
		}
		return service.MsgF("downloaded", []any{name}, nil)
	}

	go func() { _ = dl() }()
	return service.MsgF("downloading", []any{name}, nil)
}

// ModDownloadStatus 查询模组下载进度
func (s *Service) ModDownloadStatus(actor service.Actor, roomID, id int) service.Result {
	progress := 0
	if cache.ModDownloadStatus != nil {
		if item, err := cache.ModDownloadStatus.Get(roomID, id); err == nil {
			if item.Size > 0 && item.CurrentSize >= item.Size {
				progress = 100
			} else if item.Size > 0 && item.CurrentSize > 0 {
				progress = int(float64(item.CurrentSize) / float64(item.Size) * 100)
			} else if item.CurrentSize > 0 {
				progress = 100
			}
			if progress < 0 {
				progress = 0
			}
			if progress > 100 {
				progress = 100
			}
		}
	}

	return service.Success(progress)
}

// DownloadedMods 已下载模组列表
func (s *Service) DownloadedMods(actor service.Actor, roomID int) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeModNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeModDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	downloadedMods := game.GetDownloadedMods()

	if err = s.addDownloadedModInfo(downloadedMods, actor.Lang); err != nil {
		logger.Logger.Error("添加模组额外信息失败")
	}

	return service.Success(downloadedMods)
}

// DownloadedModIDs 已下载模组 ID 列表
func (s *Service) DownloadedModIDs(actor service.Actor, roomID int) service.Result {
	if roomID == 0 {
		return service.Result{Code: constant.CodeModBadRequest, Data: []int{}}
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Result{Code: constant.CodeModNoPermission, Data: []int{}}
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Result{Code: constant.CodeModDBError, Data: []int{}}
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	downloadedMods := game.GetDownloadedMods()
	modIDs := make([]int, 0, len(*downloadedMods))
	seen := make(map[int]struct{}, len(*downloadedMods))
	for _, downloadedMod := range *downloadedMods {
		if downloadedMod.ID <= 0 {
			continue
		}
		if _, exists := seen[downloadedMod.ID]; exists {
			continue
		}

		seen[downloadedMod.ID] = struct{}{}
		modIDs = append(modIDs, downloadedMod.ID)
	}
	sort.Ints(modIDs)

	return service.Success(modIDs)
}
