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
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			utils.Logger.Error("关闭文件失败", "err", err)
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
			return dstVersion, err
		}
		dstVersion.Local = number
		// 获取服务端版本
		// 发送 HTTP GET 请求
		response, err := http.Get(utils.DSTServerVersionApi)
		if err != nil {
			return dstVersion, err
		}
		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				utils.Logger.Error("关闭文件失败", "err", err)
			}
		}(response.Body) // 确保在函数结束时关闭响应体

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
			utils.Logger.Error("请求关闭失败", "err", err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体

	// 检查 HTTP 状态码
	if httpResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP 请求失败，状态码: %d", httpResponse.StatusCode)
	}
	var jsonResp JSONResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&jsonResp); err != nil {
		utils.Logger.Error("解析JSON失败", "err", err)
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
			utils.Logger.Error("请求关闭失败", "err", err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体

	// 检查 HTTP 状态码
	if httpResponse.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP 请求失败，状态码: %d", httpResponse.StatusCode)
	}
	var jsonResp JSONResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&jsonResp); err != nil {
		utils.Logger.Error("解析JSON失败", "err", err)
		return "", err
	}
	return jsonResp.Ip, nil
}

type Tags struct {
	Tag         string `json:"tag"`
	DisplayName string `json:"display_name"`
}
type VoteData struct {
	Score     float64 `json:"score"`
	VotesUp   int     `json:"votes_up"`
	VotesDown int     `json:"votes_down"`
}
type PublishedFileDetails struct {
	ID              string   `json:"publishedfileid"`
	FileSize        string   `json:"file_size"`
	FileDescription string   `json:"file_description"`
	FileUrl         string   `json:"file_url"`
	Title           string   `json:"title"`
	Tags            []Tags   `json:"tags"`
	PreviewUrl      string   `json:"preview_url"`
	VoteData        VoteData `json:"vote_data"`
}
type Response struct {
	Publishedfiledetails []PublishedFileDetails `json:"publishedfiledetails"`
}
type JSONResponse struct {
	Response Response `json:"response"`
}
type ModInfo struct {
	Name            string   `json:"name"`
	ID              int      `json:"id"`
	Size            string   `json:"size"`
	Tags            []Tags   `json:"tags"`
	PreviewUrl      string   `json:"preview_url"`
	FileDescription string   `json:"file_description"`
	FileUrl         string   `json:"file_url"`
	VoteData        VoteData `json:"vote_data"`
}

func GetModsInfo(luaScriptContent string, lang string) ([]ModInfo, error) {
	var language int
	if lang == "zh" {
		language = 6
	} else {
		language = 0
	}
	mods := utils.ModOverridesToStruct(luaScriptContent)
	url := fmt.Sprintf("%s?language=%d&key=%s", utils.SteamApiModDetail, language, utils.SteamApiKey)
	for index, mod := range mods {
		url = url + fmt.Sprintf("&publishedfileids[%d]=%d", index, mod.ID)
	}

	client := &http.Client{
		Timeout: 5 * time.Second, // 设置超时时间为5秒
	}
	httpResponse, err := client.Get(url)
	if err != nil {
		return []ModInfo{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			utils.Logger.Error("请求关闭失败", "err", err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体
	// 检查 HTTP 状态码
	if httpResponse.StatusCode != http.StatusOK {
		return []ModInfo{}, err
	}
	var jsonResp JSONResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&jsonResp); err != nil {
		utils.Logger.Error("解析JSON失败", "err", err)
		return []ModInfo{}, err
	}

	var modInfoList []ModInfo
	for _, i := range jsonResp.Response.Publishedfiledetails {
		modInfo := ModInfo{
			ID:         func() int { id, _ := strconv.Atoi(i.ID); return id }(),
			Name:       i.Title,
			Size:       i.FileSize,
			Tags:       i.Tags,
			PreviewUrl: i.PreviewUrl,
		}
		modInfoList = append(modInfoList, modInfo)
	}

	return modInfoList, nil

	//L := lua.NewState()
	//defer L.Close()
	//
	//if err := L.DoString(luaScriptContent); err != nil {
	//	return nil, fmt.Errorf("加载 Lua 文件失败: %w", err)
	//}
	//
	//modsLuaTable := L.Get(-1)
	//var modInfoList []ModInfo
	//var wg sync.WaitGroup
	//var mu sync.Mutex
	//
	//if tbl, ok := modsLuaTable.(*lua.LTable); ok {
	//	re := regexp.MustCompile(`\d+`)
	//
	//	tbl.ForEach(func(key lua.LValue, value lua.LValue) {
	//		// 检查键是否是字符串，并且以 "workshop-" 开头
	//		if strKey, ok := key.(lua.LString); ok && strings.HasPrefix(string(strKey), "workshop-") {
	//			// 提取 "workshop-" 后面的数字
	//			modID := re.FindString(string(strKey))
	//
	//			wg.Add(1)
	//			go func(modID string) {
	//				defer wg.Done()
	//
	//				url := fmt.Sprintf("%s?language=%d&publishedfileids[0]=%s&key=%s", utils.SteamApiModDetail, 6, modID, utils.SteamApiKey)
	//				client := &http.Client{
	//					Timeout: 5 * time.Second, // 设置超时时间为5秒
	//				}
	//				httpResponse, err := client.Get(url)
	//				if err != nil {
	//					return
	//				}
	//				defer func(Body io.ReadCloser) {
	//					err := Body.Close()
	//					if err != nil {
	//						utils.Logger.Error("请求关闭失败", "err", err)
	//					}
	//				}(httpResponse.Body) // 确保在函数结束时关闭响应体
	//
	//				// 检查 HTTP 状态码
	//				if httpResponse.StatusCode != http.StatusOK {
	//					return
	//				}
	//
	//				var jsonResp JSONResponse
	//				if err := json.NewDecoder(httpResponse.Body).Decode(&jsonResp); err != nil {
	//					utils.Logger.Error("解析JSON失败", "err", err)
	//					return
	//				}
	//
	//				modInfo := ModInfo{
	//					ID:         modID,
	//					Name:       jsonResp.Response.Publishedfiledetails[0].Title,
	//					Size:       jsonResp.Response.Publishedfiledetails[0].FileSize,
	//					Tags:       jsonResp.Response.Publishedfiledetails[0].Tags,
	//					PreviewUrl: jsonResp.Response.Publishedfiledetails[0].PreviewUrl,
	//				}
	//
	//				mu.Lock()
	//				modInfoList = append(modInfoList, modInfo)
	//				mu.Unlock()
	//			}(modID)
	//		}
	//	})
	//}
	//
	//wg.Wait()
	//return modInfoList, nil
}

func SearchMod(page int, pageSize int, searchText string, lang string) ([]ModInfo, error) {
	var (
		language int
		url      string
	)
	if lang == "zh" {
		language = 6
	} else {
		language = 0
	}
	if searchText == "" {
		url = fmt.Sprintf("%s?appid=322330&return_vote_data=true&return_children=true&language=%d&key=%s&page=%d&numperpage=%d",
			utils.SteamApiModSearch,
			language,
			utils.SteamApiKey,
			page,
			pageSize,
		)
	} else {
		url = fmt.Sprintf("%s?appid=322330&return_vote_data=true&return_children=true&language=%d&key=%s&page=%d&numperpage=%d&search_text=%s",
			utils.SteamApiModSearch,
			language,
			utils.SteamApiKey,
			page,
			pageSize,
			searchText,
		)
	}
	fmt.Println(url)
	client := &http.Client{
		Timeout: 5 * time.Second, // 设置超时时间为5秒
	}
	httpResponse, err := client.Get(url)
	if err != nil {
		return []ModInfo{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			utils.Logger.Error("请求关闭失败", "err", err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体
	// 检查 HTTP 状态码
	if httpResponse.StatusCode != http.StatusOK {
		return []ModInfo{}, err
	}
	var jsonResp JSONResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&jsonResp); err != nil {
		utils.Logger.Error("解析JSON失败", "err", err)
		return []ModInfo{}, err
	}

	var modInfoList []ModInfo
	for _, i := range jsonResp.Response.Publishedfiledetails {
		modInfo := ModInfo{
			ID:              func() int { id, _ := strconv.Atoi(i.ID); return id }(),
			Name:            i.Title,
			Size:            i.FileSize,
			Tags:            i.Tags,
			PreviewUrl:      i.PreviewUrl,
			FileDescription: i.FileDescription,
			FileUrl:         i.FileUrl,
			VoteData:        i.VoteData,
		}
		modInfoList = append(modInfoList, modInfo)
	}

	return modInfoList, nil

}
