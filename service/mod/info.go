package mod

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
)

// errModInfoNotFound 表示 Steam 没有返回该 ID 的创意工坊信息（ID 不存在或已下架）。
var errModInfoNotFound = errors.New("steam 未返回该模组的信息")

// ModInfoByID 按模组 ID 查询 Steam 创意工坊信息，优先读取本地缓存。
// 缓存未命中或已过期时回源 Steam 并写回缓存；lang 只区分 zh/en，其余取值按 zh 处理。
// 供其它 service（如 AI 诊断工具）复用，因此直接返回模型与错误，不包装 service.Result。
// Steam 查不到该 ID 时返回 errModInfoNotFound，调用方可据此区分“查不到”与“请求失败”。
func (s *Service) ModInfoByID(lang string, modID int) (*models.ModInfo, error) {
	if modID <= 0 {
		return nil, fmt.Errorf("模组 ID 必须为正整数: %d", modID)
	}
	lang = normalizeModLanguage(lang)

	if s.modInfoDao != nil {
		cachedModInfo, err := s.modInfoDao.GetModInfoByID(modID, lang)
		if err == nil && cachedModInfo != nil && checkCacheValidity(cachedModInfo.CacheTimestamp) {
			logger.Logger.Debugf("返回模组缓存数据, modID: %d", modID)
			return cachedModInfo, nil
		}
	}

	data, err := searchModById(modID, lang)
	if err != nil {
		return nil, err
	}
	if len(data.Rows) == 0 {
		return nil, fmt.Errorf("%w: %d", errModInfoNotFound, modID)
	}

	modInfo := data.Rows[0]
	if s.modInfoDao != nil {
		modInfo.CacheTimestamp = utils.GetTimestamp()
		if cacheErr := s.modInfoDao.UpdateModInfo(&modInfo); cacheErr != nil {
			logger.Logger.Errorf("更新模组缓存信息失败, err: %v", cacheErr)
		}
	}
	return &modInfo, nil
}

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
