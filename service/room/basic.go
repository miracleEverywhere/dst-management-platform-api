package room

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"strconv"
	"strings"
)

// Factor 前端自动分配端口
func (s *Service) Factor() service.Result {
	roomCount, err := s.roomDao.Count(nil)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	worldCount, err := s.worldDao.Count(nil)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	type Data struct {
		Room  int64 `json:"roomCount"`
		World int64 `json:"worldCount"`
	}

	return service.Success(Data{
		Room:  roomCount,
		World: worldCount,
	})
}

// AllBasic 获取room基本信息 name和id
func (s *Service) AllBasic() service.Result {
	rooms, err := s.roomDao.GetRoomBasic()
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	return service.Success(rooms)
}

// PermittedBasic 获取当前用户有权限的room基本信息
func (s *Service) PermittedBasic(actor service.Actor) service.Result {
	rooms, err := s.roomDao.GetRoomBasic()
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	if actor.IsAdmin() {
		return service.Success(rooms)
	}

	user, err := s.userDao.GetUserByUsername(actor.Username)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	roomIDs := make(map[int]struct{})
	for _, roomID := range strings.Split(user.Rooms, ",") {
		if roomID == "" {
			continue
		}
		id, errAtoi := strconv.Atoi(roomID)
		if errAtoi != nil {
			logger.Logger.Errorf("用户房间权限数据无效, username: %s, roomID: %s", user.Username, roomID)
			return service.Fail(constant.CodeRoomDBError)
		}
		roomIDs[id] = struct{}{}
	}

	permittedRooms := make([]dao.RoomBasic, 0, len(roomIDs))
	for _, room := range *rooms {
		if _, ok := roomIDs[room.RoomID]; ok {
			permittedRooms = append(permittedRooms, room)
		}
	}

	return service.Success(permittedRooms)
}

// Worlds 房间下的世界列表
func (s *Service) Worlds(actor service.Actor, roomID int) service.Result {
	if !s.hasRoomPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeRoomNoPermission)
	}

	worlds, err := s.worldDao.GetWorldsByRoomID(roomID)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		return service.Fail(constant.CodeRoomDBError)
	}

	type Data struct {
		ID        int    `json:"id"`
		WorldName string `json:"worldName"`
	}

	var data []Data

	for _, world := range *worlds {
		data = append(data, Data{
			ID:        world.ID,
			WorldName: world.WorldName,
		})
	}

	return service.Success(data)
}
