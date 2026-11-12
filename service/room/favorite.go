package room

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"gorm.io/gorm"
)

// FavoriteTypeDST 饥荒房间收藏类型
const FavoriteTypeDST = "dst"

// FavoriteItem 收藏接口返回结构，Data 解析为 JSON 对象
type FavoriteItem struct {
	ID        string         `json:"id"`
	Username  string         `json:"username"`
	Type      string         `json:"type"`
	Data      XRoomTotalInfo `json:"data"`
	Remark    string         `json:"remark"`
	CreatedAt int64          `json:"createdAt"`
	UpdatedAt int64          `json:"updatedAt"`
}

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

// favoriteFail 构造带可选格式化参数的收藏类失败结果
func favoriteFail(code int, arg string) service.Result {
	if arg == "" {
		return service.Fail(code)
	}

	return service.FailF(code, arg)
}

// FavoriteCreate 收藏已有房间的 room/world/roomSetting 配置
func (s *Service) FavoriteCreate(actor service.Actor, roomID int, remark string) service.Result {
	roomIDStr := strconv.Itoa(roomID)
	if !s.hasRoomPermission(actor, roomIDStr) {
		return service.Fail(constant.CodeRoomNoPermission)
	}

	room, err := s.roomDao.GetRoomByID(roomID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return service.Fail(constant.CodeRoomNotFound)
		}
		logger.Logger.Errorf("查询房间失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	worlds, err := s.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("查询世界失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	if len(*worlds) == 0 {
		return service.Fail(constant.CodeRoomFavoriteNoWorlds)
	}

	roomSetting, err := s.roomSettingDao.GetRoomSettingsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("查询房间设置失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	data := RoomSnapshot(room, worlds, roomSetting)
	raw, err := json.Marshal(data)
	if err != nil {
		logger.Logger.Errorf("序列化收藏数据失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	now := utils.GetTimestamp()
	favorite := &models.Favorite{
		Username:  actor.Username,
		Type:      FavoriteTypeDST,
		Data:      string(raw),
		Remark:    remark,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if err = s.favoriteDao.CreateFavorite(favorite); err != nil {
		logger.Logger.Errorf("创建收藏失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	item := favoriteItemFromData(favorite, data)
	return service.Msg("favorite create success", item)
}

// FavoriteList 分页查询当前用户的 dst 收藏
func (s *Service) FavoriteList(actor service.Actor, page, pageSize int) service.Result {
	favorites, err := s.favoriteDao.ListFavorites(actor.Username, FavoriteTypeDST, page, pageSize)
	if err != nil {
		logger.Logger.Errorf("查询收藏失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	items := make([]FavoriteItem, 0, len(favorites.Data))
	for i := range favorites.Data {
		item, errItem := favoriteItemFromModel(&favorites.Data[i])
		if errItem != nil {
			logger.Logger.Errorf("解析收藏数据失败, favoriteID: %s, err: %v", favorites.Data[i].ID, errItem)
			return service.Fail(constant.CodeRoomFavoriteDataInvalid)
		}
		items = append(items, *item)
	}

	data := dao.PaginatedResult[FavoriteItem]{
		Data:       items,
		Page:       favorites.Page,
		PageSize:   favorites.PageSize,
		TotalCount: favorites.TotalCount,
	}

	return service.Success(data)
}

// FavoriteDetail 查询当前用户的单条收藏
func (s *Service) FavoriteDetail(actor service.Actor, id string) service.Result {
	favorite, err := s.favoriteDao.GetFavoriteByID(id, actor.Username, FavoriteTypeDST)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return service.Fail(constant.CodeRoomFavoriteNotFound)
		}
		logger.Logger.Errorf("查询收藏失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	item, err := favoriteItemFromModel(favorite)
	if err != nil {
		logger.Logger.Errorf("解析收藏数据失败, favoriteID: %s, err: %v", favorite.ID, err)
		return service.Fail(constant.CodeRoomFavoriteDataInvalid)
	}

	return service.Success(item)
}

// FavoriteUpdate 修改收藏备注
func (s *Service) FavoriteUpdate(actor service.Actor, id string, remark *string) service.Result {
	favorite, err := s.favoriteDao.GetFavoriteByID(id, actor.Username, FavoriteTypeDST)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return service.Fail(constant.CodeRoomFavoriteNotFound)
		}
		logger.Logger.Errorf("查询收藏失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	favorite.Remark = *remark
	favorite.UpdatedAt = utils.GetTimestamp()

	if err = s.favoriteDao.UpdateFavorite(favorite); err != nil {
		logger.Logger.Errorf("更新收藏失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	item, err := favoriteItemFromModel(favorite)
	if err != nil {
		logger.Logger.Errorf("解析收藏数据失败, favoriteID: %s, err: %v", favorite.ID, err)
		return service.Fail(constant.CodeRoomFavoriteDataInvalid)
	}

	return service.Msg("favorite update success", item)
}

// FavoriteDelete 删除当前用户的收藏
func (s *Service) FavoriteDelete(actor service.Actor, id string) service.Result {
	rows, err := s.favoriteDao.DeleteFavorite(id, actor.Username, FavoriteTypeDST, utils.GetTimestamp())
	if err != nil {
		logger.Logger.Errorf("删除收藏失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	if rows == 0 {
		return service.Fail(constant.CodeRoomFavoriteNotFound)
	}

	return service.Msg("favorite delete success", nil)
}

// FavoriteApply 使用收藏创建新房间
func (s *Service) FavoriteApply(actor service.Actor, id, gameNameInput string) service.Result {
	permission, err := s.hasCreatePermission(actor)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	if !permission {
		return service.Fail(constant.CodeRoomNoPermission)
	}

	favorite, err := s.favoriteDao.GetFavoriteByID(id, actor.Username, FavoriteTypeDST)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return service.Fail(constant.CodeRoomFavoriteNotFound)
		}
		logger.Logger.Errorf("查询收藏失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	var data XRoomTotalInfo
	if err = json.Unmarshal([]byte(favorite.Data), &data); err != nil {
		logger.Logger.Errorf("解析收藏数据失败, favoriteID: %s, err: %v", favorite.ID, err)
		return service.Fail(constant.CodeRoomFavoriteDataInvalid)
	}
	if code, arg := ValidateFavoriteData(&data); code != 0 {
		return favoriteFail(code, arg)
	}

	// 与创建房间接口保持一致：创建房间时不携带自定义启动命令，避免命令注入。
	// 如确实需要，可在房间创建后由管理员通过专用接口设置。
	if ClearFavoriteCustomStartupCmd(&data) {
		logger.Logger.Warnf("应用收藏时已忽略自定义启动命令, favoriteID: %s, username: %s", favorite.ID, actor.Username)
	}

	gameName := strings.TrimSpace(gameNameInput)
	if gameName == "" {
		gameName = strings.TrimSpace(data.RoomData.GameName)
	}
	if gameName == "" {
		return service.Fail(constant.CodeRoomFavoriteNameRequired)
	}

	room := data.RoomData
	room.ID = 0
	room.Status = true
	room.GameName = gameName
	if strings.TrimSpace(room.Token) == "" {
		room.Token = utils.GetDstToken()
	}
	if strings.TrimSpace(room.ClusterKey) == "" {
		room.ClusterKey = utils.RandomString(14)
	}
	if strings.TrimSpace(room.MasterIP) == "" {
		room.MasterIP = "127.0.0.1"
	}

	roomCount, err := s.roomDao.Count(nil)
	if err != nil {
		logger.Logger.Errorf("查询房间数量失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}
	worldCount, err := s.worldDao.Count(nil)
	if err != nil {
		logger.Logger.Errorf("查询世界数量失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	room.MasterPort = 21000 + int(roomCount) + 1
	worlds := data.WorldData
	for i := range worlds {
		worlds[i].ID = 0
		worlds[i].RoomID = 0
		worlds[i].LastAliveTime = ""
		worlds[i].ServerPort = 11000 + int(worldCount) + i + 1
		worlds[i].MasterServerPort = 31000 + int(worldCount) + i + 1
		worlds[i].AuthenticationPort = 41000 + int(worldCount) + i + 1
	}

	var ports []int
	for _, world := range worlds {
		ports = append(ports, world.ServerPort, world.MasterServerPort, world.AuthenticationPort)
	}
	if conflictPort := s.checkGamePort(ports, 0); conflictPort != 0 {
		return service.FailF(constant.CodeRoomPortConflict, conflictPort)
	}

	setting := data.RoomSettingData
	setting.RoomID = 0

	if err = s.roomDao.CreateConfiguration(&room, &worlds, &setting); err != nil {
		logger.Logger.Errorf("创建房间失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	game := dst.NewGameController(&room, &worlds, &setting, actor.Lang)
	if err = game.SaveAll(); err != nil {
		logger.Logger.Errorf("配置写入磁盘失败, err: %v", err)
		return service.Fail(constant.CodeRoomWriteFileFail)
	}

	processJobs(game, room.ID, setting)

	if !actor.IsAdmin() {
		if err = s.addRoomToUser(actor.Username, strconv.Itoa(room.ID)); err != nil {
			logger.Logger.Errorf("更新用户房间权限失败, err: %v", err)
			return service.Fail(constant.CodeRoomDBError)
		}
	}

	webhook.Snd.Send(webhook.EventRoomCreated, room.ID, XRoomTotalInfo{
		RoomData:        room,
		WorldData:       worlds,
		RoomSettingData: setting,
	})

	return service.Msg("favorite apply success", room)
}

// addRoomToUser 将新房间加入非管理员用户的房间权限列表
func (s *Service) addRoomToUser(username, roomID string) error {
	user, err := s.userDao.GetUserByUsername(username)
	if err != nil {
		return err
	}

	rooms := make([]string, 0)
	if user.Rooms != "" {
		rooms = strings.Split(user.Rooms, ",")
	}
	rooms = append(rooms, roomID)
	user.Rooms = strings.Join(rooms, ",")

	return s.userDao.UpdateUser(user)
}
