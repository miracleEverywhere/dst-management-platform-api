package player

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"encoding/json"
	"fmt"
	"io"
	nethttp "net/http"
	"strconv"
	"time"
)

// List 获取房间的黑名单/白名单/管理员名单
func (s *Service) List(actor service.Actor, roomID int, listType string) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodePlayerBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodePlayerNoPermission)
	}
	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodePlayerDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	list := game.GetPlayerList(listType)

	return service.Success(list)
}

// UpdateList 修改房间的黑名单/白名单/管理员名单
func (s *Service) UpdateList(actor service.Actor, roomID int, uids []string, listType, actionType string) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodePlayerBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodePlayerNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodePlayerDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)

	switch actionType {
	case "add":
		err = game.AddPlayerList(uids, listType)
		if err != nil {
			logger.Logger.Infof("修改player list失败: %v", err)
			return service.Fail(constant.CodePlayerAddFail)
		}

		return service.Msg("add success", nil)
	case "delete":
		if len(uids) == 0 || uids[0] == "" {
			return service.Fail(constant.CodePlayerBadRequest)
		}
		err = game.RemovePlayerList(uids[0], listType)
		if err != nil {
			logger.Logger.Infof("修改player list失败: %v", err)
			return service.Fail(constant.CodePlayerDeleteFail)
		}

		return service.Msg("delete success", nil)
	case "deleteAll":
		err = game.RemovePlayerListAll(listType)
		if err != nil {
			logger.Logger.Infof("删除player list失败: %v", err)
			return service.Fail(constant.CodePlayerDeleteFail)
		}

		return service.Msg("delete success", nil)
	case "import":
		var importUids []string

		importUids, err = getPublicBlockList()
		if err != nil {
			logger.Logger.Error(err.Error())
		}

		err = game.AddPlayerList(importUids, listType)
		if err != nil {
			logger.Logger.Infof("修改player list失败: %v", err)
			return service.Fail(constant.CodePlayerAddFail)
		}

		return service.Msg("add success", nil)
	default:
		logger.Logger.Infof("请求参数错误, api: %s", actor.Path)
		return service.Fail(constant.CodePlayerBadRequest)
	}
}

var client *nethttp.Client = &nethttp.Client{
	Timeout: constant.HttpTimeout * time.Second,
}

// getPublicBlockList 拉取公开的饥荒黑名单列表
func getPublicBlockList() ([]string, error) {
	httpResponse, err := client.Get(constant.DstBlockList)
	if err != nil {
		return []string{}, err
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			logger.Logger.Errorf("请求关闭失败, err: %v", err)
		}
	}(httpResponse.Body) // 确保在函数结束时关闭响应体
	// 检查 HTTP 状态码
	if httpResponse.StatusCode != nethttp.StatusOK {
		return []string{}, fmt.Errorf("获取饥荒黑名单列表失败，HTTP Code: %d", httpResponse.StatusCode)
	}

	var responseData struct {
		Uids []string `json:"uids"`
	}
	if err = json.NewDecoder(httpResponse.Body).Decode(&responseData); err != nil {
		return []string{}, fmt.Errorf("解析饥荒黑名单失败, err: %v", err)
	}

	return responseData.Uids, nil
}
