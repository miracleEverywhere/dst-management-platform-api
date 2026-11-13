package platform

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

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
