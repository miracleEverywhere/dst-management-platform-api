package scheduler

import (
	"bufio"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"sort"
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
			logger.Logger.Errorf("注册定时任务失败, err: %v", err)
			panic("注册定时任务失败")
		}
		logger.Logger.Infof("定时任务[%s]注册成功", job.Name)
	}
}

type DSTVersion struct {
	Local  int `json:"local"`
	Server int `json:"server"`
}

func GetDSTVersion() DSTVersion {
	var dstVersion DSTVersion
	dstVersion.Server = 0
	dstVersion.Local = 0

	file, err := os.Open(utils.DSTLocalVersionPath)
	if err != nil {
		logger.Logger.Errorf("获取游戏版本失败, err: %v", err)
		return dstVersion
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logger.Logger.Errorf("关闭文件失败, err: %v", err)
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
			logger.Logger.Errorf("获取游戏版本失败, err: %v", err)
			return dstVersion
		}
		dstVersion.Local = number

		// 获取服务端版本
		if db.GameServerVersion != 0 {
			dstVersion.Server = db.GameServerVersion
		} else {
			dstVersion.Server = getGameServerVersion()
		}

		return dstVersion
	}

	// 如果扫描器遇到错误，返回错误
	if err := scanner.Err(); err != nil {
		logger.Logger.Errorf("获取游戏版本失败, err: %v", err)

		return dstVersion
	}

	return dstVersion
}

func getGameServerVersion() int {
	// 发送 HTTP GET 请求
	client := &http.Client{
		Timeout: utils.HttpTimeout * time.Second,
	}
	response, err := client.Get(utils.DSTServerVersionApi)
	if err != nil {
		logger.Logger.Errorf("获取游戏版本失败, err: %v", err)
		return 0
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Logger.Errorf("关闭响应体失败, err: %v", err)
		}
	}(response.Body) // 确保在函数结束时关闭响应体

	// 检查 HTTP 状态码
	if response.StatusCode != http.StatusOK {
		logger.Logger.Errorf("获取游戏版本失败, statusCode: %d", response.StatusCode)
		return 0
	}

	// 读取响应体内容
	body, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Logger.Errorf("获取游戏版本失败, err: %v", err)
		return 0
	}

	// 找到所有带 data-currentRelease 的 <a> 标签，从 href URL 中提取帖子 ID
	// 例如 href='.../dst/728321-r2733/' 提取 728321
	tagRe := regexp.MustCompile(`<a[^>]*data-currentRelease[^>]*>`)
	tags := tagRe.FindAllString(string(body), -1)

	idRe := regexp.MustCompile(`/dst/(\d+)-`)
	var versions []int
	for _, tag := range tags {
		match := idRe.FindStringSubmatch(tag)
		if match != nil {
			if num, err := strconv.Atoi(match[1]); err == nil {
				versions = append(versions, num)
			}
		}
	}

	if len(versions) == 0 {
		logger.Logger.Errorf("获取游戏版本失败, 未从页面中提取到版本号")
		return 0
	}

	sort.Ints(versions)

	return versions[len(versions)-1]
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
			logger.Logger.Errorf("请求关闭失败, err: %v", err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体

	// 检查 HTTP 状态码
	if httpResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP 请求失败，状态码: %d", httpResponse.StatusCode)
	}
	var jsonResp JSONResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&jsonResp); err != nil {
		logger.Logger.Errorf("解析JSON失败, err: %v", err)
		return "", err
	}
	return jsonResp.Query, nil
}

func GetInternetIP2() (string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second, // 设置超时时间为 10 秒
	}
	response, err := client.Get(utils.InternetIPApi2)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Logger.Errorf("请求关闭失败, err: %v", err)
		}
	}(response.Body) // 确保在函数结束时关闭响应体

	// 检查 HTTP 状态码
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP 请求失败，状态码: %d", response.StatusCode)
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		logger.Logger.Errorf("读取响应失败, err: %v", err)
		return "", fmt.Errorf("读取响应失败")
	}

	re := regexp.MustCompile(`IP\s+:\s+(\d+\.\d+\.\d+\.\d+)`)
	matches := re.FindStringSubmatch(string(body))
	if len(matches) >= 2 {
		return matches[1], nil
	}

	return "", fmt.Errorf("查询公网ip失败")
}

// ParsePlayerInfoSaveTime 天转为秒
func ParsePlayerInfoSaveTime(saveTime int) int {
	if saveTime == 0 {
		saveTime = 1
	}
	return saveTime * 24 * 60 * 60
}
