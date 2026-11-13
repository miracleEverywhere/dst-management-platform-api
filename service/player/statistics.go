package player

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"sort"
	"strconv"
	"time"
)

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
