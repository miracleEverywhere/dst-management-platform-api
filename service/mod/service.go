package mod

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Service 模组模块业务实现，不依赖 gin
type Service struct {
	roomDao      *dao.RoomDAO
	worldDao     *dao.WorldDAO
	userDao      *dao.UserDAO
	modInfoDao   *dao.ModInfoDAO
	bannedModDao *dao.BannedModDAO
}

// New 创建模组模块 service
func New(roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, userDao *dao.UserDAO, modInfoDao *dao.ModInfoDAO, bannedModDao *dao.BannedModDAO) *Service {
	return &Service{
		roomDao:      roomDao,
		worldDao:     worldDao,
		userDao:      userDao,
		modInfoDao:   modInfoDao,
		bannedModDao: bannedModDao,
	}
}

// 模组信息缓存时长 10分钟
const cacheModInfoMaxAgeMs = 10 * 60 * 1000

var client *http.Client = &http.Client{
	Timeout: constant.HttpTimeout * time.Second,
}

// ModSearch 按 id 或文本搜索模组
func (s *Service) ModSearch(actor service.Actor, searchType, searchText string, page, pageSize int) service.Result {
	langStr := normalizeModLanguage(actor.Lang)

	if searchType == "id" {
		id, err := strconv.Atoi(searchText)
		if err != nil || id <= 0 {
			logger.Logger.Infof("请求参数错误: %v, api: %s", err, actor.Path)
			return service.Fail(constant.CodeModBadRequest)
		}

		modInfo, err := s.ModInfoByID(langStr, id)
		// Steam 查不到该 ID 时保持原有契约：成功返回空列表，而不是报搜索失败。
		if errors.Is(err, errModInfoNotFound) {
			return service.Success(Data{Total: 1, Page: 1, PageSize: 1})
		}
		if err != nil {
			logger.Logger.Errorf("获取mod信息失败, err: %v", err)
			return service.Fail(constant.CodeModSearchFail)
		}

		data := Data{
			Total:    1,
			Page:     1,
			PageSize: 1,
			Rows: []models.ModInfo{
				*modInfo,
			},
		}
		return service.Success(data)
	}

	data, err := searchMod(page, pageSize, searchText, langStr)
	if err != nil {
		logger.Logger.Errorf("获取mod信息失败, err: %v", err)
		return service.Fail(constant.CodeModSearchFail)
	}

	return service.Success(data)
}

func (s *Service) hasPermission(actor service.Actor, roomID string) bool {
	// 管理员直接返回true
	if actor.IsAdmin() {
		return true
	}

	dbUser, err := s.userDao.GetUserByUsername(actor.Username)
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

	return false
}

func checkNotUgcUrl(url string) bool {
	if url == "" {
		return true
	}

	urlParts := strings.Split(url, "/")
	if len(urlParts) != 7 {
		return false
	}

	if urlParts[2] != "cdn.steamusercontent.com" {
		return false
	}

	return true
}

func checkCacheValidity(ts int64) bool {
	if ts <= 0 {
		return false
	}

	tsDiff := time.Now().UnixMilli() - ts
	return tsDiff >= 0 && tsDiff < cacheModInfoMaxAgeMs
}

func normalizeModLanguage(lang string) string {
	if lang == "en" {
		return "en"
	}
	return "zh"
}
