package room

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Handler struct {
	roomDao *dao.RoomDAO
	userDao *dao.UserDAO
}

func NewRoomHandler(roomDao *dao.RoomDAO, userDao *dao.UserDAO) *Handler {
	return &Handler{
		roomDao: roomDao,
		userDao: userDao,
	}
}

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
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "create fail"), "data": nil})
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
	}
}
