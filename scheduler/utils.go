package scheduler

import (
	"bufio"
	"crypto/tls"
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

var client = &http.Client{
	Timeout: utils.HttpTimeout * time.Second,
}

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

	dstVersion.Local = getLocalGameVersion()

	if db.GameServerVersion != 0 {
		dstVersion.Server = db.GameServerVersion
	} else {
		var err error
		dstVersion.Server, err = getServerGameVersion()
		if err != nil {
			logger.Logger.Error(err)
		}
	}

	return dstVersion
}

func getLocalGameVersion() int {
	version := 0
	file, err := os.Open(utils.DSTLocalVersionPath)
	if err != nil {
		logger.Logger.Errorf("获取游戏版本失败, err: %v", err)
		return version
	}
	defer func(file *os.File) {
		_ = file.Close()
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

			return version
		}
		version = number

		return version
	}

	// 如果扫描器遇到错误，返回错误
	if err = scanner.Err(); err != nil {
		logger.Logger.Errorf("获取游戏版本失败, err: %v", err)

		return version
	}

	return version
}

func getServerGameVersion() (int, error) {
	var (
		version int
		err     error
	)

	version, err = getServerGameVersionFromKlei()
	logger.Logger.Debug("尝试从饥荒论坛中获取游戏版本")
	if err != nil {
		logger.Logger.Warnf("从饥荒论坛中获取游戏版本失败: %v, 尝试从api获取", err)
	} else {
		logger.Logger.Debug("从饥荒论坛中获取游戏版本成功")
		return version, nil
	}

	apis := []string{
		utils.DSTServerVersionApi1,
		utils.DSTServerVersionApi2,
	}
	for _, api := range apis {
		logger.Logger.Debugf("尝试从api: %s 获取游戏版本", api)
		version, err = getServerGameVersionFromDstVersion(api)
		if err != nil {
			logger.Logger.Warnf("从api中获取游戏版本失败: %v, 尝试下一个api", err)
		} else {
			logger.Logger.Debugf("从api: %s 获取游戏版本成功", api)
			return version, nil
		}
	}

	return 0, fmt.Errorf("获取游戏版本失败，%d种方法均失败", 1+len(apis))
}

func getServerGameVersionFromKlei() (int, error) {
	const (
		versionPageURL     = utils.DSTServerVersionKlei
		maxVersionPageSize = 8 << 20
		userAgent          = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	)
	client := &http.Client{
		Transport: &http.Transport{
			Proxy:             http.ProxyFromEnvironment,
			ForceAttemptHTTP2: false,
			TLSNextProto:      make(map[string]func(string, *tls.Conn) http.RoundTripper),
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				NextProtos: []string{"http/1.1"},
			},
		},
		Timeout: utils.HttpTimeout * time.Second,
	}
	versionPageURLs := []string{
		versionPageURL,
		versionPageURL + "?rss=1",
		versionPageURL + "?sortby=newest&sortdirection=desc",
	}
	currentReleaseTagRE := regexp.MustCompile(`(?is)<a\b[^>]*\bdata-currentrelease(?:\s|=)[^>]*>`)
	versionLinkRE := regexp.MustCompile(`(?i)/dst/([0-9]+)-`)
	var lastErr error

	for _, url := range versionPageURLs {
		for attempt := 0; attempt < 2; attempt++ {
			request, err := http.NewRequest(http.MethodGet, url, nil)
			if err != nil {
				return 0, err
			}
			request.Header.Set("User-Agent", userAgent)
			request.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
			request.Header.Set("Accept-Language", "en-US,en;q=0.9")
			request.Header.Set("Cache-Control", "no-cache")
			request.Header.Set("Pragma", "no-cache")
			request.Header.Set("Sec-Fetch-Dest", "document")
			request.Header.Set("Sec-Fetch-Mode", "navigate")
			request.Header.Set("Sec-Fetch-Site", "none")
			request.Header.Set("Sec-Fetch-User", "?1")
			request.Header.Set("Upgrade-Insecure-Requests", "1")

			response, err := client.Do(request)
			if err != nil {
				lastErr = err
				if attempt == 0 {
					time.Sleep(250 * time.Millisecond)
					continue
				}
				break
			}

			body, readErr := io.ReadAll(io.LimitReader(response.Body, maxVersionPageSize+1))
			response.Body.Close()
			if readErr != nil {
				lastErr = readErr
				break
			}
			if len(body) > maxVersionPageSize {
				lastErr = fmt.Errorf("version page exceeds %d bytes", maxVersionPageSize)
				break
			}
			if response.StatusCode < 200 || response.StatusCode >= 300 {
				if response.StatusCode == http.StatusForbidden {
					lastErr = fmt.Errorf("Klei returned HTTP 403 (Cloudflare or site access rule)")
				} else {
					lastErr = fmt.Errorf("HTTP statusCode: %d", response.StatusCode)
				}
				if attempt == 0 && (response.StatusCode == http.StatusTooManyRequests || response.StatusCode >= http.StatusInternalServerError) {
					time.Sleep(500 * time.Millisecond)
					continue
				}
				break
			}

			html := string(body)
			tags := currentReleaseTagRE.FindAllString(html, -1)
			if len(tags) == 0 {
				tags = []string{html}
			}

			latestVersion := 0
			for _, tag := range tags {
				matches := versionLinkRE.FindAllStringSubmatch(tag, -1)
				for _, match := range matches {
					version, convertErr := strconv.Atoi(match[1])
					if convertErr == nil && version > latestVersion {
						latestVersion = version
					}
				}
			}
			if latestVersion > 0 {
				return latestVersion, nil
			}

			lastErr = fmt.Errorf("no version number found on the Klei version page")
			break
		}
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no Klei version page is available")
	}
	return 0, fmt.Errorf("failed to get game version: %w", lastErr)
}

func getServerGameVersionFromDstVersion(api string) (int, error) {
	response, err := client.Get(api)
	if err != nil {
		return 0, err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(response.Body) // 确保在函数结束时关闭响应体

	// 检查 HTTP 状态码
	if response.StatusCode != http.StatusOK {
		err = fmt.Errorf("HTTP statusCode: %d", response.StatusCode)
		return 0, err
	}

	// 读取响应体内容
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return 0, err
	}

	// 将字节数组转换为字符串并返回
	number, err := strconv.Atoi(string(body))
	if err != nil {
		return 0, err
	}

	return number, nil
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
