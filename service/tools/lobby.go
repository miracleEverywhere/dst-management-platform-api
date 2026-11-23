package tools

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"
)

const (
	// lobbyDefaultPageSize 大厅列表默认每页条数
	lobbyDefaultPageSize = 10
	// lobbyMaxPageSize 大厅列表每页条数上限，避免一次返回过多数据
	lobbyMaxPageSize = 100
	// lobbyMaxKeywordRunes 搜索关键字最大字符数
	lobbyMaxKeywordRunes = 64
	// lobbyMaxRowIDLen 房间 __rowId 最大长度，防止异常超长参数
	lobbyMaxRowIDLen = 512
	// lobbyCacheTTL 区域房间列表缓存有效期
	lobbyCacheTTL = 60 * time.Second
	// lobbyMinRefreshInterval 两次强制刷新同一区域的最小间隔
	lobbyMinRefreshInterval = 10 * time.Second
	// lobbyFetchConcurrency 同时下载的区域数量上限：Klei CDN 单个区域解压后可达 12MB，
	// 多个区域一起下载会互相抢出口带宽（实测四区域并发时单区域耗时从 9 秒涨到 26 秒），
	// 限制并发反而更快也更稳。
	lobbyFetchConcurrency = 2
)

// lobbyRegions Klei 大厅支持的区域，顺序即前端区域选择的展示顺序
var lobbyRegions = []string{
	constant.KleiLobbyRegionAPEast1,
	constant.KleiLobbyRegionAPSoutheast1,
	constant.KleiLobbyRegionUSEast1,
	constant.KleiLobbyRegionEUCentral1,
}

// LobbySearchInput 大厅列表查询参数
type LobbySearchInput struct {
	Keyword  string   `json:"keyword"`
	Regions  []string `json:"regions"`
	Page     int      `json:"page"`
	PageSize int      `json:"pageSize"`
	Refresh  bool     `json:"refresh"`
}

// LobbyDetailInput 大厅房间详情查询参数
type LobbyDetailInput struct {
	RowID  string `json:"rowID"`
	Region string `json:"region"`
}

// LobbyShard 附属世界（洞穴等）的连接信息
type LobbyShard struct {
	ID             string `json:"id"`
	Addr           string `json:"addr"`
	Port           int    `json:"port"`
	ConnectionCode string `json:"connectionCode"`
}

// LobbyRoom 大厅列表中的单个房间
type LobbyRoom struct {
	RowID           string       `json:"rowID"`
	Region          string       `json:"region"`
	Name            string       `json:"name"`
	Addr            string       `json:"addr"`
	Port            int          `json:"port"`
	Connected       int          `json:"connected"`
	MaxConnections  int          `json:"maxConnections"`
	Dedicated       bool         `json:"dedicated"`
	ClientHosted    bool         `json:"clientHosted"`
	Mods            bool         `json:"mods"`
	PvP             bool         `json:"pvp"`
	Password        bool         `json:"password"`
	ClanOnly        bool         `json:"clanOnly"`
	AllowNewPlayers bool         `json:"allowNewPlayers"`
	ServerPaused    bool         `json:"serverPaused"`
	Season          string       `json:"season"`
	Intent          string       `json:"intent"`
	Version         int          `json:"version"`
	Secondaries     []LobbyShard `json:"secondaries"`
	ConnectionCode  string       `json:"connectionCode"`
}

// LobbyPlayer 房间内的在线玩家
type LobbyPlayer struct {
	Name   string `json:"name"`
	Prefab string `json:"prefab"`
	Colour string `json:"colour"`
}

// LobbyMod 房间启用的模组（来自大厅上报的 mods_info）
type LobbyMod struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Version1 string `json:"version1"`
	Version2 string `json:"version2"`
	Enabled  bool   `json:"enabled"`
}

// LobbyDetail 大厅房间详情
type LobbyDetail struct {
	LobbyRoom
	Desc               string        `json:"desc"`
	Tick               int           `json:"tick"`
	Days               int           `json:"days"`
	DayElapsedInSeason int           `json:"dayElapsedInSeason"`
	DaysLeftInSeason   int           `json:"daysLeftInSeason"`
	Players            []LobbyPlayer `json:"players"`
	Mods               []LobbyMod    `json:"mods"`
	ModIDs             []int         `json:"modIDs"`
	PlayersParsed      bool          `json:"playersParsed"`
}

// LobbySearchData 大厅列表分页结果
type LobbySearchData struct {
	Total         int         `json:"total"`
	Page          int         `json:"page"`
	PageSize      int         `json:"pageSize"`
	Regions       []string    `json:"regions"`
	FailedRegions []string    `json:"failedRegions"`
	UpdatedAt     int64       `json:"updatedAt"`
	Rows          []LobbyRoom `json:"rows"`
}

// LobbySearch 分页查询游戏大厅房间
func (s *Service) LobbySearch(input LobbySearchInput) service.Result {
	keyword := strings.TrimSpace(input.Keyword)
	if utf8.RuneCountInString(keyword) > lobbyMaxKeywordRunes {
		return service.Fail(constant.CodeToolsBadRequest)
	}

	regions, ok := normalizeLobbyRegions(input.Regions)
	if !ok {
		return service.Fail(constant.CodeToolsBadRequest)
	}

	page, pageSize := normalizeLobbyPage(input.Page, input.PageSize)

	type regionResult struct {
		region    string
		rooms     []LobbyRoom
		updatedAt time.Time
		err       error
	}

	results := make([]regionResult, len(regions))
	var wg sync.WaitGroup
	for index, region := range regions {
		wg.Add(1)
		go func(index int, region string) {
			defer wg.Done()

			rooms, fetchedAt, err := s.lobbyCache.rooms(region, input.Refresh, s.fetchLobbyRoomsLimited)
			results[index] = regionResult{
				region:    region,
				rooms:     rooms,
				updatedAt: fetchedAt,
				err:       err,
			}
		}(index, region)
	}
	wg.Wait()

	var (
		rooms         []LobbyRoom
		failedRegions = make([]string, 0)
		updatedAt     time.Time
		successCount  int
	)
	for _, result := range results {
		if result.err != nil {
			logger.Logger.Warnf("获取游戏大厅房间列表失败, region: %s, err: %v", result.region, result.err)
			failedRegions = append(failedRegions, result.region)
		} else {
			successCount++
		}
		if len(result.rooms) > 0 {
			rooms = append(rooms, result.rooms...)
		}
		if !result.updatedAt.IsZero() && (updatedAt.IsZero() || result.updatedAt.Before(updatedAt)) {
			updatedAt = result.updatedAt
		}
	}

	// 所有区域都失败且没有任何缓存数据时才整体报错
	if successCount == 0 && len(rooms) == 0 {
		return service.Fail(constant.CodeToolsLobbyListFail)
	}

	if keyword != "" {
		matcher := strings.ToLower(keyword)
		filtered := make([]LobbyRoom, 0, len(rooms))
		for _, room := range rooms {
			if strings.Contains(strings.ToLower(room.Name), matcher) {
				filtered = append(filtered, room)
			}
		}
		rooms = filtered
	}

	sortLobbyRooms(rooms)

	total := len(rooms)
	if total == 0 {
		page = 1
	}
	start := (page - 1) * pageSize
	if total > 0 && start >= total {
		page = (total + pageSize - 1) / pageSize
		start = (page - 1) * pageSize
	}
	if start < 0 || start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}

	pageRows := make([]LobbyRoom, 0, end-start)
	if start < total {
		pageRows = append(pageRows, rooms[start:end]...)
	}

	data := LobbySearchData{
		Total:         total,
		Page:          page,
		PageSize:      pageSize,
		Regions:       append([]string(nil), lobbyRegions...),
		FailedRegions: failedRegions,
		Rows:          pageRows,
	}
	if !updatedAt.IsZero() {
		data.UpdatedAt = updatedAt.Unix()
	}

	return service.Success(data)
}

// LobbyDetail 查询游戏大厅房间详情
func (s *Service) LobbyDetail(input LobbyDetailInput) service.Result {
	rowID := strings.TrimSpace(input.RowID)
	if rowID == "" || len(rowID) > lobbyMaxRowIDLen {
		return service.Fail(constant.CodeToolsBadRequest)
	}

	region, ok := normalizeLobbyRegion(input.Region)
	if !ok {
		return service.Fail(constant.CodeToolsBadRequest)
	}

	room, err := s.fetchLobbyDetail(region, rowID)
	if err != nil {
		logger.Logger.Warnf("获取游戏大厅房间详情失败, region: %s, rowID: %s, err: %v", region, rowID, err)
		return service.Fail(constant.CodeToolsLobbyDetailFail)
	}

	detail := LobbyDetail{
		LobbyRoom: lobbyRoomFromJSON(room, region),
		Desc:      room.Desc,
		Tick:      room.Tick,
		Players:   []LobbyPlayer{},
		Mods:      []LobbyMod{},
		ModIDs:    []int{},
	}

	detail.Days, detail.DayElapsedInSeason, detail.DaysLeftInSeason = parseLobbyDays(room.Data)

	players, playersOK := parseLobbyPlayers(room.Players)
	if !playersOK {
		logger.Logger.Warnf("解析游戏大厅玩家列表失败, region: %s, rowID: %s", region, rowID)
		players = []LobbyPlayer{}
	}
	detail.Players = players
	detail.PlayersParsed = playersOK

	mods, modIDs := parseLobbyMods(room.ModsInfo)
	detail.Mods = mods
	detail.ModIDs = modIDs

	return service.Success(detail)
}

// normalizeLobbyRegions 校验并去重区域，空列表表示查询全部区域
func normalizeLobbyRegions(regions []string) ([]string, bool) {
	if len(regions) == 0 {
		return append([]string(nil), lobbyRegions...), true
	}

	result := make([]string, 0, len(regions))
	seen := make(map[string]struct{}, len(regions))
	for _, region := range regions {
		region = strings.TrimSpace(region)
		if _, ok := normalizeLobbyRegion(region); !ok {
			return nil, false
		}
		if _, ok := seen[region]; ok {
			continue
		}
		seen[region] = struct{}{}
		result = append(result, region)
	}

	if len(result) == 0 {
		return nil, false
	}

	return result, true
}

// normalizeLobbyRegion 校验单个区域，区域会拼进请求 URL，必须命中白名单
func normalizeLobbyRegion(region string) (string, bool) {
	region = strings.TrimSpace(region)
	for _, supported := range lobbyRegions {
		if region == supported {
			return region, true
		}
	}

	return "", false
}

// normalizeLobbyPage 归一化分页参数
func normalizeLobbyPage(page, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = lobbyDefaultPageSize
	}
	if pageSize > lobbyMaxPageSize {
		pageSize = lobbyMaxPageSize
	}

	return page, pageSize
}

// sortLobbyRooms 按在线人数降序、房间名升序、rowID 升序排序，保证翻页稳定
func sortLobbyRooms(rooms []LobbyRoom) {
	sort.SliceStable(rooms, func(i, j int) bool {
		if rooms[i].Connected != rooms[j].Connected {
			return rooms[i].Connected > rooms[j].Connected
		}
		left := strings.ToLower(rooms[i].Name)
		right := strings.ToLower(rooms[j].Name)
		if left != right {
			return left < right
		}
		return rooms[i].RowID < rooms[j].RowID
	})
}
