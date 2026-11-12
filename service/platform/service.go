package platform

import (
	"bufio"
	"dst-management-platform-api/aichat"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"encoding/json"
	"fmt"
	"io"
	nethttp "net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/mem"
	lua "github.com/yuin/gopher-lua"
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

// OverviewData 平台概览响应数据
type OverviewData struct {
	RunningTime int64   `json:"runningTime"`
	Memory      uint64  `json:"memory"`
	RoomCount   int64   `json:"roomCount"`
	WorldCount  int64   `json:"worldCount"`
	UserCount   int64   `json:"userCount"`
	UidCount    int64   `json:"uidCount"`
	MaxCpu      float64 `json:"maxCpu"`
	MaxMemory   float64 `json:"maxMemory"`
	MaxNetUp    float64 `json:"maxNetUp"`
	MaxNetDown  float64 `json:"maxNetDown"`
}

// Overview 平台概览
func (s *Service) Overview() service.Result {
	// 运行时间
	t := time.Since(utils.StartTime).Seconds()
	// 内存占用
	mem := utils.GetRES()
	// 房间数
	roomCount, err := s.roomDao.Count(nil)
	if err != nil {
		logger.Logger.Error("统计房间数失败")
		roomCount = 0
	}
	// 世界数
	worldCount, err := s.worldDao.Count(nil)
	if err != nil {
		logger.Logger.Error("统计世界数失败")
		worldCount = 0
	}
	// 用户数
	userCount, err := s.userDao.Count(nil)
	if err != nil {
		logger.Logger.Error("统计用户数失败")
		userCount = 0
	}
	// uid数
	uidCount, err := s.uidMapDao.Count(nil)
	if err != nil {
		logger.Logger.Error("统计用户数失败")
		uidCount = 0
	}
	// 1小时cpu内存网络最大值
	cache.SystemMetricsMutex.RLock()
	systemMetricsLength := len(cache.SystemMetrics)
	reqLength := 60
	var systemMetricsData []cache.SysMetrics
	if systemMetricsLength > reqLength {
		systemMetricsData = cache.SystemMetrics[systemMetricsLength-reqLength:]
	} else {
		systemMetricsData = cache.SystemMetrics
	}
	cache.SystemMetricsMutex.RUnlock()
	var maxCpu, maxMemory, maxNetUp, maxNetDown float64
	for _, m := range systemMetricsData {
		if m.Cpu > maxCpu {
			maxCpu = m.Cpu
		}
		if m.Memory > maxMemory {
			maxMemory = m.Memory
		}
		if m.NetUplink > maxNetUp {
			maxNetUp = m.NetUplink
		}
		if m.NetDownlink > maxNetDown {
			maxNetDown = m.NetDownlink
		}
	}

	// TODO 玩家数最多的的房间Top3

	data := OverviewData{
		RunningTime: int64(t),
		Memory:      mem,
		RoomCount:   roomCount,
		WorldCount:  worldCount,
		UserCount:   userCount,
		UidCount:    uidCount,
		MaxCpu:      maxCpu,
		MaxMemory:   maxMemory,
		MaxNetUp:    maxNetUp,
		MaxNetDown:  maxNetDown,
	}

	return service.Success(data)
}

// GameVersion 当前 DST 版本
func (s *Service) GameVersion() service.Result {
	return service.Success(scheduler.GetDSTVersion())
}

// OSInfo 系统信息
type OSInfo struct {
	Architecture    string
	OS              string
	CPUModel        string
	CPUCores        int
	CPUDetail       []cpu.InfoStat
	MemorySize      uint64
	Platform        string
	PlatformVersion string
	Uptime          uint64
}

func getOSInfo() (*OSInfo, error) {
	architecture := runtime.GOARCH

	// 获取CPU信息
	cpuInfo, err := cpu.Info()
	if err != nil {
		return nil, err
	}
	cpuModel := cpuInfo[0].ModelName
	cpuCount, _ := cpu.Counts(true)
	cpuCore := cpuCount
	cpuDetail := cpuInfo

	// 获取内存信息
	virtualMemory, err := mem.VirtualMemory()
	if err != nil {
		return nil, err
	}
	memorySize := virtualMemory.Total

	// 获取主机信息
	hostInfo, err := host.Info()
	if err != nil {
		return nil, err
	}
	platformVersion := hostInfo.PlatformVersion
	platform := hostInfo.Platform
	uptime := hostInfo.Uptime
	osName := hostInfo.OS
	// 返回系统信息
	return &OSInfo{
		Architecture:    architecture,
		OS:              osName,
		CPUModel:        cpuModel,
		CPUCores:        cpuCore,
		CPUDetail:       cpuDetail,
		MemorySize:      memorySize,
		Platform:        platform,
		Uptime:          uptime,
		PlatformVersion: platformVersion,
	}, nil
}

// OSInfoGet 获取系统信息
func (s *Service) OSInfoGet() service.Result {
	osInfo, err := getOSInfo()
	if err != nil {
		logger.Logger.Errorf("获取系统信息失败, err: %v", err)
		return service.Fail(constant.CodePlatformOSInfoFail)
	}

	return service.Success(osInfo)
}

// Metrics 系统监控数据，timeRange 单位为小时
func (s *Service) Metrics(timeRange int) service.Result {
	cache.SystemMetricsMutex.RLock()
	systemMetricsLength := len(cache.SystemMetrics)
	reqLength := timeRange * 60
	if reqLength <= 0 {
		reqLength = 60 // 默认1小时
	}

	var result service.Result
	if systemMetricsLength > reqLength {
		result = service.Success(cache.SystemMetrics[systemMetricsLength-reqLength:])
	} else {
		result = service.Success(cache.SystemMetrics)
	}
	cache.SystemMetricsMutex.RUnlock()

	return result
}

// GlobalSettingsGet 查询全局设置
func (s *Service) GlobalSettingsGet() service.Result {
	var globalSettings models.GlobalSetting

	err := s.globalSettingDao.GetGlobalSetting(&globalSettings)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	return service.Success(globalSettings)
}

// GlobalSettingsPost 更新全局设置
func (s *Service) GlobalSettingsPost(actor service.Actor, reqForm models.GlobalSetting) service.Result {
	// PlayerGetFrequency 秒
	// PlayerInfoSaveTime 天
	if reqForm.PlayerGetFrequency <= 0 || reqForm.PlayerGetFrequency > 86400 ||
		reqForm.PlayerInfoSaveTime <= 0 || reqForm.PlayerInfoSaveTime > 3650 ||
		(reqForm.SysMetricsEnable && (reqForm.SysMetricsSetting <= 0 || reqForm.SysMetricsSetting > 86400)) {
		return service.Fail(constant.CodePlatformBadRequest)
	}

	// 先完成全部参数校验，再产生副作用：任务更新、缓存裁剪和数据库写入都放在校验之后。
	rawWebhookSetting := strings.TrimSpace(reqForm.WebhookSetting)
	webhookSetting := rawWebhookSetting
	if webhookSetting == "" {
		webhookSetting = "[]"
	}
	var webhooks []webhook.GlobalWebhookItem
	if err := json.Unmarshal([]byte(webhookSetting), &webhooks); err != nil {
		logger.Logger.Infof("webhook参数错误, api: %s", actor.Path)
		return service.Fail(constant.CodePlatformBadRequest)
	}
	for _, w := range webhooks {
		if !utils.IsValidURL(w.URL) {
			logger.Logger.Warnf("非法请求已拦截, api: %s, username: %s", actor.Path, actor.Username)
			return service.Fail(constant.CodePlatformInvalidURL)
		}
	}
	// 自动更新使用 Day 类型 + At，时间格式非法时 UpdateJob 会失败，必须提前校验。
	if reqForm.AutoUpdateEnable && !validDayAt(reqForm.AutoUpdateSetting) {
		logger.Logger.Infof("自动更新时间参数错误, api: %s", actor.Path)
		return service.Fail(constant.CodePlatformBadRequest)
	}

	var dbGlobalSettings models.GlobalSetting

	err := s.globalSettingDao.GetGlobalSetting(&dbGlobalSettings)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	needUpdateDB := false

	if dbGlobalSettings.PlayerGetFrequency != reqForm.PlayerGetFrequency || dbGlobalSettings.PlayerInfoSaveTime != reqForm.PlayerInfoSaveTime || dbGlobalSettings.UIDMaintainEnable != reqForm.UIDMaintainEnable {
		needUpdateDB = true
		err = scheduler.UpdateJob(&scheduler.JobConfig{
			Name:     "onlinePlayerGet",
			Func:     scheduler.OnlinePlayerGet,
			Args:     []any{reqForm.PlayerGetFrequency, reqForm.PlayerInfoSaveTime, reqForm.UIDMaintainEnable},
			TimeType: scheduler.SecondType,
			Interval: reqForm.PlayerGetFrequency,
			DayAt:    "",
		})
		if err != nil {
			logger.Logger.Errorf("定时任务设置失败, err: %v, name: %v", err, "onlinePlayerGet")
			return service.Fail(constant.CodePlatformUpdateFail)
		}

		// 任务更新成功后再裁剪缓存，避免任务没生效却已经丢掉统计历史
		cache.PlayersStatisticMutex.Lock()
		for roomID := range cache.PlayersStatistic {
			if len(cache.PlayersStatistic[roomID])*reqForm.PlayerGetFrequency > scheduler.ParsePlayerInfoSaveTime(reqForm.PlayerInfoSaveTime) {
				n := int(scheduler.ParsePlayerInfoSaveTime(reqForm.PlayerInfoSaveTime) / reqForm.PlayerGetFrequency)
				cache.PlayersStatistic[roomID] = utils.GetLastNElements(cache.PlayersStatistic[roomID], n)
			}
		}
		cache.PlayersStatisticMutex.Unlock()
	}

	if dbGlobalSettings.SysMetricsEnable != reqForm.SysMetricsEnable || dbGlobalSettings.SysMetricsSetting != reqForm.SysMetricsSetting {
		needUpdateDB = true
		if reqForm.SysMetricsEnable {
			err = scheduler.UpdateJob(&scheduler.JobConfig{
				Name:     "systemMetricsGet",
				Func:     scheduler.SystemMetricsGet,
				Args:     []any{reqForm.SysMetricsSetting},
				TimeType: scheduler.MinuteType,
				Interval: scheduler.SystemMetricsGetInInterval,
				DayAt:    "",
			})
			if err != nil {
				logger.Logger.Errorf("定时任务设置失败, err: %v, name: %v", err, "systemMetricsGet")
				return service.Fail(constant.CodePlatformUpdateFail)
			}
		} else {
			scheduler.DeleteJob("systemMetricsGet")
			cache.SystemMetricsMutex.Lock()
			cache.SystemMetrics = []cache.SysMetrics{}
			cache.SystemMetricsMutex.Unlock()
		}
	}

	if dbGlobalSettings.AutoUpdateEnable != reqForm.AutoUpdateEnable || dbGlobalSettings.AutoUpdateSetting != reqForm.AutoUpdateSetting || dbGlobalSettings.AutoUpdateRestart != reqForm.AutoUpdateRestart {
		needUpdateDB = true
		if reqForm.AutoUpdateEnable {
			err = scheduler.UpdateJob(&scheduler.JobConfig{
				Name:     "gameUpdate",
				Func:     scheduler.GameUpdate,
				Args:     []any{reqForm.AutoUpdateEnable, reqForm.AutoUpdateRestart},
				TimeType: scheduler.DayType,
				Interval: 0,
				DayAt:    reqForm.AutoUpdateSetting,
			})
			if err != nil {
				logger.Logger.Errorf("定时任务设置失败, err: %v, name: %v", err, "gameUpdate")
				return service.Fail(constant.CodePlatformUpdateFail)
			}
		} else {
			scheduler.DeleteJob("gameUpdate")
		}
	}

	// webhook 已在入口校验，这里只判断是否需要写库；空值按 [] 写入
	if strings.TrimSpace(dbGlobalSettings.WebhookSetting) != rawWebhookSetting {
		needUpdateDB = true
	}
	reqForm.WebhookSetting = webhookSetting

	if needUpdateDB {
		err = s.globalSettingDao.UpdateGlobalSetting(&reqForm)
		if err != nil {
			logger.Logger.Errorf("更新数据库失败, err: %v", err)
			return service.Fail(constant.CodePlatformDBError)
		}

		webhook.Snd.Send(webhook.EventGlobalSettingUpdated, 0, map[string]interface{}{
			"dbUpdated": needUpdateDB,
		})

		return service.Msg("update success", nil)
	}

	return service.Fail(constant.CodePlatformBadRequest)
}

// dayAtPattern 与 gocron 的 At 时间格式保持一致，避免校验通过但任务注册失败。
var dayAtPattern = regexp.MustCompile(`^\d{1,2}:\d{2}(:\d{2})?$`)

// validDayAt 校验定时任务的 At 时间，支持 HH:MM、HH:MM:SS 以及用 ; 分隔的多个时间。
func validDayAt(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	for _, item := range strings.Split(value, ";") {
		item = strings.TrimSpace(item)
		if !dayAtPattern.MatchString(item) {
			return false
		}
		layout := "15:04"
		if strings.Count(item, ":") == 2 {
			layout = "15:04:05"
		}
		if _, err := time.Parse(layout, item); err != nil {
			return false
		}
	}

	return true
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

// PluginList 插件列表
func (s *Service) PluginList(page, pageSize int, q string) service.Result {
	plugins, err := s.pluginDao.ListPlugins(q, page, pageSize)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	data := *plugins

	return service.Success(data)
}

// PluginInstall 安装插件，imageParse 仅 TMI 插件使用
func (s *Service) PluginInstall(actor service.Actor, name, proxy string, imageParse *bool) service.Result {
	switch name {
	case models.PluginTmi:
		if imageParse == nil {
			logger.Logger.Infof("请求参数缺少imageParse, api: %s", actor.Path)
			return service.Fail(constant.CodePlatformBadRequest)
		}
		parse := *imageParse
		if cache.OsType == constant.Darwin {
			parse = false
		}
		return s.installTmiPlugin(actor, proxy, parse)
	case models.PluginChat:
		return s.installAiChatPlugin(actor, proxy)
	default:
		logger.Logger.Infof("请求参数错误, api: %s", actor.Path)
		return service.Fail(constant.CodePlatformBadRequest)
	}
}

func (s *Service) installTmiPlugin(actor service.Actor, proxy string, imageParse bool) service.Result {
	plugin, err := s.pluginDao.GetPluginByPluginName(models.PluginTmi)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	if plugin.Step == 100 {
		return service.Msg("install success", nil)
	}

	var (
		images     []models.DstImage
		installErr error
	)

	if imageParse {
		if !utils.IsValidURL(proxy) && proxy != "" {
			logger.Logger.Warnf("非法代理url已拦截, api: %s, proxy: %s", actor.Path, proxy)
			return service.Fail(constant.CodePlatformBadRequest)
		}

		step := plugin.Step

		step, images, installErr = initTmi(step)
		plugin.Step = step
		if installErr != nil {
			if result, ok := s.updatePlugin(plugin); !ok {
				return result
			}
			return service.FailF(constant.CodePlatformInstallFail, installErr.Error())
		}
	} else {
		images, installErr = installTMIR("")
		if installErr != nil {
			plugin.Step = 5
			if result, ok := s.updatePlugin(plugin); !ok {
				return result
			}
			return service.FailF(constant.CodePlatformInstallFail, installErr.Error())
		}
		plugin.Step = 100
	}

	if err := s.dstImageDao.InitImages(images); err != nil {
		logger.Logger.Errorf("更新数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	plugin.Status = true
	if result, ok := s.updatePlugin(plugin); !ok {
		return result
	}

	if err := s.dstImageDao.DeleteNoName(); err != nil {
		logger.Logger.Warnf("清理异常图片失败: %v", err)
	}

	return service.Msg("install success", nil)
}

func (s *Service) installAiChatPlugin(actor service.Actor, proxy string) service.Result {
	if proxy != "" && !utils.IsValidURL(proxy) {
		logger.Logger.Warnf("非法代理url已拦截, api: %s, proxy: %s", actor.Path, proxy)
		return service.Fail(constant.CodePlatformBadRequest)
	}

	plugin, err := s.pluginDao.GetPluginByPluginName(models.PluginChat)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	if plugin.Step == 100 {
		return service.Msg("install success", nil)
	}

	original := *plugin

	if err := installAiChatWiki(proxy); err != nil {
		logger.Logger.Errorf("安装 AI Chat 插件失败: %v", err)
		return service.FailF(constant.CodePlatformInstallFail, err.Error())
	}

	plugin.Step = 100
	plugin.Status = true
	if result, ok := s.updatePlugin(plugin); !ok {
		return result
	}

	// 卸载会调用 StopAll，重新安装后必须像启用流程一样恢复房间监听。
	if err := s.aiManager.Start(); err != nil {
		if !original.Status {
			s.aiManager.StopAll()
		}
		plugin.Status = original.Status
		plugin.Step = original.Step
		if restoreErr := s.pluginDao.UpdatePlugin(plugin); restoreErr != nil {
			logger.Logger.Errorf("恢复插件状态失败, plugin: %s, err: %v", models.PluginChat, restoreErr)
		}
		logger.Logger.Errorf("启动 AI 对话服务失败: %v", err)
		return service.FailF(constant.CodePlatformInstallFail, err.Error())
	}

	return service.Msg("install success", nil)
}

// PluginAction 插件启用、禁用、更新、卸载
func (s *Service) PluginAction(actor service.Actor, name, action string) service.Result {
	plugin, err := s.pluginDao.GetPluginByPluginName(name)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	// 保存动作前的状态，运行时动作失败时用于回滚数据库与运行时
	original := *plugin
	wasEnabled := plugin.Status

	if result, ok := s.preparePluginAction(actor, plugin, name, action); !ok {
		return result
	}

	if err := s.pluginDao.UpdatePlugin(plugin); err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	if result, ok := s.applyPluginRuntimeAction(name, action); !ok {
		// Start 在已激活时是幂等的，ListEnabled 失败也不会停止现有监听；
		// 仅在原先未启用、可能已启动部分房间时清理运行时。
		if name == models.PluginChat && action == "enable" && !wasEnabled {
			s.aiManager.StopAll()
		}
		plugin.Status = original.Status
		plugin.Step = original.Step
		if restoreErr := s.pluginDao.UpdatePlugin(plugin); restoreErr != nil {
			logger.Logger.Errorf("恢复插件状态失败, plugin: %s, err: %v", name, restoreErr)
		}
		return result
	}

	s.notifyPluginAction(actor, name, action)

	return service.Msg("exec success", nil)
}

func (s *Service) preparePluginAction(actor service.Actor, plugin *models.Plugin, name, action string) (service.Result, bool) {
	switch action {
	case "enable":
		plugin.Status = true
	case "disable":
		plugin.Status = false
	case "update":
		if name == models.PluginChat {
			logger.Logger.Infof("请求参数错误, api: %s", actor.Path)
			return service.Fail(constant.CodePlatformBadRequest), false
		}
		if name == models.PluginTmi {
			if result, ok := s.updateTmiPlugin(); !ok {
				return result, false
			}
			plugin.Status = true
		}
	case "uninstall":
		pluginDir := pluginDirectory(name)
		if pluginDir == "" {
			return service.Fail(constant.CodePlatformBadRequest), false
		}
		if err := utils.RemoveDir(pluginDir); err != nil {
			logger.Logger.Errorf("卸载插件失败: %v", err)
			return service.Fail(constant.CodePlatformUninstallFail), false
		}
		plugin.Step = 0
		plugin.Status = false
	default:
		logger.Logger.Infof("请求参数错误, api: %s", actor.Path)
		return service.Fail(constant.CodePlatformBadRequest), false
	}
	return service.Result{}, true
}

func (s *Service) updateTmiPlugin() (service.Result, bool) {
	images, err := installTMIR("")
	if err != nil {
		logger.Logger.Errorf("更新插件失败: %v", err)
		return service.Fail(constant.CodePlatformUpdateFail), false
	}
	if err = s.dstImageDao.InitImages(images); err != nil {
		logger.Logger.Errorf("更新数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError), false
	}
	if err = s.dstImageDao.DeleteNoName(); err != nil {
		logger.Logger.Warnf("清理异常图片失败: %v", err)
	}
	return service.Result{}, true
}

func pluginDirectory(name string) string {
	switch name {
	case models.PluginTmi:
		return constant.PluginTmiPath
	case models.PluginChat:
		return constant.PluginAiChatPath
	default:
		return ""
	}
}

func (s *Service) applyPluginRuntimeAction(name, action string) (service.Result, bool) {
	if name != models.PluginChat {
		return service.Result{}, true
	}

	switch action {
	case "enable":
		if err := s.aiManager.Start(); err != nil {
			logger.Logger.Errorf("启动 AI 对话服务失败: %v", err)
			return service.Fail(constant.CodePlatformUpdateFail), false
		}
	case "disable", "uninstall":
		s.aiManager.StopAll()
	}
	return service.Result{}, true
}

func (s *Service) notifyPluginAction(actor service.Actor, name, action string) {
	// webhook 通知
	webhook.Snd.Send(webhook.EventPluginUpdated, 0, map[string]interface{}{
		"username":    actor.Username,
		"plugin_name": name,
		"action":      action,
	})
}

// PluginStatus 查询插件状态
func (s *Service) PluginStatus(name string) service.Result {
	plugin, err := s.pluginDao.GetPluginByPluginName(name)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError)
	}

	return service.Success(plugin.Status)
}

// updatePlugin 更新插件记录，失败时返回数据库错误结果
func (s *Service) updatePlugin(plugin *models.Plugin) (service.Result, bool) {
	if err := s.pluginDao.UpdatePlugin(plugin); err != nil {
		logger.Logger.Errorf("更新数据库失败, err: %v", err)
		return service.Fail(constant.CodePlatformDBError), false
	}
	return service.Result{}, true
}

// stexDir 是旧版本下载的 Stex 二进制目录，现在改用内置解包，仅保留用于升级清理
const stexDir = constant.PluginTmiPath + "/stex"
const tmirID = constant.TmirID
const gameImagesPath = constant.PluginTmiPath + "/dst_images"
const aiChatWikiURL = "https://raw.githubusercontent.com/miracleEverywhere/dst-management-platform-api/master/aichat/wiki/wiki.zip"

var tmirPath = fmt.Sprintf("%s/tmir/steamapps/workshop/content/322330/%d/scripts/TMIR/itemlist/lists", constant.PluginTmiPath, constant.TmirID)

func installAiChatWiki(proxy string) error {
	downloadURL := aiChatWikiURL
	if proxy != "" {
		if proxy[len(proxy)-1] != '/' {
			proxy += "/"
		}
		downloadURL = proxy + aiChatWikiURL
	}

	if err := utils.EnsureDirExists(constant.PluginAiChatPath); err != nil {
		return fmt.Errorf("创建 AI Chat 插件目录失败: %w", err)
	}
	temporaryDir, err := os.MkdirTemp(constant.PluginAiChatPath, ".wiki-install-")
	if err != nil {
		return fmt.Errorf("创建 Wiki 临时目录失败: %w", err)
	}
	defer os.RemoveAll(temporaryDir)

	client := &nethttp.Client{Timeout: constant.HttpTimeout * time.Second}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("下载 Wiki 压缩包失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != nethttp.StatusOK {
		return fmt.Errorf("下载 Wiki 压缩包失败，HTTP 状态码: %s", resp.Status)
	}

	zipPath := filepath.Join(temporaryDir, "wiki.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("创建 Wiki 压缩包失败: %w", err)
	}
	_, copyErr := io.Copy(zipFile, resp.Body)
	closeErr := zipFile.Close()
	if copyErr != nil {
		return fmt.Errorf("保存 Wiki 压缩包失败: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("关闭 Wiki 压缩包失败: %w", closeErr)
	}

	temporaryWikiDir := filepath.Join(temporaryDir, "wiki")
	if err := utils.Unzip(zipPath, temporaryWikiDir); err != nil {
		return fmt.Errorf("解压 Wiki 压缩包失败: %w", err)
	}
	if err := os.Remove(zipPath); err != nil {
		return fmt.Errorf("清理 Wiki 压缩包失败: %w", err)
	}

	if err := utils.RemoveDir(constant.PluginAiChatWikiPath); err != nil {
		return fmt.Errorf("清理旧 Wiki 目录失败: %w", err)
	}
	if err := os.Rename(temporaryWikiDir, constant.PluginAiChatWikiPath); err != nil {
		return fmt.Errorf("替换 Wiki 目录失败: %w", err)
	}
	return nil
}

// 初始化Tmi工具，仅支持ubuntu24及以上
func initTmi(step int) (int, []models.DstImage, error) {
	steps := []struct {
		fn   func(string) ([]models.DstImage, error)
		code int
	}{
		{installDependency, 1},
		{removeStex, 2},
		{unzipDstImages, 3},
		{texToPng, 4},
		{installTMIR, 5},
	}

	startIndex := 0
	for i, s := range steps {
		if s.code == step {
			startIndex = i
			break
		}
	}

	var (
		err    error
		images []models.DstImage
	)

	for i := startIndex; i < len(steps); i++ {
		images, err = steps[i].fn("")
		if err != nil {
			return steps[i].code, []models.DstImage{}, err
		}
	}

	return 100, images, nil
}

// 安装依赖，使用内置 TEX 解包后只依赖 unzip
func installDependency(string) ([]models.DstImage, error) {
	var images []models.DstImage

	cmd := exec.Command("apt", "install", "-y", "unzip")
	if _, err := cmd.CombinedOutput(); err != nil {
		logger.Logger.Errorf("安装依赖失败: %v", err)
		return images, fmt.Errorf("安装依赖失败: %v", err)
	}

	return images, nil
}

// 旧版本会下载安装 Stex 二进制，现在改用内置 TEX 解包，这里清理遗留的安装目录
func removeStex(string) ([]models.DstImage, error) {
	var images []models.DstImage

	_ = utils.RemoveDir(stexDir)

	return images, nil
}

// 解压游戏tex文件
func unzipDstImages(string) ([]models.DstImage, error) {
	var images []models.DstImage

	// 幂等
	_ = utils.RemoveDir(gameImagesPath)

	_ = utils.EnsureDirExists(gameImagesPath)

	var dstDataPath string
	if cache.OsType == constant.Darwin {
		dstDataPath = constant.DarwinDstContentsPath + "/data"
	} else {
		dstDataPath = constant.GameMainPath + "/data"
	}

	cmdArgs := []string{
		dstDataPath + "/databundles/images.zip",
		"-d",
		gameImagesPath,
	}
	cmd := exec.Command("unzip", cmdArgs...)
	if _, err := cmd.CombinedOutput(); err != nil {
		_ = utils.RemoveDir(gameImagesPath)
		logger.Logger.Errorf("解压游戏tex失败: %v", err)
		return images, fmt.Errorf("解压游戏tex失败: %v", err)
	}

	return images, nil
}

// 游戏tex文件转为png文件，使用内置 TEX 解包
func texToPng(string) ([]models.DstImage, error) {
	var images []models.DstImage

	inventoryimages := []string{
		"inventoryimages1",
		"inventoryimages2",
		"inventoryimages3",
		"inventoryimages4",
	}
	for _, inventoryimage := range inventoryimages {
		xmlPath := filepath.Join(gameImagesPath, "images", inventoryimage+".xml")
		outputDir, _, err := utils.Unpack(xmlPath, gameImagesPath)
		if err != nil {
			cleanTexOutput(inventoryimages)
			logger.Logger.Errorf("游戏tex: %s 转png失败: %v", inventoryimage, err)
			return images, fmt.Errorf("游戏tex: %s 转png失败: %v", inventoryimage, err)
		}
		if err := movePngFiles(outputDir, gameImagesPath); err != nil {
			cleanTexOutput(inventoryimages)
			logger.Logger.Errorf("整理png文件失败: %v", err)
			return images, fmt.Errorf("整理png文件失败: %v", err)
		}
		_ = utils.RemoveDir(outputDir)
	}
	_ = utils.RemoveDir(filepath.Join(gameImagesPath, "images"))

	return images, nil
}

// movePngFiles 把解包目录里的 png 移到图集根目录，前端按 <prefab>.png 读取
func movePngFiles(srcDir, dstDir string) error {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".png") {
			continue
		}
		if err := os.Rename(filepath.Join(srcDir, entry.Name()), filepath.Join(dstDir, entry.Name())); err != nil {
			return err
		}
	}

	return nil
}

// cleanTexOutput 清理本次生成的部分产物，避免失败后留下残缺数据
func cleanTexOutput(inventoryimages []string) {
	entries, err := os.ReadDir(gameImagesPath)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".png") {
			continue
		}
		_ = os.Remove(filepath.Join(gameImagesPath, entry.Name()))
	}

	for _, inventoryimage := range inventoryimages {
		_ = utils.RemoveDir(filepath.Join(gameImagesPath, inventoryimage))
	}
}

// 安装TMIR 与 获取 翻译文件
func installTMIR(string) ([]models.DstImage, error) {
	var images []models.DstImage

	modDir := filepath.Join(cache.CurrentDir, constant.PluginTmiPath, "tmir")
	cmdArgs := []string{
		"+force_install_dir",
		modDir,
		"+login",
		"anonymous",
		"+workshop_download_item",
		"322330",
		fmt.Sprintf("%d", tmirID),
		"+quit",
	}
	cmd := exec.Command("steamcmd/steamcmd.sh", cmdArgs...)
	if _, err := cmd.CombinedOutput(); err != nil {
		_ = utils.RemoveDir(modDir)
		logger.Logger.Errorf("下载TMIR模组失败: %v", err)
		return images, fmt.Errorf("下载TMIR模组失败: %v", err)
	}

	var dstDataPath string
	if cache.OsType == constant.Darwin {
		dstDataPath = constant.DarwinDstContentsPath + "/data"
	} else {
		dstDataPath = constant.GameMainPath + "/data"
	}

	scriptPath := filepath.Join(dstDataPath, "databundles", "scripts.zip")
	tmpDir := filepath.Join(constant.PluginTmiPath, "tmp")
	gameChinesePo := filepath.Join(tmpDir, "scripts", "languages", "chinese_s.po")

	_ = utils.RemoveDir(tmpDir)

	cmdArgs = []string{
		scriptPath,
		"-d",
		tmpDir,
	}
	cmd = exec.Command("unzip", cmdArgs...)
	if _, err := cmd.CombinedOutput(); err != nil {
		logger.Logger.Errorf("解压游戏scripts失败: %v", err)
		return images, fmt.Errorf("解压游戏scripts失败: %v", err)
	}

	// 解析 PO 文件，构建 prefab -> {en, zh} 映射
	translations, err := parseChinesePo(gameChinesePo)
	if err != nil {
		logger.Logger.Errorf("解析PO文件失败: %v", err)
		return images, fmt.Errorf("解析PO文件失败: %v", err)
	}

	// 构建数据库数据
	items, err := tmiItemLists()
	if err != nil {
		logger.Logger.Errorf("获取tmir模组信息失败: %v", err)
		return images, fmt.Errorf("获取tmir模组信息失败: %v", err)
	}
	for _, category := range items {
		for _, prefab := range category.Items {
			nameEn, nameZh := "", ""
			if entry, ok := translations[strings.ToUpper(prefab)]; ok {
				nameEn = entry.MsgID
				nameZh = entry.MsgStr
			}
			images = append(images, models.DstImage{
				Prefab:   prefab,
				Category: category.Category,
				NameEn:   nameEn,
				NameZh:   nameZh,
			})
		}
	}

	// 删除临时目录
	_ = utils.RemoveDir(tmpDir)

	return images, nil
}

type poEntry struct {
	MsgID  string
	MsgStr string
}

// parseChinesePo 解析中文 PO 文件，返回大写 prefab -> {英文名, 中文名} 的映射
// 只解析 msgctxt 以 STRINGS.NAMES. 开头的条目
func parseChinesePo(filePath string) (map[string]poEntry, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开PO文件失败: %v", err)
	}
	defer file.Close()

	result := make(map[string]poEntry)
	scanner := bufio.NewScanner(file)

	var currentCtxt, currentMsgID, currentMsgStr string

	for scanner.Scan() {
		line := scanner.Text()

		switch {
		case strings.HasPrefix(line, "msgctxt "):
			// 遇到新条目，保存上一个（如果有）
			if currentCtxt != "" && strings.HasPrefix(currentCtxt, "STRINGS.NAMES.") {
				prefab := strings.TrimPrefix(currentCtxt, "STRINGS.NAMES.")
				result[prefab] = poEntry{MsgID: currentMsgID, MsgStr: currentMsgStr}
			}
			currentCtxt = parsePoValue(line)
			currentMsgID = ""
			currentMsgStr = ""
		case strings.HasPrefix(line, "msgid "):
			currentMsgID = parsePoValue(line)
		case strings.HasPrefix(line, "msgstr "):
			currentMsgStr = parsePoValue(line)
		}
	}

	// 保存最后一个条目
	if currentCtxt != "" && strings.HasPrefix(currentCtxt, "STRINGS.NAMES.") {
		prefab := strings.TrimPrefix(currentCtxt, "STRINGS.NAMES.")
		result[prefab] = poEntry{MsgID: currentMsgID, MsgStr: currentMsgStr}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("读取PO文件失败: %v", err)
	}

	return result, nil
}

// parsePoValue 解析 PO 文件中单行带引号的值，如 "Birchnut" -> Birchnut
func parsePoValue(line string) string {
	start := strings.Index(line, "\"")
	if start == -1 {
		return ""
	}
	end := strings.LastIndex(line, "\"")
	if end <= start {
		return ""
	}
	return line[start+1 : end]
}

type itemCategory struct {
	Category string   `json:"category"` // 分类名 (如 tool, food, equip)
	Items    []string `json:"items"`    // 该分类下的物品代码列表
}

// tmiItemLists 解析 mod 的 Lua 物品列表文件，返回按分类组织的物品代码
func tmiItemLists() ([]itemCategory, error) {
	entries, err := os.ReadDir(tmirPath)
	if err != nil {
		return nil, fmt.Errorf("读取物品列表目录失败: %v", err)
	}

	var categories []itemCategory

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), "itemlist_") || !strings.HasSuffix(entry.Name(), ".lua") {
			continue
		}

		// 从文件名提取分类名: itemlist_tool.lua → tool
		category := strings.TrimPrefix(entry.Name(), "itemlist_")
		category = strings.TrimSuffix(category, ".lua")

		filePath := filepath.Join(tmirPath, entry.Name())
		items, err := parseLuaItemList(filePath)
		if err != nil {
			return nil, fmt.Errorf("解析 %s 失败: %v", entry.Name(), err)
		}

		categories = append(categories, itemCategory{
			Category: category,
			Items:    items,
		})
	}

	// 按分类名排序，保证输出稳定
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Category < categories[j].Category
	})

	return categories, nil
}

// parseLuaItemList 解析单个 Lua 物品列表文件，格式为 return{"item1","item2",...}
func parseLuaItemList(filePath string) ([]string, error) {
	L := utils.NewSafeLuaState()
	defer L.Close()

	if err := L.DoFile(filePath); err != nil {
		return nil, fmt.Errorf("执行 Lua 文件失败: %v", err)
	}

	// 获取返回值 (Lua 栈顶)
	ret := L.Get(-1)
	if ret.Type() != lua.LTTable {
		return nil, fmt.Errorf("返回值不是 table 类型: %s", ret.Type().String())
	}

	table := ret.(*lua.LTable)
	var items []string

	table.ForEach(func(_, value lua.LValue) {
		if value.Type() == lua.LTString {
			items = append(items, value.String())
		}
	})

	return items, nil
}
