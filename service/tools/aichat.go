package tools

import (
	"dst-management-platform-api/aichat"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// AISettingPutInput 房间 AI 设置请求体。
// 同时接受平铺字段与旧版 {setting: {...}} 结构，嵌套对象也只解析房间级字段。
type AISettingPutInput struct {
	RoomID         int                   `json:"roomID"`
	Enabled        bool                  `json:"enabled"`
	AllowChat      bool                  `json:"allowChat"`
	Prefix         string                `json:"prefix"`
	MaxResults     int                   `json:"maxResults"`
	MaxReplyLength int                   `json:"maxReplyLength"`
	Setting        *models.RoomAISetting `json:"setting"`
}

// AIBaseSettingResponse AI 基础设置响应，附带磁盘上已有向量索引的维度（没有索引时为 -1）
type AIBaseSettingResponse struct {
	models.AIBaseSetting
	EmbeddingIndexDimensions int `json:"embeddingIndexDimensions"`
}

// AISettingGet 获取房间 AI 设置，未配置时返回默认值
func (s *Service) AISettingGet(actor service.Actor, roomID int) service.Result {
	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeToolsNoPermission)
	}

	setting, err := s.roomAISettingDao.GetByRoomID(roomID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		setting = &models.RoomAISetting{
			RoomID:         roomID,
			MaxResults:     models.DefaultAIWikiMaxResults,
			MaxReplyLength: models.DefaultAIReplyMaxLength,
		}
	} else if err != nil {
		logger.Logger.Errorf("获取房间 AI 配置失败, roomID: %d, err: %v", roomID, err)
		return service.Fail(constant.CodeToolsDBError)
	}
	if setting.MaxResults == 0 {
		setting.MaxResults = models.DefaultAIWikiMaxResults
	}
	if setting.MaxReplyLength < models.MinAIReplyLength {
		setting.MaxReplyLength = models.DefaultAIReplyMaxLength
	}

	return service.Success(setting)
}

// AISettingPut 保存房间 AI 设置并重载该房间的 AI 对话
func (s *Service) AISettingPut(actor service.Actor, input AISettingPutInput) service.Result {
	setting := models.RoomAISetting{
		RoomID:         input.RoomID,
		Enabled:        input.Enabled,
		AllowChat:      input.AllowChat,
		Prefix:         input.Prefix,
		MaxResults:     input.MaxResults,
		MaxReplyLength: input.MaxReplyLength,
	}
	// 兼容旧版 {setting: {...}} 请求体，嵌套对象也只能解析房间级字段。
	if input.Setting != nil {
		setting = *input.Setting
	}

	if !s.hasPermission(actor, strconv.Itoa(setting.RoomID)) {
		return service.Fail(constant.CodeToolsNoPermission)
	}
	if _, err := s.roomDao.GetRoomByID(setting.RoomID); err != nil {
		logger.Logger.Errorf("获取房间信息失败, roomID: %d, err: %v", setting.RoomID, err)
		return service.Fail(constant.CodeToolsDBError)
	}
	if err := aichat.ValidateRoomSetting(&setting); err != nil {
		return service.FailMsg(constant.CodeToolsEmbeddingConfigInvalid, err.Error())
	}
	if setting.Enabled {
		baseSetting, err := s.systemDao.GetAIBaseSetting()
		if err != nil {
			logger.Logger.Errorf("获取 AI 基础配置失败, err: %v", err)
			return service.Fail(constant.CodeToolsDBError)
		}
		if err = aichat.ValidateBaseSetting(baseSetting); err != nil {
			return service.FailMsg(constant.CodeToolsEmbeddingConfigInvalid, err.Error())
		}
	}

	if err := s.roomAISettingDao.UpdateSetting(&setting); err != nil {
		logger.Logger.Errorf("更新房间 AI 配置失败, roomID: %d, err: %v", setting.RoomID, err)
		return service.Fail(constant.CodeToolsDBError)
	}

	webhook.Snd.Send(webhook.EventAIChatSettingUpdated, setting.RoomID, map[string]interface{}{
		"enabled":        setting.Enabled,
		"allowChat":      setting.AllowChat,
		"prefix":         setting.Prefix,
		"maxResults":     setting.MaxResults,
		"maxReplyLength": setting.MaxReplyLength,
		"username":       actor.Username,
	})
	if err := s.aiManager.Reload(setting.RoomID); err != nil {
		logger.Logger.Errorf("重载房间 AI 对话失败, roomID: %d, err: %v", setting.RoomID, err)
		return service.Fail(constant.CodeToolsUpdateFail)
	}

	return service.Msg("update success", nil)
}

// AIBaseSettingGet 获取 AI 基础设置
func (s *Service) AIBaseSettingGet() service.Result {
	setting, err := s.systemDao.GetAIBaseSetting()
	if err != nil {
		logger.Logger.Errorf("获取 AI 基础配置失败, err: %v", err)
		return service.Fail(constant.CodeToolsDBError)
	}

	// embeddingIndexDimensions 为磁盘上已有向量索引的维度，没有索引时为 -1。
	response := AIBaseSettingResponse{
		AIBaseSetting:            *setting,
		EmbeddingIndexDimensions: s.aiManager.GetEmbeddingIndexDimensions(),
	}

	return service.Success(response)
}

// AIBaseSettingPut 保存 AI 基础设置并重载 AI 对话服务
func (s *Service) AIBaseSettingPut(actor service.Actor, setting models.AIBaseSetting) service.Result {
	if err := aichat.ValidateBaseSetting(&setting); err != nil {
		return service.FailMsg(constant.CodeToolsEmbeddingConfigInvalid, err.Error())
	}
	if err := s.systemDao.UpdateAIBaseSetting(&setting); err != nil {
		logger.Logger.Errorf("更新 AI 基础配置失败, err: %v", err)
		return service.Fail(constant.CodeToolsDBError)
	}

	webhook.Snd.Send(webhook.EventAIChatBaseSettingUpdated, 0, map[string]interface{}{
		"username":       actor.Username,
		"chatModel":      setting.ChatModel,
		"embeddingModel": setting.EmbeddingModel,
	})
	if err := s.aiManager.Reload(); err != nil {
		logger.Logger.Errorf("重载 AI 对话服务失败, err: %v", err)
		return service.Fail(constant.CodeToolsUpdateFail)
	}

	return service.Msg("update success", nil)
}

// AIKeywordIndexRebuild 重建关键词索引
func (s *Service) AIKeywordIndexRebuild() service.Result {
	_ = s.aiManager.BuildKeywordIndex(true)

	return service.Msg("update success", nil)
}

// AIEmbeddingIndexRebuild 重建 embedding 索引
func (s *Service) AIEmbeddingIndexRebuild() service.Result {
	setting, err := s.systemDao.GetAIBaseSetting()
	if err != nil {
		logger.Logger.Errorf("获取 AI 基础配置失败, err: %v", err)
		return service.Fail(constant.CodeToolsDBError)
	}

	baseURL := strings.TrimSpace(setting.EmbeddingBaseURL)
	model := strings.TrimSpace(setting.EmbeddingModel)
	if baseURL == "" || setting.EmbeddingApiKey == "" || model == "" {
		return service.Fail(constant.CodeToolsEmbeddingConfigIncomplete)
	}
	if !utils.IsValidURL(baseURL) {
		return service.Fail(constant.CodeToolsEmbeddingBaseURLInvalid)
	}

	err = s.aiManager.BuildEmbeddingIndex(aichat.EmbeddingConfig{
		APIURL:     baseURL,
		APIKey:     setting.EmbeddingApiKey,
		Model:      model,
		Dimensions: setting.EmbeddingDimensions,
	}, true)
	if err != nil {
		logger.Logger.Errorf("重建 embedding 索引失败, err: %v", err)
		return service.Fail(constant.CodeToolsUpdateFail)
	}

	return service.Msg("update success", nil)
}

// AIEmbeddingIndexCancel 取消正在进行的 embedding 索引构建
func (s *Service) AIEmbeddingIndexCancel() service.Result {
	if !s.aiManager.CancelEmbeddingIndexBuild() {
		return service.Fail(constant.CodeToolsEmbeddingNotRunning)
	}

	return service.Msg("embedding build cancel success", nil)
}
