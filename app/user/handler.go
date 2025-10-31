package user

import (
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func (h *Handler) registerPost(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	num, err := h.userDao.Count(nil)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	if num != 0 {
		logger.Logger.Info("创建用户失败，用户已存在", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "user exist"), "data": nil})
		return
	}

	// 注册的用户默认拥有最高权限
	user.Disabled = false
	user.Role = "admin"

	if errCreate := h.userDao.Create(&user); errCreate != nil {
		logger.Logger.Error("创建用户失败", "err", errCreate)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "register success"), "data": nil})
	return
}

func (h *Handler) basePost(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	dbUser, err := h.userDao.GetUserByUsername(user.Username)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	if dbUser.Username != "" {
		logger.Logger.Info("创建用户失败，用户已存在", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "user exist"), "data": nil})
		return
	}

	if errCreate := h.userDao.Create(&user); errCreate != nil {
		logger.Logger.Error("创建用户失败", "err", errCreate)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "create success"), "data": nil})
	return
}

func (h *Handler) loginPost(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	if user.Username == "" || user.Password == "" {
		logger.Logger.Info("请求参数缺失", "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	dbUser, err := h.userDao.GetUserByUsername(user.Username)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	if dbUser.Username == "" {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "user not exist"), "data": nil})
		return
	}

	if dbUser.Disabled {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "disabled"), "data": nil})
		return
	}

	if dbUser.Password != user.Password {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "wrong password"), "data": nil})
		return
	}

	token, err := utils.GenerateJWT(*dbUser, []byte(db.JwtSecret), 24)
	if err != nil {
		logger.Logger.Error("生成jwt失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "login fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "login success"), "data": token})
}

func (h *Handler) menuGet(c *gin.Context) {
	role, _ := c.Get("role")
	type Response struct {
		Code    int        `json:"code"`
		Message string     `json:"message"`
		Data    []menuItem `json:"data"`
	}

	response := Response{
		Code:    200,
		Message: "success",
		Data:    nil,
	}

	if role.(string) == "admin" {
		response.Data = []menuItem{
			rooms,
			dashboard,
			game,
			upload,
			platform,
		}
	} else {
		response.Data = []menuItem{
			rooms,
			dashboard,
			game,
			upload,
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) baseGet(c *gin.Context) {
	username, _ := c.Get("username")
	dbUser, err := h.userDao.GetUserByUsername(username.(string))
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}
	dbUser.Password = ""

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": dbUser})
}
