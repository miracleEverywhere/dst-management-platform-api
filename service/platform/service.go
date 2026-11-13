package platform

import (
	"dst-management-platform-api/aichat"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"fmt"
	"strconv"
)

// Service 平台模块业务实现，不依赖 gin
type Service struct {
	userDao          *dao.UserDAO
	roomDao          *dao.RoomDAO
	worldDao         *dao.WorldDAO
	systemDao        *dao.SystemDAO
	globalSettingDao *dao.GlobalSettingDAO
	uidMapDao        *dao.UidMapDAO
	pluginDao        *dao.PluginDAO
	dstImageDao      *dao.DstImageDAO
	aiManager        *aichat.Manager
}

// New 创建平台模块 service
func New(userDao *dao.UserDAO, roomDao *dao.RoomDAO, worldDao *dao.WorldDAO, systemDao *dao.SystemDAO, globalSettingDao *dao.GlobalSettingDAO, uidMapDao *dao.UidMapDAO, pluginDao *dao.PluginDAO, dstImageDao *dao.DstImageDAO, aiManager *aichat.Manager) *Service {
	return &Service{
		userDao:          userDao,
		roomDao:          roomDao,
		worldDao:         worldDao,
		systemDao:        systemDao,
		globalSettingDao: globalSettingDao,
		uidMapDao:        uidMapDao,
		pluginDao:        pluginDao,
		dstImageDao:      dstImageDao,
		aiManager:        aiManager,
	}
}

// ReadmeGet 查询 README 阅读状态
func (s *Service) ReadmeGet() service.Result {
	readme, err := s.systemDao.Get(models.ReadmeKey)
	if err != nil {
		logger.Logger.Errorf("获取README阅读状态失败: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	value, err := strconv.ParseBool(readme.Value)
	if err != nil {
		logger.Logger.Errorf("README阅读状态无效: %q", readme.Value)
		return service.Fail(constant.CodePlatformDBError)
	}

	return service.Success(value)
}

// ReadmePost 标记 README 已读
func (s *Service) ReadmePost() service.Result {
	if err := s.systemDao.Set(models.ReadmeKey, "true"); err != nil {
		logger.Logger.Errorf("更新README阅读状态失败: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	return service.Success(true)
}

// GameVersion 当前 DST 版本
func (s *Service) GameVersion() service.Result {
	return service.Success(scheduler.GetDSTVersion())
}

// ScreenRunning 查询房间正在运行的 screen
func (s *Service) ScreenRunning(actor service.Actor, roomID int) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodePlatformBadRequest)
	}
	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	screens, err := game.RunningScreens()
	if err != nil {
		logger.Logger.Errorf("获取正在运行的screen失败, err: %v", err)
		return service.Result{Code: constant.CodePlatformScreensFail, Data: []string{}}
	}

	return service.Success(screens)
}

// KillScreen 关闭指定 screen
func (s *Service) KillScreen(actor service.Actor, screenName string) service.Result {
	if screenName == "" {
		logger.Logger.Infof("请求参数错误, api: %s", actor.Path)
		return service.Fail(constant.CodePlatformBadRequest)
	}

	// 校验 ScreenName 只允许字母、数字、下划线和连字符，防止命令注入
	for _, ch := range screenName {
		if !((ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-') {
			logger.Logger.Infof("ScreenName包含非法字符, api: %s", actor.Path)
			return service.Fail(constant.CodePlatformBadRequest)
		}
	}

	cmd := fmt.Sprintf("screen -X -S %s quit", utils.ShellQuote(screenName))
	err := utils.BashCMD(cmd)
	if err != nil {
		logger.Logger.Warnf("关闭Screen失败: %v", err)
		return service.Fail(constant.CodePlatformKillScreenFail)
	}

	return service.Msg("kill screen success", nil)
}

// WebhookTest 发送 webhook 测试请求
func (s *Service) WebhookTest(actor service.Actor, url, secret string) service.Result {
	// webhook url 安全检测
	if !utils.IsValidURL(url) {
		logger.Logger.Warnf("非法请求已拦截, api: %s, username: %s", actor.Path, actor.Username)
		return service.Fail(constant.CodePlatformInvalidURL)
	}

	err := webhook.Snd.SendTest(url, secret)
	if err != nil {
		logger.Logger.Warnf("webhook 测试失败, url: %s, err: %v", url, err)
		return service.FailF(constant.CodePlatformWebhookTestFail, err.Error())
	}

	return service.Msg("webhook test success", nil)
}

// WebhookEvents 可选 webhook 事件类型
func (s *Service) WebhookEvents() service.Result {
	return service.Success(webhook.AllEventTypes)
}
