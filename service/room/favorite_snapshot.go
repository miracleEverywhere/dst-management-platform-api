package room

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"encoding/json"
	"strings"
)

// RoomSnapshot 生成 XRoomTotalInfo 房间快照，清空本地关联 ID，保留 token。
// 收藏与云存档元数据共用该结构，保证云端看到的数据形状一致。
func RoomSnapshot(room *models.Room, worlds *[]models.World, setting *models.RoomSetting) XRoomTotalInfo {
	roomData := *room
	roomData.ID = 0

	worldData := make([]models.World, len(*worlds))
	for i, world := range *worlds {
		world.ID = 0
		world.RoomID = 0
		world.LastAliveTime = ""
		worldData[i] = world
	}

	roomSetting := *setting
	roomSetting.RoomID = 0

	return XRoomTotalInfo{
		RoomData:        roomData,
		WorldData:       worldData,
		RoomSettingData: roomSetting,
	}
}

// ValidateFavoriteData 校验收藏数据，返回错误码和文案的格式化参数（0 表示校验通过）。
// 云同步下载的快照同样必须经过该函数校验。
func ValidateFavoriteData(data *XRoomTotalInfo) (int, string) {
	if len(data.WorldData) == 0 {
		return constant.CodeRoomFavoriteNoWorlds, ""
	}

	if worldName, code := validateWorldNames(data.WorldData); code != 0 {
		return code, worldName
	}

	if !utils.IsValidGameMode(data.RoomData.GameMode) {
		return constant.CodeRoomFavoriteInvalidGameMode, ""
	}
	if data.RoomData.GameMode == "custom" && data.RoomData.CustomGameMode != "" && !utils.IsValidGameMode(data.RoomData.CustomGameMode) {
		return constant.CodeRoomFavoriteInvalidGameMode, ""
	}

	if strings.TrimSpace(data.RoomSettingData.WebhookSetting) != "" {
		var webhooks []webhook.WebhookItem
		if err := json.Unmarshal([]byte(data.RoomSettingData.WebhookSetting), &webhooks); err != nil {
			return constant.CodeRoomFavoriteInvalidWebhook, ""
		}
		for _, item := range webhooks {
			if !utils.IsValidURL(item.URL) {
				return constant.CodeRoomInvalidURL, ""
			}
		}
	}

	return 0, ""
}

// ClearFavoriteCustomStartupCmd 清空自定义启动命令，返回是否有内容被清空。
func ClearFavoriteCustomStartupCmd(data *XRoomTotalInfo) bool {
	cleared := false
	for i := range data.WorldData {
		if data.WorldData[i].CustomStartupCmd != "" {
			data.WorldData[i].CustomStartupCmd = ""
			cleared = true
		}
	}

	return cleared
}

func favoriteItemFromModel(favorite *models.Favorite) (*FavoriteItem, error) {
	var data XRoomTotalInfo
	if err := json.Unmarshal([]byte(favorite.Data), &data); err != nil {
		return nil, err
	}

	item := favoriteItemFromData(favorite, data)
	return &item, nil
}

func favoriteItemFromData(favorite *models.Favorite, data XRoomTotalInfo) FavoriteItem {
	return FavoriteItem{
		ID:        favorite.ID,
		Username:  favorite.Username,
		Type:      favorite.Type,
		Data:      data,
		Remark:    favorite.Remark,
		CreatedAt: favorite.CreatedAt,
		UpdatedAt: favorite.UpdatedAt,
	}
}
