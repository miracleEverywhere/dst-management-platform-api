package room

import (
	"dst-management-platform-api/aichat"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"strings"
)

// Service 房间模块业务实现，不依赖 gin
type Service struct {
	roomDao          *dao.RoomDAO
	userDao          *dao.UserDAO
	worldDao         *dao.WorldDAO
	roomSettingDao   *dao.RoomSettingDAO
	favoriteDao      *dao.FavoriteDAO
	globalSettingDao *dao.GlobalSettingDAO
	uidMapDao        *dao.UidMapDAO
	roomAISettingDao *dao.RoomAISettingDAO
	aiManager        *aichat.Manager
}

// New 创建房间模块 service
func New(userDao *dao.UserDAO, roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, roomSettingDao *dao.RoomSettingDAO, favoriteDao *dao.FavoriteDAO, globalSettingDao *dao.GlobalSettingDAO, uidMapDao *dao.UidMapDAO, roomAISettingDao *dao.RoomAISettingDAO, aiManager *aichat.Manager) *Service {
	return &Service{
		roomDao:          roomDao,
		userDao:          userDao,
		worldDao:         worldDao,
		roomSettingDao:   roomSettingDao,
		favoriteDao:      favoriteDao,
		globalSettingDao: globalSettingDao,
		uidMapDao:        uidMapDao,
		roomAISettingDao: roomAISettingDao,
		aiManager:        aiManager,
	}
}

// Partition 分页参数
type Partition struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"pageSize" form:"pageSize"`
}

// XRoomWorld 房间列表项，附带世界与玩家统计
type XRoomWorld struct {
	models.Room
	Worlds  []models.World  `json:"worlds"`
	Players []cache.Players `json:"players"`
}

// RoomWorldPage 房间列表分页结构，绑定失败时需要返回零值保持原响应形状
type RoomWorldPage = dao.PaginatedResult[XRoomWorld]

// XRoomTotalInfo 房间完整配置，收藏与云存档元数据共用
type XRoomTotalInfo struct {
	RoomData        models.Room        `json:"roomData"`
	WorldData       []models.World     `json:"worldData"`
	RoomSettingData models.RoomSetting `json:"roomSettingData"`
}

// validateWorldNames 校验世界名是否合法
// 世界名会被拼接进bash命令和文件路径，因此必须限定字符集；
// 同时同一房间内世界名不能重复，否则多个世界会使用同一个目录、互相覆盖存档。
// 返回第一个不合法的世界名和对应的错误码，全部合法时返回空字符串和 0
func validateWorldNames(worlds []models.World) (string, int) {
	seen := make(map[string]struct{}, len(worlds))
	for _, world := range worlds {
		name := world.WorldName
		if !utils.IsSafeString(name) {
			return name, constant.CodeRoomInvalidWorldName
		}
		if _, ok := seen[name]; ok {
			return name, constant.CodeRoomDuplicateWorldName
		}
		seen[name] = struct{}{}
	}

	return "", 0
}

// hasCreatePermission 是否拥有房间创建权限
func (s *Service) hasCreatePermission(actor service.Actor) (bool, error) {
	var (
		has    bool
		err    error
		dbUser *models.User
	)

	// 管理员直接返回true
	if actor.IsAdmin() {
		has = true
	} else {
		dbUser, err = s.userDao.GetUserByUsername(actor.Username)
		if err != nil {
			return has, err
		}
		if dbUser.RoomCreation {
			has = true
		}
	}

	return has, err
}

// hasRoomPermission 是否拥有对应房间权限
func (s *Service) hasRoomPermission(actor service.Actor, roomID string) bool {
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

// CreateGuard 校验房间创建权限，返回 nil 表示允许继续。
// 原实现先做权限判断再解析请求体，handler 需要在绑定前调用它保持响应顺序。
func (s *Service) CreateGuard(actor service.Actor) *service.Result {
	permission, err := s.hasCreatePermission(actor)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		r := service.Fail(constant.CodeRoomDBError)
		return &r
	}
	if !permission {
		r := service.Fail(constant.CodeRoomNoPermission)
		return &r
	}

	return nil
}

// WebhookTest 测试 webhook 地址
func (s *Service) WebhookTest(actor service.Actor, url, secret string) service.Result {
	// webhook url 安全检测
	if !utils.IsValidURL(url) {
		logger.Logger.Warnf("非法请求已拦截, api: %s, username: %s", actor.Path, actor.Username)
		return service.Fail(constant.CodeRoomInvalidURL)
	}

	err := webhook.Snd.SendTest(url, secret)
	if err != nil {
		logger.Logger.Warnf("webhook 测试失败, url: %s, err: %v", url, err)
		return service.FailF(constant.CodeRoomWebhookTestFail, err.Error())
	}

	return service.Msg("webhook test success", nil)
}

// WebhookEvents 返回可用的 webhook 事件类型
func (s *Service) WebhookEvents() service.Result {
	return service.Success(webhook.AllEventTypes)
}

// checkGamePort 检测端口是否冲突，excludeRoomID 为 0 时不排除任何房间
// 先检测入参内部是否重复，再检测与数据库中已有的世界端口是否冲突
// MasterPort（[SHARD] master_port）不参与检测，允许重复
// 返回冲突的端口，0 表示无冲突
func (s *Service) checkGamePort(ports []int, excludeRoomID int) int {
	// 1. 检测入参端口自身是否重复
	seen := make(map[int]bool)
	for _, port := range ports {
		if seen[port] {
			return port
		}
		seen[port] = true
	}

	// 2. 获取所有世界端口
	worlds, err := s.worldDao.Query(1, 10000, nil)
	if err != nil {
		logger.Logger.Errorf("checkGamePort 查询世界失败, err: %v", err)
		return 0
	}

	// 3. 已使用的端口集合
	usedPorts := make(map[int]bool)
	for _, world := range worlds.Data {
		if world.RoomID != excludeRoomID {
			usedPorts[world.ServerPort] = true
			usedPorts[world.MasterServerPort] = true
			usedPorts[world.AuthenticationPort] = true
		}
	}

	// 4. 检查与已有端口是否冲突
	for _, port := range ports {
		if usedPorts[port] {
			return port
		}
	}

	return 0
}
