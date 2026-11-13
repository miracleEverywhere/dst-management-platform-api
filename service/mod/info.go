package mod

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

func (s *Service) addDownloadedModInfo(mods *[]dst.DownloadedMod, lang string) error {
	if len(*mods) == 0 {
		return nil
	}

	var ids []int
	for _, mod := range *mods {
		ids = append(ids, mod.ID)
	}

	lang = normalizeModLanguage(lang)
	cacheValid := s.modInfoDao != nil
	modInfos := []models.ModInfo(nil)
	if cacheValid {
		var err error
		modInfos, err = s.modInfoDao.GetModInfoByIDs(ids, lang)
		cacheValid = err == nil
	}

	if cacheValid {
		cachedByID := make(map[int]models.ModInfo, len(modInfos))
		for _, modInfo := range modInfos {
			cachedByID[modInfo.ID] = modInfo
		}
		for idx := range *mods {
			modInfo, ok := cachedByID[(*mods)[idx].ID]
			if !ok || !checkCacheValidity(modInfo.CacheTimestamp) {
				cacheValid = false
				continue
			}
			(*mods)[idx].Name = modInfo.Name
			(*mods)[idx].FileURL = modInfo.FileUrl
			(*mods)[idx].PreviewURL = modInfo.PreviewUrl
			(*mods)[idx].ServerSize = modInfo.Size
		}
	}

	if cacheValid {
		logger.Logger.Debugf("返回模组缓存数据")
		return nil
	}

	var language int
	if lang == "zh" {
		language = 6
	} else {
		language = 0
	}

	req, err := http.NewRequest("GET", constant.SteamApiModDetail, nil)
	if err != nil {
		return err
	}

	q := req.URL.Query()
	q.Add("language", strconv.Itoa(language))
	q.Add("key", utils.GetSteamApiKey())
	for index, mod := range *mods {
		q.Add(fmt.Sprintf("publishedfileids[%d]", index), strconv.Itoa(mod.ID))
	}
	req.URL.RawQuery = q.Encode()

	httpResponse, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Logger.Errorf("请求关闭失败, err: %v", err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体
	// 检查 HTTP 状态码
	if httpResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("steam api returned status %d", httpResponse.StatusCode)
	}
	var jsonResp JSONResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&jsonResp); err != nil {
		logger.Logger.Errorf("解析JSON失败, err: %v", err)
		return err
	}

	for _, i := range jsonResp.Response.Publishedfiledetails {
		id := func() int { id, _ := strconv.Atoi(i.ID); return id }()
		for idx := range *mods {
			if (*mods)[idx].ID == id {
				(*mods)[idx].Name = i.Title
				(*mods)[idx].FileURL = i.FileUrl
				(*mods)[idx].PreviewURL = i.PreviewUrl
				(*mods)[idx].ServerSize = i.FileSize
			}
		}
		if s.modInfoDao != nil {
			modInfo := modInfoFromPublishedFile(i, lang)
			modInfo.CacheTimestamp = utils.GetTimestamp()
			if err = s.modInfoDao.UpdateModInfo(&modInfo); err != nil {
				logger.Logger.Errorf("更新模组缓存信息失败, err: %v", err)
			}
		}

	}

	return nil
}
