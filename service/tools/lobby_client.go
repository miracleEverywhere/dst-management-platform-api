package tools

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/logger"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"time"
)

// lobbyHTTPClient Klei 大厅请求使用的 HTTP 客户端，超时由 constant.KleiLobbyTimeout 控制
var lobbyHTTPClient = &http.Client{
	Timeout: constant.KleiLobbyTimeout * time.Second,
}

// lobbyRoomJSON 大厅列表与详情接口返回的房间结构（详情多出部分字段）
type lobbyRoomJSON struct {
	Addr            string                        `json:"__addr"`
	RowID           string                        `json:"__rowId"`
	Host            string                        `json:"host"`
	ClanOnly        bool                          `json:"clanonly"`
	Platform        int                           `json:"platform"`
	Mods            bool                          `json:"mods"`
	Name            string                        `json:"name"`
	PvP             bool                          `json:"pvp"`
	Session         string                        `json:"session"`
	Fo              bool                          `json:"fo"`
	Password        bool                          `json:"password"`
	GUID            string                        `json:"guid"`
	MaxConnections  int                           `json:"maxconnections"`
	Dedicated       bool                          `json:"dedicated"`
	ClientHosted    bool                          `json:"clienthosted"`
	Connected       int                           `json:"connected"`
	Mode            string                        `json:"mode"`
	Port            int                           `json:"port"`
	Version         int                           `json:"v"`
	Tags            string                        `json:"tags"`
	Season          string                        `json:"season"`
	LanOnly         bool                          `json:"lanonly"`
	Intent          string                        `json:"intent"`
	AllowNewPlayers bool                          `json:"allownewplayers"`
	ServerPaused    bool                          `json:"serverpaused"`
	SteamID         string                        `json:"steamid"`
	SteamRoom       string                        `json:"steamroom"`
	Secondaries     map[string]lobbySecondaryJSON `json:"secondaries"`

	// 详情接口新增字段
	Tick     int    `json:"tick"`
	Desc     string `json:"desc"`
	Data     string `json:"data"`
	Players  string `json:"players"`
	ModsInfo []any  `json:"mods_info"`
}

// lobbySecondaryJSON 附属世界的连接信息
type lobbySecondaryJSON struct {
	Addr    string `json:"__addr"`
	ID      string `json:"id"`
	SteamID string `json:"steamid"`
	Port    int    `json:"port"`
}

// lobbyListResponse 大厅列表接口响应
type lobbyListResponse struct {
	GET []lobbyRoomJSON `json:"GET"`
}

// lobbyDetailResponse 大厅详情接口响应
type lobbyDetailResponse struct {
	GET []lobbyRoomJSON `json:"GET"`
}

// fetchLobbyRoomsLimited 在全局并发上限内拉取区域列表：
// 列表接口单次下载量大（解压后 0.5~12MB），多区域同时下载会互相抢带宽，
// 这里用 Service 上的信号量限制同时进行的下载数量，后台刷新同样受限。
func (s *Service) fetchLobbyRoomsLimited(region string) ([]LobbyRoom, error) {
	s.lobbyFetchGuard <- struct{}{}
	defer func() { <-s.lobbyFetchGuard }()

	return s.fetchLobbyRooms(region)
}

// fetchLobbyRooms 拉取指定区域的大厅房间列表
func (s *Service) fetchLobbyRooms(region string) ([]LobbyRoom, error) {
	ctx, cancel := context.WithTimeout(context.Background(), constant.KleiLobbyTimeout*time.Second)
	defer cancel()

	url := fmt.Sprintf(constant.KleiLobbyListURL, region)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := lobbyHTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.Logger.Warnf("关闭大厅列表响应失败, err: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("lobby list api returned status %d", resp.StatusCode)
	}

	var payload lobbyListResponse
	if err = decodeLobbyBody(resp, &payload); err != nil {
		return nil, err
	}

	rooms := make([]LobbyRoom, 0, len(payload.GET))
	for _, item := range payload.GET {
		rooms = append(rooms, lobbyRoomFromJSON(item, region))
	}

	return rooms, nil
}

// fetchLobbyDetail 拉取指定房间的详情
func (s *Service) fetchLobbyDetail(region, rowID string) (lobbyRoomJSON, error) {
	ctx, cancel := context.WithTimeout(context.Background(), constant.KleiLobbyTimeout*time.Second)
	defer cancel()

	payload := map[string]any{
		"__token":  constant.KleiLobbyToken,
		"__gameId": constant.KleiLobbyGameID,
		"query": map[string]string{
			"__rowId": rowID,
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return lobbyRoomJSON{}, err
	}

	url := fmt.Sprintf(constant.KleiLobbyDetailURL, region)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return lobbyRoomJSON{}, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := lobbyHTTPClient.Do(req)
	if err != nil {
		return lobbyRoomJSON{}, err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			logger.Logger.Warnf("关闭大厅详情响应失败, err: %v", closeErr)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return lobbyRoomJSON{}, fmt.Errorf("lobby detail api returned status %d", resp.StatusCode)
	}

	var result lobbyDetailResponse
	if err = decodeLobbyBody(resp, &result); err != nil {
		return lobbyRoomJSON{}, err
	}

	if len(result.GET) == 0 {
		return lobbyRoomJSON{}, fmt.Errorf("lobby detail api returned empty result")
	}

	return result.GET[0], nil
}

// decodeLobbyBody 解析大厅响应体：CDN 返回 .json.gz，Go 通常已按 Content-Encoding 透明解压，
// 这里再按 gzip 魔数兜底，避免 CDN 响应头变化导致整页不可用。
func decodeLobbyBody(resp *http.Response, target any) error {
	reader := bufio.NewReaderSize(resp.Body, 64*1024)

	magic, err := reader.Peek(2)
	if err != nil && err != io.EOF {
		return err
	}
	if len(magic) == 2 && magic[0] == 0x1f && magic[1] == 0x8b {
		gzipReader, err := gzip.NewReader(reader)
		if err != nil {
			return err
		}
		defer func() {
			if closeErr := gzipReader.Close(); closeErr != nil {
				logger.Logger.Warnf("关闭大厅 gzip 流失败, err: %v", closeErr)
			}
		}()

		return json.NewDecoder(gzipReader).Decode(target)
	}

	return json.NewDecoder(reader).Decode(target)
}

// lobbyRoomFromJSON 转换大厅接口返回的房间结构
func lobbyRoomFromJSON(item lobbyRoomJSON, region string) LobbyRoom {
	room := LobbyRoom{
		RowID:           item.RowID,
		Region:          region,
		Name:            item.Name,
		Addr:            item.Addr,
		Port:            item.Port,
		Connected:       item.Connected,
		MaxConnections:  item.MaxConnections,
		Dedicated:       item.Dedicated,
		ClientHosted:    item.ClientHosted,
		Mods:            item.Mods,
		PvP:             item.PvP,
		Password:        item.Password,
		ClanOnly:        item.ClanOnly,
		AllowNewPlayers: item.AllowNewPlayers,
		ServerPaused:    item.ServerPaused,
		Season:          item.Season,
		Intent:          item.Intent,
		Version:         item.Version,
		Secondaries:     make([]LobbyShard, 0, len(item.Secondaries)),
		ConnectionCode:  lobbyConnectionCode(item.Addr, item.Port),
	}

	for _, secondary := range item.Secondaries {
		room.Secondaries = append(room.Secondaries, LobbyShard{
			ID:             secondary.ID,
			Addr:           secondary.Addr,
			Port:           secondary.Port,
			ConnectionCode: lobbyConnectionCode(secondary.Addr, secondary.Port),
		})
	}

	sort.SliceStable(room.Secondaries, func(i, j int) bool {
		left, leftErr := strconv.Atoi(room.Secondaries[i].ID)
		right, rightErr := strconv.Atoi(room.Secondaries[j].ID)
		if leftErr == nil && rightErr == nil && left != right {
			return left < right
		}
		return room.Secondaries[i].ID < room.Secondaries[j].ID
	})

	return room
}

// lobbyConnectionCode 生成游戏内使用的连接代码
func lobbyConnectionCode(addr string, port int) string {
	if addr == "" || port <= 0 {
		return ""
	}

	return fmt.Sprintf("c_connect(%s, %d)", strconv.Quote(addr), port)
}
