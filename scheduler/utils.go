package scheduler

import (
	"bufio"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/go-co-op/gocron"
)

var (
	Scheduler   = gocron.NewScheduler(time.Local)
	jobMutex    sync.RWMutex
	currentJobs = make(map[string]*gocron.Job)
	DBHandler   *Handler
)

type JobConfig struct {
	Name     string
	Func     any
	Args     []any
	TimeType string
	Interval int
	DayAt    string
}

const (
	SecondType = "second"
	MinuteType = "minute"
	HourType   = "hour"
	DayType    = "day"
)

type Handler struct {
	roomDao          *dao.RoomDAO
	worldDao         *dao.WorldDAO
	roomSettingDao   *dao.RoomSettingDAO
	globalSettingDao *dao.GlobalSettingDAO
	uidMapDao        *dao.UidMapDAO
}

func newDBHandler(roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO, globalSettingDao *dao.GlobalSettingDAO, uidMapDao *dao.UidMapDAO) *Handler {
	return &Handler{
		roomDao:          roomDao,
		worldDao:         worldDao,
		roomSettingDao:   roomSettingDao,
		globalSettingDao: globalSettingDao,
		uidMapDao:        uidMapDao,
	}
}

func registerJobs() {
	for _, job := range Jobs {
		err := UpdateJob(&job)
		if err != nil {
			logger.Logger.Error("注册定时任务失败", "err", err)
			panic("注册定时任务失败")
		}
		logger.Logger.Info(fmt.Sprintf("定时任务[%s]注册成功", job.Name))
	}
}

func fetchGameInfo(roomID int) (*models.Room, *[]models.World, *models.RoomSetting, error) {
	room, err := DBHandler.roomDao.GetRoomByID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}
	worlds, err := DBHandler.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}
	roomSetting, err := DBHandler.roomSettingDao.GetRoomSettingsByRoomID(roomID)
	if err != nil {
		return &models.Room{}, &[]models.World{}, &models.RoomSetting{}, err
	}

	return room, worlds, roomSetting, nil
}

type DSTVersion struct {
	Local  int `json:"local"`
	Server int `json:"server"`
}

func GetDSTVersion() DSTVersion {
	var dstVersion DSTVersion
	dstVersion.Server = 0
	dstVersion.Local = 0

	client := &http.Client{
		Timeout: utils.HttpTimeout * time.Second,
	}

	file, err := os.Open(utils.DSTLocalVersionPath)
	if err != nil {
		logger.Logger.Error("获取游戏版本失败", "err", err)
		return dstVersion
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logger.Logger.Error("关闭文件失败", "err", err)
		}
	}(file) // 确保文件在函数结束时关闭

	// 创建一个扫描器来读取文件内容
	scanner := bufio.NewScanner(file)

	// 扫描文件的第一行
	if scanner.Scan() {
		// 读取第一行的文本
		line := scanner.Text()

		// 将字符串转换为整数
		number, err := strconv.Atoi(line)
		if err != nil {
			logger.Logger.Error("获取游戏版本失败", "err", err)
			return dstVersion
		}
		dstVersion.Local = number
		// 获取服务端版本
		// 发送 HTTP GET 请求
		response, err := client.Get(utils.DSTServerVersionApi)
		if err != nil {
			logger.Logger.Error("获取游戏版本失败", "err", err)
			return dstVersion
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				logger.Logger.Error("关闭文件失败", "err", err)
			}
		}(response.Body) // 确保在函数结束时关闭响应体

		// 检查 HTTP 状态码
		if response.StatusCode != http.StatusOK {
			logger.Logger.Error("获取游戏版本失败", "err", err)
			return dstVersion
		}

		// 读取响应体内容
		body, err := io.ReadAll(response.Body)
		if err != nil {
			logger.Logger.Error("获取游戏版本失败", "err", err)
			return dstVersion
		}

		// 将字节数组转换为字符串并返回
		serverVersion, err := strconv.Atoi(string(body))
		if err != nil {
			logger.Logger.Error("获取游戏版本失败", "err", err)
			return dstVersion
		}

		dstVersion.Server = serverVersion

		return dstVersion
	}

	// 如果扫描器遇到错误，返回错误
	if err := scanner.Err(); err != nil {
		dstVersion.Server = 0
		dstVersion.Local = 0
		logger.Logger.Error("获取游戏版本失败", "err", err)

		return dstVersion
	}

	// 如果文件为空，返回错误
	dstVersion.Server = 0
	dstVersion.Local = 0

	return dstVersion
}

type AnnounceSetting struct {
	ID       string `json:"id"`
	Status   bool   `json:"status"`
	Interval int    `json:"interval"`
	Content  string `json:"content"`
}

func GetInternetIP1() (string, error) {
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

func GetInternetIP2() (string, error) {
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
