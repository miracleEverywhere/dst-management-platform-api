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
	"encoding/json"
	"fmt"
	"io"
	nethttp "net/http"
	"sort"
	"strconv"
	"strings"
	"time"
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

// Gantt 玩家在线时长甘特图
func (s *Service) Gantt(actor service.Actor, roomID, timeRange int) service.Result {
	if roomID <= 0 {
		return service.Fail(constant.CodePlayerBadRequest)
	}
	if timeRange == 0 {
		timeRange = 1
	}
	if timeRange < 1 || timeRange > 24*3650 {
		return service.Fail(constant.CodePlayerBadRequest)
	}
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodePlayerNoPermission)
	}

	var globalSetting models.GlobalSetting
	if err := s.globalSettingDao.GetGlobalSetting(&globalSetting); err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodePlayerDBError)
	}
	intervalSeconds := globalSetting.PlayerGetFrequency
	if intervalSeconds <= 0 {
		intervalSeconds = 60
	}

	now := time.Now()
	rangeStart := now.Add(-time.Duration(timeRange) * time.Hour).UnixMilli()
	rangeEnd := now.UnixMilli()
	intervalMillis := int64(intervalSeconds) * 1000

	type ganttItem struct {
		UID       string `json:"uid"`
		Nickname  string `json:"nickname"`
		Prefab    string `json:"prefab"`
		StartTime int64  `json:"startTime"`
		EndTime   int64  `json:"endTime"`
		Duration  int64  `json:"duration"`
	}

	itemsByPlayer := make(map[string][]ganttItem)
	cache.PlayersStatisticMutex.Lock()
	snapshots := cache.PlayersStatistic[roomID]
	for _, snapshot := range snapshots {
		if snapshot.Timestamp <= 0 || snapshot.Timestamp > rangeEnd {
			continue
		}
		start := snapshot.Timestamp
		if start < rangeStart {
			start = rangeStart
		}
		end := snapshot.Timestamp + intervalMillis
		if end > rangeEnd {
			end = rangeEnd
		}
		if start >= end {
			continue
		}

		for _, player := range snapshot.PlayerInfo {
			key := player.UID
			if key == "" {
				key = player.Nickname
			}
			if key == "" {
				continue
			}
			items := itemsByPlayer[key]
			item := ganttItem{
				UID:       player.UID,
				Nickname:  player.Nickname,
				Prefab:    player.Prefab,
				StartTime: start,
				EndTime:   end,
				Duration:  (end - start) / 1000,
			}
			if len(items) > 0 {
				last := &items[len(items)-1]
				if last.EndTime >= item.StartTime {
					if item.EndTime > last.EndTime {
						last.EndTime = item.EndTime
					}
					if item.Nickname != "" {
						last.Nickname = item.Nickname
					}
					if item.Prefab != "" {
						last.Prefab = item.Prefab
					}
					last.Duration = (last.EndTime - last.StartTime) / 1000
					itemsByPlayer[key] = items
					continue
				}
			}
			itemsByPlayer[key] = append(items, item)
		}
	}
	cache.PlayersStatisticMutex.Unlock()

	data := make([]ganttItem, 0)
	for _, items := range itemsByPlayer {
		data = append(data, items...)
	}
	sort.Slice(data, func(i, j int) bool {
		if data[i].StartTime != data[j].StartTime {
			return data[i].StartTime < data[j].StartTime
		}
		if data[i].Nickname != data[j].Nickname {
			return data[i].Nickname < data[j].Nickname
		}
		return data[i].UID < data[j].UID
	})

	return service.Success(data)
}

// List 获取房间的黑名单/白名单/管理员名单
func (s *Service) List(actor service.Actor, roomID int, listType string) service.Result {
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
	list := game.GetPlayerList(listType)

	return service.Success(list)
}

// UpdateList 修改房间的黑名单/白名单/管理员名单
func (s *Service) UpdateList(actor service.Actor, roomID int, uids []string, listType, actionType string) service.Result {
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

	switch actionType {
	case "add":
		err = game.AddPlayerList(uids, listType)
		if err != nil {
			logger.Logger.Infof("修改player list失败: %v", err)
			return service.Fail(constant.CodePlayerAddFail)
		}

		return service.Msg("add success", nil)
	case "delete":
		if len(uids) == 0 || uids[0] == "" {
			return service.Fail(constant.CodePlayerBadRequest)
		}
		err = game.RemovePlayerList(uids[0], listType)
		if err != nil {
			logger.Logger.Infof("修改player list失败: %v", err)
			return service.Fail(constant.CodePlayerDeleteFail)
		}

		return service.Msg("delete success", nil)
	case "deleteAll":
		err = game.RemovePlayerListAll(listType)
		if err != nil {
			logger.Logger.Infof("删除player list失败: %v", err)
			return service.Fail(constant.CodePlayerDeleteFail)
		}

		return service.Msg("delete success", nil)
	case "import":
		var importUids []string

		importUids, err = getPublicBlockList()
		if err != nil {
			logger.Logger.Error(err.Error())
		}

		err = game.AddPlayerList(importUids, listType)
		if err != nil {
			logger.Logger.Infof("修改player list失败: %v", err)
			return service.Fail(constant.CodePlayerAddFail)
		}

		return service.Msg("add success", nil)
	default:
		logger.Logger.Infof("请求参数错误, api: %s", actor.Path)
		return service.Fail(constant.CodePlayerBadRequest)
	}
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

// StatisticsOnlineTime 玩家在线时长饼图数据
func (s *Service) StatisticsOnlineTime(actor service.Actor, roomID int) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodePlayerBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodePlayerNoPermission)
	}

	cache.PlayersOnlineTimeMutex.Lock()
	defer cache.PlayersOnlineTimeMutex.Unlock()

	// 返回副本：锁在函数返回时释放，而响应的 JSON 编码发生在锁外，
	// 调度器会持锁原地写这个内层 map，直接返回共享引用会触发并发 map 读写。
	source := cache.PlayersOnlineTime[roomID]
	if len(source) == 0 {
		return service.Success(nil)
	}
	copied := make(map[string]int, len(source))
	for nickname, seconds := range source {
		copied[nickname] = seconds
	}

	return service.Success(copied)
}

// StatisticsPlayerCount 玩家数量折线图数据
func (s *Service) StatisticsPlayerCount(actor service.Actor, roomID, timeRange int) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodePlayerBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodePlayerNoPermission)
	}

	cache.PlayersStatisticMutex.Lock()
	defer cache.PlayersStatisticMutex.Unlock()

	var globalSettings models.GlobalSetting

	err := s.globalSettingDao.GetGlobalSetting(&globalSettings)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodePlayerDBError)
	}

	if globalSettings.PlayerGetFrequency <= 0 {
		return service.Fail(constant.CodePlayerDBError)
	}

	if timeRange <= 0 {
		timeRange = 24 * 60 * 60
	}

	dataCount := int(timeRange / globalSettings.PlayerGetFrequency) // 返回多少个数据
	statistics := cache.PlayersStatistic[roomID]
	if statistics == nil {
		return service.Success(nil)
	}
	if len(statistics) > dataCount {
		statistics = statistics[len(statistics)-dataCount:]
	}

	// 调度器会持锁对统计切片做 [1:] 和 append，锁释放后直接返回子切片会与
	// append 写入同一块内存；这里在锁内复制一份再返回。
	copied := make([]cache.Players, len(statistics))
	copy(copied, statistics)

	return service.Success(copied)
}

// StatisticsPlayerCountV2 玩家数量折线图数据（接口优化版）
func (s *Service) StatisticsPlayerCountV2(actor service.Actor, roomID, timeRange int) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodePlayerBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodePlayerNoPermission)
	}

	cache.PlayersStatisticMutex.Lock()
	defer cache.PlayersStatisticMutex.Unlock()

	var globalSettings models.GlobalSetting

	err := s.globalSettingDao.GetGlobalSetting(&globalSettings)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodePlayerDBError)
	}

	if timeRange == 0 {
		timeRange = 24 * 60 * 60
	}

	dataCount := int(timeRange / globalSettings.PlayerGetFrequency) // 返回多少个数据
	dataLength := len(cache.PlayersStatistic[roomID])               // 当前房间统计数据的个数

	var neededStatistics []cache.Players

	if dataLength > dataCount {
		neededStatistics = cache.PlayersStatistic[roomID][dataLength-dataCount:]
	} else {
		neededStatistics = cache.PlayersStatistic[roomID]
	}

	type dataItem struct {
		Count     int   `json:"count"`
		Timestamp int64 `json:"timestamp"`
	}
	var data []dataItem

	for _, players := range neededStatistics {
		data = append(data, dataItem{
			Count:     len(players.PlayerInfo),
			Timestamp: players.Timestamp,
		})
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

var client *nethttp.Client = &nethttp.Client{
	Timeout: constant.HttpTimeout * time.Second,
}

// getPublicBlockList 拉取公开的饥荒黑名单列表
func getPublicBlockList() ([]string, error) {
	httpResponse, err := client.Get(constant.DstBlockList)
	if err != nil {
		return []string{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Logger.Errorf("请求关闭失败, err: %v", err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体
	// 检查 HTTP 状态码
	if httpResponse.StatusCode != nethttp.StatusOK {
		return []string{}, fmt.Errorf("获取饥荒黑名单列表失败，HTTP Code: %d", httpResponse.StatusCode)
	}

	var responseData struct {
		Uids []string `json:"uids"`
	}
	if err = json.NewDecoder(httpResponse.Body).Decode(&responseData); err != nil {
		return []string{}, fmt.Errorf("解析饥荒黑名单失败, err: %v", err)
	}

	return responseData.Uids, nil
}
