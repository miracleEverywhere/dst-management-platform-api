package user

import (
    "dst-management-platform-api/dao"
    "dst-management-platform-api/db"
    "dst-management-platform-api/logger"
    "dst-management-platform-api/models"
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
