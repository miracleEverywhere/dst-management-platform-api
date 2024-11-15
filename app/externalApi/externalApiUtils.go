package externalApi

import (
	"bufio"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type DSTVersion struct {
	Local  int `json:"local"`
	Server int `json:"server"`
}

func GetDSTVersion() (DSTVersion, error) { // 打开文件
	var dstVersion DSTVersion
	dstVersion.Server = -1
	dstVersion.Local = -1
	file, err := os.Open(utils.DSTLocalVersionPath)
	if err != nil {
		return dstVersion, err
	}
	defer file.Close() // 确保文件在函数结束时关闭

	// 创建一个扫描器来读取文件内容
	scanner := bufio.NewScanner(file)

	// 扫描文件的第一行
	if scanner.Scan() {
		// 读取第一行的文本
		line := scanner.Text()

		// 将字符串转换为整数
		number, err := strconv.Atoi(line)
		if err != nil {
			return dstVersion, err
		}
		dstVersion.Local = number
		// 获取服务端版本
		// 发送 HTTP GET 请求
		response, err := http.Get(utils.DSTServerVersionApi)
		if err != nil {
			return dstVersion, err
		}
		defer response.Body.Close() // 确保在函数结束时关闭响应体

		// 检查 HTTP 状态码
		if response.StatusCode != http.StatusOK {
			return dstVersion, fmt.Errorf("HTTP 请求失败，状态码: %d", response.StatusCode)
		}

		// 读取响应体内容
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return dstVersion, err
		}

		// 将字节数组转换为字符串并返回
		serverVersion, err := strconv.Atoi(string(body))
		if err != nil {
			return dstVersion, err
		}

		dstVersion.Server = serverVersion

		return dstVersion, nil
	}

	// 如果扫描器遇到错误，返回错误
	if err := scanner.Err(); err != nil {
		dstVersion.Server = -1
		dstVersion.Local = -1
		return dstVersion, err
	}

	// 如果文件为空，返回错误
	dstVersion.Server = -1
	dstVersion.Local = -1
	return dstVersion, fmt.Errorf("文件为空")
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
		Timeout: 3 * time.Second, // 设置超时时间为5秒
	}
	httpResponse, err := client.Get(utils.InternetIPApi1)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体

	// 检查 HTTP 状态码
	if httpResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP 请求失败，状态码: %d", httpResponse.StatusCode)
	}
	var jsonResp JSONResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&jsonResp); err != nil {
		fmt.Println("解析JSON失败:", err)
		return "", err
	}
	return jsonResp.Query, nil
}

func GetInternetIP2() (string, error) {
	type JSONResponse struct {
		Ip string `json:"ip"`
	}
	client := &http.Client{
		Timeout: 3 * time.Second, // 设置超时时间为5秒
	}
	httpResponse, err := client.Get(utils.InternetIPApi2)
	if err != nil {
		return "", err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			fmt.Println(err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体

	// 检查 HTTP 状态码
	if httpResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP 请求失败，状态码: %d", httpResponse.StatusCode)
	}
	var jsonResp JSONResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&jsonResp); err != nil {
		fmt.Println("解析JSON失败:", err)
		return "", err
	}
	return jsonResp.Ip, nil
}
