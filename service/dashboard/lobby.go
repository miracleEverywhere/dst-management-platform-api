package dashboard

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"encoding/json"
	"fmt"
	nethttp "net/http"
	"sync"
	"time"
)

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
