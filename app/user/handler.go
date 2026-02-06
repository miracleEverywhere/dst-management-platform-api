package user

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) registerGet(c *gin.Context) {
	var registered bool

	num, err := h.userDao.Count(nil)
	if err != nil {
		registered = false
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "fail", "data": registered})
		return
	}

	if num != 0 {
		registered = false
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": registered})
		return
	} else {
		registered = true
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": registered})
		return
	}
}

func (h *Handler) registerPost(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	logger.Logger.Debug(utils.StructToFlatString(user))

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

func (h *Handler) loginPost(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	logger.Logger.Debug(utils.StructToFlatString(user))

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

	token, err := utils.GenerateJWT(*dbUser, []byte(db.JwtSecret), utils.JwtExpirationHours)
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
			tools,
			logs,
			upload,
			install,
			platform,
		}
	} else {
		toolsLess := tools
		toolsLess.Links = []menuItem{
			tools.Links[0],
			tools.Links[1],
			tools.Links[2],
			tools.Links[4],
		}
		logsLess := logs
		logsLess.Links = []menuItem{
			logs.Links[0],
			logs.Links[1],
			logs.Links[2],
		}

		response.Data = []menuItem{
			rooms,
			dashboard,
			game,
			toolsLess,
			logsLess,
			upload,
		}
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) basePost(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	logger.Logger.Debug(utils.StructToFlatString(user))

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

func (h *Handler) basePut(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	logger.Logger.Debug(utils.StructToFlatString(user))

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

	user.Password = dbUser.Password
	err = h.userDao.UpdateUser(&user)
	if err != nil {
		logger.Logger.Error("更新数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "update fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "update success"), "data": nil})
}

func (h *Handler) baseDelete(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	logger.Logger.Debug(utils.StructToFlatString(user))

	// 用户数小于等于1时，禁止删除
	num, err := h.userDao.Count(nil)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	if num <= 1 {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "delete all users"), "data": nil})
		return
	}

	// 查询用户是否存在
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

	// 执行删除
	err = h.userDao.Delete(dbUser)
	if err != nil {
		logger.Logger.Error("更新数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "delete fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "delete success"), "data": nil})
}

func (h *Handler) userListGet(c *gin.Context) {
	type ReqForm struct {
		Partition
		Q string `json:"q" form:"q"`
	}
	var (
		reqForm ReqForm
		data    dao.PaginatedResult[models.User]
	)
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": data})
		return
	}

	role, _ := c.Get("role")
	if role.(string) != "admin" {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": data})
		return
	}

	users, err := h.userDao.ListUsers(reqForm.Q, reqForm.Page, reqForm.PageSize)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": data})
		return
	}

	data.Data = []models.User{} // 防止Data为nil
	for _, user := range users.Data {
		user.Password = ""
		data.Data = append(data.Data, user)
	}

	data.Page = users.Page
	data.PageSize = users.PageSize
	data.TotalCount = users.TotalCount

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}

func (h *Handler) myselfPut(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	logger.Logger.Debug(utils.StructToFlatString(user))

	dbUser, err := h.userDao.GetUserByUsername(user.Username)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	dbUser.Password = user.Password
	dbUser.Nickname = user.Nickname
	dbUser.Avatar = user.Avatar

	err = h.userDao.UpdateUser(dbUser)
	if err != nil {
		logger.Logger.Error("更新数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "update fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "myself update success"), "data": nil})
}
