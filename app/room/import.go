package room

import (
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ImportNewRoomFromZip 从已经保存到磁盘的 zip 导入一个新房间。
// savePath 是 zip 文件路径，unzipPath 是解压目录（需要以路径分隔符结尾）。
// 返回 room、业务响应 code、失败时的 i18n key 和原始错误。
func (h *Handler) ImportNewRoomFromZip(c *gin.Context, savePath, unzipPath string) (*models.Room, int, string, error) {
	return h.importRoomFromZip(c, savePath, unzipPath, 0, true)
}

func (h *Handler) importRoomFromZip(c *gin.Context, savePath, unzipPath string, roomID int, newRoom bool) (*models.Room, int, string, error) {

	var (
		room            models.Room
		worlds          []models.World
		roomSetting     models.RoomSetting
		uploadExtraInfo UploadExtraInfo
	)

	errMsg, err := handleUpload(savePath, unzipPath, &room, &worlds, &roomSetting, &uploadExtraInfo)
	if err != nil {
		logger.Logger.Errorf("处理上传文件失败, err: %v", err)
		return nil, 201, errMsg, err
	}

	if len(uploadExtraInfo.worldPath) == 0 {
		return nil, 201, "no available worlds found", fmt.Errorf("存档中没有可用世界")
	}

	// 设置所有的port和roomSetting
	if newRoom {
		room.Status = true
		roomCount, err := h.roomDao.Count(nil)
		if err != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", err)
			return nil, 500, "database error", err
		}

		worldCount, err := h.worldDao.Count(nil)
		if err != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", err)
			return nil, 500, "database error", err
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
		dbRoom, err := h.roomDao.GetRoomByID(roomID)
		if err != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", err)
			return nil, 500, "database error", err
		}
		room.MasterPort = dbRoom.MasterPort
		room.ID = roomID

		dbWorlds, err := h.worldDao.GetWorldsByRoomID(roomID)
		if err != nil {
			logger.Logger.Errorf("查询数据库失败, err: %v", err)
			return nil, 500, "database error", err
		}
		if len(worlds) != len(*dbWorlds) {
			return nil, 201, "number of worlds does not match", fmt.Errorf("上传存档世界个数与当前房间不一致")
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
		if _, err = h.roomDao.CreateRoom(&room); err != nil {
			logger.Logger.Errorf("创建房间失败, err: %v", err)
			return nil, 500, "database error", err
		}
		for index := range worlds {
			worlds[index].RoomID = room.ID
			if errCreateWorld := h.worldDao.Create(&worlds[index]); errCreateWorld != nil {
				logger.Logger.Errorf("创建房间失败, err: %v", errCreateWorld)
				return nil, 500, "database error", errCreateWorld
			}
		}

		roomSetting.RoomID = room.ID
		if errCreate := h.roomSettingDao.Create(&roomSetting); errCreate != nil {
			logger.Logger.Errorf("创建房间失败, err: %v", errCreate)
			return nil, 500, "database error", errCreate
		}
	} else {
		if err = h.roomDao.UpdateRoom(&room); err != nil {
			logger.Logger.Errorf("更新房间失败, err: %v", err)
			return nil, 500, "database error", err
		}
		if err = h.worldDao.UpdateWorlds(&worlds); err != nil {
			logger.Logger.Errorf("更新房间失败, err: %v", err)
			return nil, 500, "database error", err
		}
		// 不更新 roomSetting
	}

	game := dst.NewGameController(&room, &worlds, &roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	_ = game.StopAllWorld()
	if err = game.SaveAll(); err != nil {
		logger.Logger.Errorf("配置写入磁盘失败, err: %v", err)
		return nil, 201, "write file fail", err
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
		if c.GetString("role") != "admin" {
			if err = h.addRoomToUser(c.GetString("username"), strconv.Itoa(room.ID)); err != nil {
				logger.Logger.Errorf("更新用户房间权限失败, err: %v", err)
				return nil, 500, "database error", err
			}
		}

		webhook.Snd.Send(webhook.EventRoomCreated, room.ID, XRoomTotalInfo{
			RoomData:        room,
			WorldData:       worlds,
			RoomSettingData: roomSetting,
		})
	} else {
		if err = h.aiManager.Reload(room.ID); err != nil {
			logger.Logger.Errorf("重载房间 AI 对话失败, roomID: %d, err: %v", room.ID, err)
		}
	}

	return &room, 200, "", nil
}
