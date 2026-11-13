package room

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"fmt"
	"strconv"
)

// WorldPath 存档中世界的名称与解压目录
type WorldPath struct {
	name string
	path string
}

// UploadExtraInfo 存档导入过程中提取的附加信息
type UploadExtraInfo struct {
	adminlist string
	blocklist string
	whitelist string
	worldPath []WorldPath
}

// ImportNewRoomFromZip 从已经保存到磁盘的 zip 导入一个新房间。
// savePath 是 zip 文件路径，unzipPath 是解压目录（需要以路径分隔符结尾）。
// 返回 room、失败时的业务错误码（0 表示未指定）和原始错误。
func (s *Service) ImportNewRoomFromZip(actor service.Actor, savePath, unzipPath string) (*models.Room, int, error) {
	return s.importRoomFromZip(actor, savePath, unzipPath, 0, true)
}

func (s *Service) importRoomFromZip(actor service.Actor, savePath, unzipPath string, roomID int, newRoom bool) (*models.Room, int, error) {

	var (
		room            models.Room
		worlds          []models.World
		roomSetting     models.RoomSetting
		uploadExtraInfo UploadExtraInfo
	)

	code, err := handleUpload(savePath, unzipPath, &room, &worlds, &roomSetting, &uploadExtraInfo)
	if err != nil {
		logger.Logger.Errorf("处理上传文件失败, err: %v", err)
		return nil, code, err
	}

	if len(uploadExtraInfo.worldPath) == 0 {
		return nil, constant.CodeRoomImportNoWorlds, fmt.Errorf("存档中没有可用世界")
	}

	// 设置所有的port和roomSetting
	if newRoom {
		room.Status = true
		roomCount, errCount := s.roomDao.Count(nil)
		if errCount != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", errCount)
			return nil, constant.CodeRoomDBError, errCount
		}

		worldCount, errCount := s.worldDao.Count(nil)
		if errCount != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", errCount)
			return nil, constant.CodeRoomDBError, errCount
		}

		room.MasterPort = 21000 + int(roomCount) + 1
		for index := range worlds {
			worlds[index].ServerPort = 11000 + int(worldCount) + index + 1
			logger.Logger.Debugf("正在设置ServerPort, world.ServerPort: %v", worlds[index].ServerPort)
			worlds[index].MasterServerPort = 31000 + int(worldCount) + index + 1
			logger.Logger.Debugf("正在设置MasterServerPort, world.MasterServerPort: %v", worlds[index].MasterServerPort)
			worlds[index].AuthenticationPort = 41000 + int(worldCount) + index + 1
			logger.Logger.Debugf("正在设置AuthenticationPort, world.AuthenticationPort: %v", worlds[index].AuthenticationPort)
		}

		// 新房间的默认设置，与 roomPost 保持一致
		roomSetting.BackupEnable = true
		roomSetting.BackupSetting = "[{\"time\":\"06:00:00\"}]"
		roomSetting.BackupCleanEnable = false
		roomSetting.BackupCleanSetting = 30
		roomSetting.RestartEnable = false
		roomSetting.RestartSetting = "06:30:00"
		roomSetting.KeepaliveEnable = false
		roomSetting.KeepaliveSetting = 30
		roomSetting.ScheduledStartStopEnable = false
		roomSetting.ScheduledStartStopSetting = "{\"start\":\"07:00:00\",\"stop\":\"01:00:00\"}"
		roomSetting.AnnounceSetting = "[]"
		roomSetting.StartType = "32-bit"
	} else {
		dbRoom, errRoom := s.roomDao.GetRoomByID(roomID)
		if errRoom != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", errRoom)
			return nil, constant.CodeRoomDBError, errRoom
		}
		room.MasterPort = dbRoom.MasterPort
		room.ID = roomID
		// Status 是运行时状态，Lan/Offline/CustomGameMode 当前不由 handleUpload
		// 从存档解析；保留数据库中的值，避免 UpdateRoom 的 Save 用零值覆盖。
		room.Status = dbRoom.Status
		room.Lan = dbRoom.Lan
		room.Offline = dbRoom.Offline
		room.CustomGameMode = dbRoom.CustomGameMode

		dbWorlds, errWorlds := s.worldDao.GetWorldsByRoomID(roomID)
		if errWorlds != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", errWorlds)
			return nil, constant.CodeRoomDBError, errWorlds
		}
		if len(worlds) != len(*dbWorlds) {
			return nil, constant.CodeRoomImportWorldCountMismatch, fmt.Errorf("上传存档世界个数与当前房间不一致")
		}
		for index := range worlds {
			worlds[index].ServerPort = (*dbWorlds)[index].ServerPort
			worlds[index].MasterServerPort = (*dbWorlds)[index].MasterServerPort
			worlds[index].AuthenticationPort = (*dbWorlds)[index].AuthenticationPort
			worlds[index].RoomID = roomID
		}
	}

	// 判断是否为统一模组
	if len(worlds) >= 2 {
		if worlds[0].ModData == worlds[1].ModData {
			room.ModInOne = true
			room.ModData = worlds[0].ModData
			for index := range worlds {
				worlds[index].ModData = ""
			}
		} else {
			room.ModInOne = false
		}
	} else {
		room.ModInOne = true
	}

	// 写入数据库
	if newRoom {
		if _, err = s.roomDao.CreateRoom(&room); err != nil {
			logger.Logger.Errorf("创建房间失败, err: %v", err)
			return nil, constant.CodeRoomDBError, err
		}
		for index := range worlds {
			worlds[index].RoomID = room.ID
			if errCreateWorld := s.worldDao.Create(&worlds[index]); errCreateWorld != nil {
				logger.Logger.Errorf("创建房间失败, err: %v", errCreateWorld)
				return nil, constant.CodeRoomDBError, errCreateWorld
			}
		}

		roomSetting.RoomID = room.ID
		if errCreate := s.roomSettingDao.Create(&roomSetting); errCreate != nil {
			logger.Logger.Errorf("创建房间失败, err: %v", errCreate)
			return nil, constant.CodeRoomDBError, errCreate
		}
	} else {
		if err = s.roomDao.UpdateRoom(&room); err != nil {
			logger.Logger.Errorf("更新房间失败, err: %v", err)
			return nil, constant.CodeRoomDBError, err
		}
		if err = s.worldDao.UpdateWorlds(&worlds); err != nil {
			logger.Logger.Errorf("更新房间失败, err: %v", err)
			return nil, constant.CodeRoomDBError, err
		}
		// 不更新 roomSetting
	}

	game := dst.NewGameController(&room, &worlds, &roomSetting, actor.Lang)
	_ = game.StopAllWorld()
	if err = game.SaveAll(); err != nil {
		logger.Logger.Errorf("配置写入磁盘失败, err: %v", err)
		return nil, constant.CodeRoomWriteFileFail, err
	}

	clusterPath := fmt.Sprintf("%s/Cluster_%d", utils.ClusterPath, room.ID)

	// 设置三个名单
	if err = utils.TruncAndWriteFile(fmt.Sprintf("%s/adminlist.txt", clusterPath), uploadExtraInfo.adminlist); err != nil {
		logger.Logger.Errorf("设置管理员失败, err: %v", err)
	}
	if err = utils.TruncAndWriteFile(fmt.Sprintf("%s/blocklist.txt", clusterPath), uploadExtraInfo.blocklist); err != nil {
		logger.Logger.Errorf("设置黑名单失败, err: %v", err)
	}
	if err = utils.TruncAndWriteFile(fmt.Sprintf("%s/whitelist.txt", clusterPath), uploadExtraInfo.whitelist); err != nil {
		logger.Logger.Errorf("设置预留位失败, err: %v", err)
	}

	// 覆盖 save 目录
	for _, world := range uploadExtraInfo.worldPath {
		if err = utils.RemoveDir(fmt.Sprintf("%s/%s/save", clusterPath, world.name)); err != nil {
			logger.Logger.Errorf("删除旧存档数据失败, err: %v", err)
			continue
		}
		savePath := utils.ShellQuote(fmt.Sprintf("%s/save", world.path))
		worldPath := utils.ShellQuote(fmt.Sprintf("%s/%s/", clusterPath, world.name))
		cmd := fmt.Sprintf("cp -r %s %s", savePath, worldPath)
		logger.Logger.Debug(cmd)
		if err = utils.BashCMD(cmd); err != nil {
			logger.Logger.Errorf("复制存档数据失败, err: %v", err)
		}
	}

	if newRoom {
		// 新房间需要注册默认备份/重启等定时任务
		processJobs(game, room.ID, roomSetting)

		// 非管理员创建者需要把房间加入自己的房间权限列表，否则房间创建后不可见
		if !actor.IsAdmin() {
			if err = s.addRoomToUser(actor.Username, strconv.Itoa(room.ID)); err != nil {
				logger.Logger.Errorf("更新用户房间权限失败, err: %v", err)
				return nil, constant.CodeRoomDBError, err
			}
		}

		webhook.Snd.Send(webhook.EventRoomCreated, room.ID, XRoomTotalInfo{
			RoomData:        room,
			WorldData:       worlds,
			RoomSettingData: roomSetting,
		})
	} else {
		if err = s.aiManager.Reload(room.ID); err != nil {
			logger.Logger.Errorf("重载房间 AI 对话失败, roomID: %d, err: %v", room.ID, err)
		}
	}

	return &room, 0, nil
}

// UploadTarget 校验上传权限并解析目标房间ID；newRoom 为 true 表示新建房间
func (s *Service) UploadTarget(actor service.Actor, roomIDStr string) (roomID int, newRoom bool, failed *service.Result) {
	if roomIDStr == "" {
		// 新建房间，新建权限验证
		permission, _ := s.hasCreatePermission(actor)
		if !permission {
			r := service.Fail(constant.CodeRoomNoPermission)
			return 0, true, &r
		}
		return 0, true, nil
	}

	// 修改当前房间，修改权限验证
	if !s.hasRoomPermission(actor, roomIDStr) {
		r := service.Fail(constant.CodeRoomNoPermission)
		return 0, false, &r
	}

	id, err := strconv.Atoi(roomIDStr)
	if err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, actor.Path)
		r := service.Fail(constant.CodeRoomBadRequest)
		return 0, false, &r
	}

	return id, false, nil
}

// UploadImport 从已落盘的 zip 导入存档，newRoom 为 true 表示新建房间
func (s *Service) UploadImport(actor service.Actor, savePath, unzipPath string, roomID int, newRoom bool) service.Result {
	_, code, err := s.importRoomFromZip(actor, savePath, unzipPath, roomID, newRoom)
	if err != nil {
		if code == 0 {
			code = constant.CodeRoomUploadSaveFail
		}
		logger.Logger.Errorf("处理上传文件失败, err: %v", err)
		return service.Fail(code)
	}

	return service.Msg("upload success", nil)
}
