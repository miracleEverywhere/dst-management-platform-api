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
	"encoding/json"
	"fmt"
	nethttp "net/http"
	"strconv"
	"strings"
	"sync"
	"time"
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

// CheckLobby 检查世界大厅信息
func (s *Service) CheckLobby(gameName string, maxPlayer int, regions []string) service.Result {
	if gameName == "" || maxPlayer == 0 {
		return service.Fail(constant.CodeDashboardBadRequest)
	}

	var urls []string
	for _, region := range regions {
		urls = append(urls, getDSTRoomsApi(region))
	}
	rooms, err := checkDstLobbyRoom(urls, gameName)
	if err != nil {
		return service.Result{Code: constant.CodeDashboardCheckLobbyFail, Data: false}
	}

	for _, room := range rooms {
		if room.MaxConnections == maxPlayer {
			return service.Success(true)
		}
	}

	return service.Success(false)
}

// Room 大厅接口返回的房间信息
type Room struct {
	Name           string `json:"name"`
	MaxConnections int    `json:"maxconnections"`
}

// NeededResponse 大厅接口返回结构
type NeededResponse struct {
	GET []Room `json:"GET"`
}

// getDSTRoomsApi 拼接指定区域的 DST 大厅接口地址
func getDSTRoomsApi(region string) string {
	return fmt.Sprintf("https://lobby-v2-cdn.klei.com/%s-Steam.json.gz", region)
}

// checkDstLobbyRoom 并发查询各区域大厅，返回匹配世界名的房间
func checkDstLobbyRoom(urls []string, clusterName string) ([]Room, error) {
	var (
		mu        sync.Mutex
		wg        sync.WaitGroup
		rooms     []Room
		errChanel = make(chan error, len(urls))
	)

	client := &nethttp.Client{
		Timeout: constant.HttpTimeout * time.Second,
	}

	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			resp, err := client.Get(u)
			if err != nil {
				logger.Logger.Errorf("请求失败: %v, url: %s", err, u)
				errChanel <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != nethttp.StatusOK {
				logger.Logger.Warnf("非200相应，跳过, url: %s", u)
				errChanel <- fmt.Errorf("非200响应")
				return
			}

			var neededResponse NeededResponse
			if err := json.NewDecoder(resp.Body).Decode(&neededResponse); err != nil {
				logger.Logger.Errorf("解析JSON失败: %v", err)
				errChanel <- err
				return
			}

			mu.Lock()
			for _, room := range neededResponse.GET {
				if room.Name == clusterName {
					rooms = append(rooms, room)
				}
			}
			mu.Unlock()
		}(url)
	}

	go func() {
		wg.Wait()
		close(errChanel)
	}()

	for err := range errChanel {
		if err != nil {
			return []Room{}, err
		}
	}

	return rooms, nil
}
