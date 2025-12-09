package dashboard

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Handler struct {
	userDao        *dao.UserDAO
	roomDao        *dao.RoomDAO
	worldDao       *dao.WorldDAO
	roomSettingDao *dao.RoomSettingDAO
}

func NewHandler(userDao *dao.UserDAO, roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO) *Handler {
	return &Handler{
		userDao:        userDao,
		roomDao:        roomDao,
		worldDao:       worldDao,
		roomSettingDao: roomSettingDao,
	}
}

func (h *Handler) fetchGameInfo(roomID int) (*models.Room, *[]models.World, *models.RoomSetting, error) {
	room, err := h.roomDao.GetRoomByID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}
	worlds, err := h.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}
	roomSetting, err := h.roomSettingDao.GetRoomSettingsByRoomID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}

	return room, worlds, roomSetting, nil
}

func (h *Handler) hasPermission(c *gin.Context, roomID string) bool {
	role, _ := c.Get("role")
	username, _ := c.Get("username")

	// 管理员直接返回true
	if role.(string) == "admin" {
		return true
	} else {
		dbUser, err := h.userDao.GetUserByUsername(username.(string))
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
	}

	return false
}

func cpuUsage() float64 {
	percent, err := cpu.Percent(0, false)
	if err != nil {
		return 0
	}
	return percent[0]
}

func memoryUsage() float64 {
	vmStat, err := mem.VirtualMemory()
	if err != nil {
		return 0
	}
	return vmStat.UsedPercent
}

func getInternetIP1() (string, error) {
	type JSONResponse struct {
		Status      string  `json:"status"`
		Country     string  `json:"country"`
		CountryCode string  `json:"countryCode"`
		Region      string  `json:"region"`
		RegionName  string  `json:"regionName"`
		City        string  `json:"city"`
		Zip         string  `json:"zip"`
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
		Timezone    string  `json:"timezone"`
		Isp         string  `json:"isp"`
		Org         string  `json:"org"`
		As          string  `json:"as"`
		Query       string  `json:"query"`
	}
	client := &http.Client{
		Timeout: 5 * time.Second, // 设置超时时间为 5 秒
	}
	httpResponse, err := client.Get(utils.InternetIPApi1)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Logger.Error("请求关闭失败", "err", err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体

	// 检查 HTTP 状态码
	if httpResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP 请求失败，状态码: %d", httpResponse.StatusCode)
	}
	var jsonResp JSONResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&jsonResp); err != nil {
		logger.Logger.Error("解析JSON失败", "err", err)
		return "", err
	}
	return jsonResp.Query, nil
}

func getInternetIP2() (string, error) {
	type JSONResponse struct {
		Ip string `json:"ip"`
	}
	client := &http.Client{
		Timeout: 10 * time.Second, // 设置超时时间为 10 秒
	}
	httpResponse, err := client.Get(utils.InternetIPApi2)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Logger.Error("请求关闭失败", "err", err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体

	// 检查 HTTP 状态码
	if httpResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP 请求失败，状态码: %d", httpResponse.StatusCode)
	}
	var jsonResp JSONResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&jsonResp); err != nil {
		logger.Logger.Error("解析JSON失败", "err", err)
		return "", err
	}
	return jsonResp.Ip, nil
}

func getDSTRoomsApi(region string) string {
	return fmt.Sprintf("https://lobby-v2-cdn.klei.com/%s-Steam.json.gz", region)
}

type Room struct {
	Name           string `json:"name"`
	MaxConnections int    `json:"maxconnections"`
}

type NeededResponse struct {
	GET []Room `json:"GET"`
}

func checkDstLobbyRoom(urls []string, clusterName string) ([]Room, error) {
	var (
		mu        sync.Mutex
		wg        sync.WaitGroup
		rooms     []Room
		errChanel = make(chan error, len(urls))
	)

	client := &http.Client{
		Timeout: utils.HttpTimeout * time.Second,
	}

	for _, url := range urls {
		wg.Add(1)
		go func(u string) {
			defer wg.Done()
			resp, err := client.Get(u)
			if err != nil {
				logger.Logger.Error("请求失败", "url", u, "err", err)
				errChanel <- err
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				logger.Logger.Warn("非200相应，跳过", "url", u)
				errChanel <- fmt.Errorf("非200响应")
				return
			}

			var neededResponse NeededResponse
			if err := json.NewDecoder(resp.Body).Decode(&neededResponse); err != nil {
				logger.Logger.Error("解析JSON失败", "err", err)
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
