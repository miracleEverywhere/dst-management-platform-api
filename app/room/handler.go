package room

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"github.com/gin-gonic/gin"
	"net/http"
	"strings"
)

// createPost 创建房间
func (h *Handler) createPost(c *gin.Context) {
	role, _ := c.Get("role")
	username, _ := c.Get("username")
	hasPermission := false

	if role.(string) == "admin" {
		hasPermission = true
	} else {
		dbUser, err := h.userDao.GetUserByUsername(username.(string))
		if err != nil {
			logger.Logger.Error("查询数据库失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}
		if dbUser.RoomCreation {
			hasPermission = true
		}
	}

	if hasPermission {
		var room models.Room
		if err := c.ShouldBindJSON(&room); err != nil {
			logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
			return
		}
		if room.Name == "" {
			logger.Logger.Info("请求参数错误", "api", c.Request.URL.Path)
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
			return
		}

		if room.DisplayName == "" {
			room.DisplayName = room.Name
		}

		if errCreate := h.roomDao.Create(&room); errCreate != nil {
			logger.Logger.Error("创建房间失败", "err", errCreate)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "create success"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
	return
}

// listGet 按分页获取集群信息，并附带对应世界信息
func (h *Handler) listGet(c *gin.Context) {
	type ReqForm struct {
		Partition
		Name string `json:"name" form:"name"`
	}
	var reqForm ReqForm
	var data dao.PaginatedResult[XRoomWorld]
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": data})
		return
	}

	role, _ := c.Get("role")
	var (
		rooms *dao.PaginatedResult[models.Room]
		err   error
	)
	if role.(string) == "admin" {
		// 管理员返回所有房间
		rooms, err = h.roomDao.ListRooms([]string{}, reqForm.Name, reqForm.Page, reqForm.PageSize)
		if err != nil {
			logger.Logger.Error("查询数据库失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": data})
			return
		}
	} else {
		username, _ := c.Get("username")
		user, err := h.userDao.GetUserByUsername(username.(string))
		if err != nil {
			logger.Logger.Error("查询数据库失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": data})
			return
		}
		if user.Rooms == "" {
			data.Page = reqForm.Page
			data.PageSize = reqForm.PageSize
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
			return
		}
		// 非管理员返回有权限的房间 user.Rooms like "dst1,dst2,dst89"
		roomNames := strings.Split(user.Rooms, ",")
		rooms, err = h.roomDao.ListRooms(roomNames, reqForm.Name, reqForm.Page, reqForm.PageSize)
		if err != nil {
			logger.Logger.Error("查询数据库失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": data})
			return
		}

	}

	data.Page = rooms.Page
	data.PageSize = rooms.PageSize
	data.TotalCount = rooms.TotalCount

	// 为房间加上世界信息
	for _, room := range rooms.Data {
		xRoomWorld := XRoomWorld{
			Room:   room,
			Worlds: []models.World{},
		}
		worlds, errWorld := h.worldDao.GetWorldsByRoomName(room.Name)
		if errWorld != nil {
			logger.Logger.Error("查询数据库失败", "err", errWorld)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": data})
			return
		}
		xRoomWorld.Worlds = worlds.Data
		data.Data = append(data.Data, xRoomWorld)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}
