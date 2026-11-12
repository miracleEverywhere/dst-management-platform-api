package tools

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"strconv"
)

// TMICategoryList 获取 TMI 图片分类
func (s *Service) TMICategoryList() service.Result {
	categories, err := s.dstImageDao.Categories()
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeToolsDBError)
	}

	return service.Success(categories)
}

// TMICategoryItems 分页获取分类下的图片，缺失的图片从本地 TMI 资源转 base64 并回写缓存
func (s *Service) TMICategoryItems(category string, page, pageSize int) service.Result {
	images, err := s.dstImageDao.List(category, page, pageSize)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeToolsDBError)
	}

	var needParse []string

	for _, image := range images.Data {
		if image.Image == "" {
			needParse = append(needParse, image.Prefab)
		}
	}

	if len(needParse) > 0 {
		b64Map := pngsToBase64(needParse, page)
		for k, v := range b64Map {
			if v == "" {
				continue
			}
			for index, img := range images.Data {
				if img.Prefab == k {
					images.Data[index].Image = v
					err = s.dstImageDao.UpdateImage(&images.Data[index])
					if err != nil {
						logger.Logger.Errorf("更新图片失败: %v, 跳过", err)
					}
				}
			}
		}
	}

	return service.Success(images)
}

// Console 通过 TMI 控制台执行指令
func (s *Service) Console(actor service.Actor, roomID, worldID int, cmdType, uid, prefab string, num int) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeToolsNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeToolsDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	err = game.TmiConsoleCmd(cmdType, uid, prefab, num, worldID)
	if err != nil {
		return service.Fail(constant.CodeToolsExecFail)
	}

	return service.Msg("exec success", nil)
}
