package platform

import (
	"bufio"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"github.com/shirou/gopsutil/v3/process"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Handler struct {
	userDao   *dao.UserDAO
	roomDao   *dao.RoomDAO
	worldDao  *dao.WorldDAO
	systemDao *dao.SystemDAO
}

func NewHandler(userDao *dao.UserDAO, roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, systemDao *dao.SystemDAO) *Handler {
	return &Handler{
		userDao:   userDao,
		roomDao:   roomDao,
		worldDao:  worldDao,
		systemDao: systemDao,
	}
}

func getRES() uint64 {
	p, err := process.NewProcess(int32(os.Getpid()))
	if err != nil {
		return 0
	}

	memoryInfo, err := p.MemoryInfo()
	if err != nil {
		return 0
	}

	return memoryInfo.RSS
}

type DSTVersion struct {
	Local  int `json:"local"`
	Server int `json:"server"`
}

func GetDSTVersion() DSTVersion { // 打开文件
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
