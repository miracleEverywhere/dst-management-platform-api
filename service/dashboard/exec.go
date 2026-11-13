package dashboard

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"fmt"
	"strconv"
	"time"
)

// ExecGame 房间/世界操作入口，按操作类型分发
func (s *Service) ExecGame(actor service.Actor, execType string, roomID, worldID int, extra string) service.Result {
	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败：%v", err)
		return service.Fail(constant.CodeDashboardDBError)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeDashboardNoPermission)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)

	switch execType {
	case "startup":
		// 启动
		if extra == "all" {
			err = game.StartAllWorld()
			if err != nil {
				logger.Logger.Errorf("启动失败：%v", err)
				return service.Fail(constant.CodeDashboardStartupFail)
			}
		} else {
			err = game.StartWorld(worldID)
			if err != nil {
				logger.Logger.Errorf("启动失败: %v", err)
				return service.Fail(constant.CodeDashboardStartupFail)
			}
		}

		return service.Msg("startup game success", nil)
	case "shutdown":
		// 关闭
		if extra == "all" {
			err = game.StopAllWorld()
			if err != nil {
				logger.Logger.Errorf("关闭失败: %v", err)
				return service.Fail(constant.CodeDashboardShutdownFail)
			}
		} else {
			err = game.StopWorld(worldID)
			if err != nil {
				logger.Logger.Errorf("关闭失败: %v", err)
				return service.Fail(constant.CodeDashboardShutdownFail)
			}
		}

		return service.Msg("shutdown game success", nil)
	case "restart":
		// 重启
		_ = game.StopAllWorld()
		err = game.StartAllWorld()
		if err != nil {
			logger.Logger.Errorf("启动失败: %v", err)
			return service.Fail(constant.CodeDashboardRestartFail)
		}

		return service.Msg("restart game success", nil)
	case "update":
		// 更新，需要管理员权限
		if !actor.IsAdmin() {
			return service.Fail(constant.CodeDashboardNoPermission)
		}

		go func() {
			cache.DstUpdating = true
			updateCmd := fmt.Sprintf("cd ~/steamcmd && ./steamcmd.sh +login anonymous +force_install_dir ~/dst +app_update 343050 validate +quit")
			_ = utils.BashCMD(updateCmd)
			cache.DstUpdating = false

			// 如果需要重启，则重启激活的房间
			var globalSettings models.GlobalSetting
			err := s.globalSettingDao.GetGlobalSetting(&globalSettings)
			if err != nil {
				logger.Logger.Errorf("获取全局设置失败: %v", err)
				return
			}

			if !globalSettings.AutoUpdateRestart {
				return
			}

			roomBasic, err := s.roomDao.GetRoomBasic()
			if err != nil {
				logger.Logger.Errorf("获取全局房间信息失败: %v", err)
				return
			}
			for _, rb := range *roomBasic {
				if !rb.Status {
					continue
				}
				room, worlds, roomSetting, err := dao.FetchGameInfo(rb.RoomID)
				if err != nil {
					logger.Logger.Errorf("获取基本信息失败: %v", err)
					continue
				}
				game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
				_ = game.StopAllWorld()
				_ = game.StartAllWorld()
				time.Sleep(5 * time.Second)
			}
		}()

		return service.Msg("updating", nil)
	case "reset":
		if extra == "force" {
			err = game.Reset(true)
			if err != nil {
				return service.Fail(constant.CodeDashboardResetFail)
			}
		} else {
			err = game.Reset(false)
			if err != nil {
				return service.Fail(constant.CodeDashboardResetFail)
			}
		}

		return service.Msg("reset game success", nil)
	case "delete":
		defer func(game *dst.Game, id int) {
			// 清空世界中的存档数据后，启动该世界
			err := game.StartWorld(id)
			if err != nil {
				logger.Logger.Errorf("启动世界失败: %v", err)
			}
		}(game, worldID)

		// 先关闭世界
		err = game.StopWorld(worldID)
		if err != nil {
			logger.Logger.Warnf("关闭世界失败，可能是该世界已被关闭: %v", err)
		}

		err = game.DeleteWorld(worldID)
		if err != nil {
			return service.Fail(constant.CodeDashboardDeleteGameFail)
		}

		return service.Msg("delete game success", nil)
	case "announce":
		if extra == "" {
			return service.Fail(constant.CodeDashboardAnnounceFail)
		}
		err = game.Announce(extra)
		if err != nil {
			return service.Fail(constant.CodeDashboardAnnounceFail)
		}

		return service.Msg("announce success", nil)
	case "systemMsg":
		if extra == "" {
			return service.Fail(constant.CodeDashboardSystemMsgFail)
		}
		err = game.SystemMsg(extra)
		if err != nil {
			return service.Fail(constant.CodeDashboardSystemMsgFail)
		}

		return service.Msg("system msg success", nil)
	case "console":
		if extra == "" {
			return service.Fail(constant.CodeDashboardExecFail)
		}
		err = game.ConsoleCmd(extra, worldID)
		if err != nil {
			return service.Fail(constant.CodeDashboardExecFail)
		}

		return service.Msg("exec success", nil)
	default:
		return service.Fail(constant.CodeDashboardBadRequest)
	}
}
