// Package player 玩家模块业务实现，只依赖 service.Actor 等公共契约，不依赖 gin。
package player

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"strconv"
	"strings"
)

// PaginatedUidMap uidmap 分页查询的响应结构，沿用原有 JSON 字段。
// handler 在参数绑定失败时也用它构造 data，保持原有响应结构。
type PaginatedUidMap = dao.PaginatedResult[models.UidMap]

// Service 玩家模块业务实现，不依赖 gin
type Service struct {
	userDao          *dao.UserDAO
	uidMapDao        *dao.UidMapDAO
	globalSettingDao *dao.GlobalSettingDAO
}

// New 创建玩家模块 service
func New(userDao *dao.UserDAO, uidMapDao *dao.UidMapDAO, globalSettingDao *dao.GlobalSettingDAO) *Service {
	return &Service{
		userDao:          userDao,
		uidMapDao:        uidMapDao,
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

// Online 当前在线玩家
func (s *Service) Online(actor service.Actor, roomID int) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodePlayerBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodePlayerNoPermission)
	}

	cache.PlayersStatisticMutex.Lock()
	defer cache.PlayersStatisticMutex.Unlock()

	var players []cache.PlayerInfo

	if len(cache.PlayersStatistic[roomID]) > 0 {
		players = cache.PlayersStatistic[roomID][len(cache.PlayersStatistic[roomID])-1].PlayerInfo
	} else {
		players = []cache.PlayerInfo{}
	}

	return service.Success(players)
}

// UIDMap uidmap 分页查询
func (s *Service) UIDMap(actor service.Actor, roomID int, q string, page, pageSize int) service.Result {
	if roomID == 0 {
		return service.Result{Code: constant.CodePlayerBadRequest, Data: PaginatedUidMap{}}
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Result{Code: constant.CodePlayerNoPermission, Data: PaginatedUidMap{}}
	}

	uidMap, err := s.uidMapDao.ListUidMaps(roomID, q, page, pageSize)
	if err != nil {
		logger.Logger.Errorf("获取uidmap失败, err: %v", err)
		return service.Result{Code: constant.CodePlayerDBError, Data: PaginatedUidMap{}}
	}

	data := *uidMap
	if data.Data == nil {
		data.Data = []models.UidMap{}
	}

	return service.Success(data)
}

// Chat 玩家聊天列表
func (s *Service) Chat(actor service.Actor, roomID, lines int, needTime bool) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodePlayerBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodePlayerNoPermission)
	}
	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodePlayerDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	chat, err := game.ChatMessages(lines, needTime)
	if err != nil {
		logger.Logger.Errorf("获取玩家聊天信息失败：%s", err.Error())
		return service.Fail(constant.CodePlayerChatMessageFail)
	}

	return service.Success(chat)
}
