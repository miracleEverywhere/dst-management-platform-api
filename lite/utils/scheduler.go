package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// gameVersionInterval 游戏版本拉取间隔
	gameVersionInterval = 2 * time.Hour
	// gameVersionMaxBody 版本接口响应体上限，避免异常响应把内存吃满
	gameVersionMaxBody = 64
	// internetIPInterval 公网IP拉取间隔
	internetIPInterval = 6 * time.Hour
	// internetIPMaxBody 公网IP接口响应体上限，避免异常响应把内存吃满
	internetIPMaxBody = 64 << 10
	// internetIPApi1 公网IP查询接口1，返回 JSON
	internetIPApi1 = "http://ip-api.com/json/?lang=zh-CN"
	// internetIPApi2 公网IP查询接口2，返回纯文本
	internetIPApi2 = "http://cip.cc"
)

var (
	// GameVersionServer 服务端最新游戏版本号，由后台任务定时刷新。
	// 会被 HTTP handler 并发读取，必须用 atomic 访问。
	GameVersionServer atomic.Int64

	// internetIP 后台任务最近一次成功获取到的公网 IP。
	// 会被 HTTP handler 并发读取，必须持锁访问。
	internetIPMu sync.RWMutex
	internetIP   string

	// internetIPPattern 从接口2的纯文本响应中提取 IPv4 地址
	internetIPPattern = regexp.MustCompile(`IP\s+:\s+(\d+\.\d+\.\d+\.\d+)`)

	client = &http.Client{Timeout: 30 * time.Second}
)

// GetInternetIP 返回后台任务最近一次成功获取到的公网 IP，尚未获取成功时返回空字符串。
func GetInternetIP() string {
	internetIPMu.RLock()
	defer internetIPMu.RUnlock()

	return internetIP
}

func setInternetIP(ip string) {
	internetIPMu.Lock()
	defer internetIPMu.Unlock()

	internetIP = ip
}

// Start 启动后台任务：游戏版本与公网IP在启动时立即拉取一次，之后按各自的间隔刷新。
// 返回的 stop 用于停止任务（停止 ticker 并退出循环），可重复调用。
func Start() (stop func()) {
	stopCh := make(chan struct{})
	var once sync.Once
	stop = func() {
		once.Do(func() { close(stopCh) })
	}

	go runPeriodic(stopCh, gameVersionInterval, gameVersionServerGet)
	go runPeriodic(stopCh, internetIPInterval, internetIPGet)

	return stop
}

// runPeriodic 立即执行一次 fn，之后每隔 interval 执行一次，stopCh 关闭后退出。
func runPeriodic(stopCh <-chan struct{}, interval time.Duration, fn func()) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	fn()
	for {
		select {
		case <-stopCh:
			return
		case <-ticker.C:
			fn()
		}
	}
}

func gameVersionServerGet() {
	Logger.Info("正在获取游戏版本中")
	res, err := client.Get("http://ver.tugos.cn/getLocalVersion")
	if err != nil {
		Logger.Error(fmt.Sprintf("获取饥荒版本失败, err: %v", err))
		return
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	if res.StatusCode != http.StatusOK {
		Logger.Error(fmt.Sprintf("获取饥荒版本失败, HTTP statusCode: %d", res.StatusCode))
		return
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, gameVersionMaxBody))
	if err != nil {
		Logger.Error(fmt.Sprintf("获取饥荒版本失败, err: %v", err))
		return
	}

	number, err := strconv.Atoi(string(body))
	if err != nil {
		Logger.Error(fmt.Sprintf("获取饥荒版本失败, err: %v", err))
		return
	}

	GameVersionServer.Store(int64(number))
	Logger.Info("游戏版本获取成功")
}

// internetIPGet 依次尝试两个公网IP接口，成功后更新缓存；全部失败时保留上一次的值。
func internetIPGet() {
	Logger.Info("正在获取公网IP")

	ip, err := getInternetIP1()
	if err != nil {
		Logger.Error(fmt.Sprintf("调用公网IP接口1失败, err: %v", err))
		ip, err = getInternetIP2()
		if err != nil {
			Logger.Error(fmt.Sprintf("调用公网IP接口2失败, err: %v", err))
			return
		}
	}

	if ip == "" {
		Logger.Error("获取公网IP失败, 接口返回为空")
		return
	}

	setInternetIP(ip)
	Logger.Info("公网IP获取成功")
}

// getInternetIP1 从接口1获取公网IP，接口返回 JSON，IP 在 query 字段中。
func getInternetIP1() (string, error) {
	res, err := client.Get(internetIPApi1)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP statusCode: %d", res.StatusCode)
	}

	var resp struct {
		Query string `json:"query"`
	}
	if err := json.NewDecoder(io.LimitReader(res.Body, internetIPMaxBody)).Decode(&resp); err != nil {
		return "", err
	}

	return resp.Query, nil
}

// getInternetIP2 从接口2获取公网IP，接口返回纯文本，需要正则提取。
func getInternetIP2() (string, error) {
	res, err := client.Get(internetIPApi2)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		_ = Body.Close()
	}(res.Body)

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP statusCode: %d", res.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, internetIPMaxBody))
	if err != nil {
		return "", err
	}

	matches := internetIPPattern.FindStringSubmatch(string(body))
	if len(matches) < 2 {
		return "", fmt.Errorf("响应中没有解析到公网IP")
	}

	return matches[1], nil
}
