package user

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Handler struct {
	userDao   *dao.UserDAO
	systemDao *dao.SystemDAO
}

func NewUserHandler(userDao *dao.UserDAO, systemDao *dao.SystemDAO) *Handler {
	return &Handler{
		userDao:   userDao,
		systemDao: systemDao,
	}
}

func (h *Handler) registerPost(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message(c, "bad request"), "data": nil})
		return
	}

	num, err := h.userDao.Count(nil)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message(c, "register fail"), "data": nil})
		return
	}

	if num != 0 {
		logger.Logger.Info("创建用户失败，用户已存在", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message(c, "user exist"), "data": nil})
		return
	}

	user.Disabled = false
	user.Role = "admin"

	if errCreate := h.userDao.Create(&user); errCreate != nil {
		logger.Logger.Error("创建用户失败", "err", errCreate)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message(c, "register fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message(c, "register success"), "data": nil})
	return
}

func (h *Handler) loginPost(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message(c, "bad request"), "data": nil})
		return
	}
	if user.Username == "" || user.Password == "" {
		logger.Logger.Info("请求参数缺失", "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message(c, "bad request"), "data": nil})
		return
	}

	dbUser, err := h.userDao.GetUserByUsername(user.Username)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message(c, "login fail"), "data": nil})
		return
	}

	if dbUser.Username == "" {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message(c, "user not exist"), "data": nil})
		return
	}

	if dbUser.Disabled {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message(c, "disabled"), "data": nil})
		return
	}

	if dbUser.Password != user.Password {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message(c, "wrong password"), "data": nil})
		return
	}

	token, err := utils.GenerateJWT(*dbUser, []byte(db.JwtSecret), 24)
	if err != nil {
		logger.Logger.Error("生成jwt失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message(c, "login fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message(c, "login success"), "data": token})
}

func (h *Handler) menuGet(c *gin.Context) {
	role, _ := c.Get("role")
	type menuItem struct {
		ID        int        `json:"id"`
		Type      string     `json:"type"`
		Section   string     `json:"section"`
		Title     string     `json:"title"`
		To        string     `json:"to"`
		Component string     `json:"component"`
		Icon      string     `json:"icon"`
		Links     []menuItem `json:"links"`
	}
	type Response struct {
		Code    int        `json:"code"`
		Message string     `json:"message"`
		Data    []menuItem `json:"data"`
	}
	dashboard := menuItem{
		ID:        1,
		Type:      "link",
		Section:   "",
		Title:     "dashboard",
		To:        "/dashboard",
		Component: "dashboard/index",
		Icon:      "ri-table-alt-line",
		Links:     nil,
	}

	response := Response{
		Code:    200,
		Message: "success",
		Data:    nil,
	}

	if role.(string) == "admin" {
		response.Data = []menuItem{
			dashboard,
		}
	} else {
		response.Data = []menuItem{
			dashboard,
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) userInfo(c *gin.Context) {
	username, _ := c.Get("username")
	dbUser, err := h.userDao.GetUserByUsername(username.(string))
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message(c, "login fail"), "data": nil})
		return
	}
	dbUser.Password = ""

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": dbUser})
}
