package dashboard

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"fmt"
	"strconv"
	"strings"
)

// Service 面板模块业务实现，不依赖 gin
type Service struct {
	userDao          *dao.UserDAO
	roomDao          *dao.RoomDAO
	roomSettingDao   *dao.RoomSettingDAO
	globalSettingDao *dao.GlobalSettingDAO
}

// New 创建面板模块 service
func New(userDao *dao.UserDAO, roomDao *dao.RoomDAO, roomSettingDao *dao.RoomSettingDAO, globalSettingDao *dao.GlobalSettingDAO) *Service {
	return &Service{
		userDao:          userDao,
		roomDao:          roomDao,
		roomSettingDao:   roomSettingDao,
		globalSettingDao: globalSettingDao,
	}
}

// hasPermission 判断当前用户是否有房间权限
func (s *Service) hasPermission(actor service.Actor, roomID string) bool {
	// 管理员直接返回true
	if actor.IsAdmin() {
		return true
	}

	dbUser, err := s.userDao.GetUserByUsername(actor.Username)
	if err != nil {
		logger.Logger.Error("查询数据库失败")
		return false
	}
	roomIDs := strings.Split(dbUser.Rooms, ",")
	for _, id := range roomIDs {
		if id == roomID {
			return true
		}
	}

	return false
}

// GameWorldInfo 世界信息及运行状态
type GameWorldInfo struct {
	*models.World
	Status            bool                  `json:"status"`
	PerformanceStatus dst.PerformanceStatus `json:"performanceStatus"`
}

// InfoBaseData 面板基础信息响应 data
type InfoBaseData struct {
	Room        models.Room         `json:"room"`
	Worlds      []GameWorldInfo     `json:"worlds"`
	RoomSetting models.RoomSetting  `json:"roomSetting"`
	Session     dst.RoomSessionInfo `json:"session"`
	Players     []cache.PlayerInfo  `json:"players"`
}

// InfoBase 面板基础信息
func (s *Service) InfoBase(actor service.Actor, roomID int) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodeDashboardBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeDashboardNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败: %v", err)
		return service.Fail(constant.CodeDashboardDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)

	var gameWorldInfo []GameWorldInfo

	for _, world := range *worlds {
		gameWorldInfo = append(gameWorldInfo, GameWorldInfo{
			World:             &world,
			Status:            game.WorldUpStatus(world.ID),
			PerformanceStatus: game.WorldPerformanceStatus(world.ID),
		})
	}

	cache.PlayersStatisticMutex.Lock()
	defer cache.PlayersStatisticMutex.Unlock()

	var players []cache.PlayerInfo

	if len(cache.PlayersStatistic[roomID]) > 0 {
		players = cache.PlayersStatistic[roomID][len(cache.PlayersStatistic[roomID])-1].PlayerInfo
	} else {
		players = []cache.PlayerInfo{}
	}

	return service.Success(InfoBaseData{
		Room:        *room,
		Worlds:      gameWorldInfo,
		RoomSetting: *roomSetting,
		Session:     *game.SessionInfo(),
		Players:     players,
	})
}

// InfoSysData 系统信息响应 data
type InfoSysData struct {
	Cpu      float64 `json:"cpu"`
	Memory   float64 `json:"memory"`
	Updating bool    `json:"updating"`
}

// InfoSys 系统资源与更新状态
func (s *Service) InfoSys() service.Result {
	return service.Success(InfoSysData{
		Cpu:      utils.CpuUsage(),
		Memory:   utils.MemoryUsage(),
		Updating: cache.DstUpdating,
	})
}

// ConnectionCode 生成直连代码
func (s *Service) ConnectionCode(actor service.Actor, roomID int) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodeDashboardBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeDashboardNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败: %v", err)
		return service.Fail(constant.CodeDashboardDBError)
	}

	var connectionCode string

	if roomSetting.CustomIP == "" {
		// 返回默认直连代码
		var (
			internetIp string
			masterPort int
		)

		if cache.InternetIP == "" {
			internetIp, err = scheduler.GetInternetIP1()
			if err != nil {
				logger.Logger.Warnf("调用公网ip接口1失败: %v", err)
				internetIp, err = scheduler.GetInternetIP2()
				if err != nil {
					logger.Logger.Warnf("调用公网ip接口2失败: %v", err)
					return service.Fail(constant.CodeDashboardConnectionCodeFail)
				}
			}
			cache.InternetIP = internetIp
		} else {
			logger.Logger.Debug("发现缓存的公网IP")
			internetIp = cache.InternetIP
		}

		for _, world := range *worlds {
			if world.IsMaster {
				masterPort = world.ServerPort
			}
		}
		if masterPort == 0 {
			masterPort = (*worlds)[0].ServerPort
		}
		if room.Password == "" {
			connectionCode = fmt.Sprintf("c_connect('%s', %d)", internetIp, masterPort)
		} else {
			connectionCode = fmt.Sprintf("c_connect('%s', %d, '%s')", internetIp, masterPort, room.Password)
		}
	} else {
		// 返回自定义直连代码
		if room.Password == "" {
			connectionCode = fmt.Sprintf("c_connect('%s', %d)", roomSetting.CustomIP, roomSetting.CustomPort)
		} else {
			connectionCode = fmt.Sprintf("c_connect('%s', %d, '%s')", roomSetting.CustomIP, roomSetting.CustomPort, room.Password)
		}
	}

	return service.Success(connectionCode)
}

// ConnectionCodeUpdate 保存自定义直连地址
func (s *Service) ConnectionCodeUpdate(actor service.Actor, roomID int, ip string, port int) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeDashboardNoPermission)
	}

	roomSetting, err := s.roomSettingDao.GetRoomSettingsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败: %v", err)
		return service.Fail(constant.CodeDashboardDBError)
	}
	roomSetting.CustomIP = ip
	roomSetting.CustomPort = port

	err = s.roomSettingDao.UpdateRoomSetting(roomSetting)
	if err != nil {
		logger.Logger.Errorf("修改房间设置失败, err: %v", err)
		return service.Fail(constant.CodeDashboardUpdateFail)
	}

	return service.Msg("update success", nil)
}
