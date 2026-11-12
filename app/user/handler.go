package user

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/service"

	"github.com/gin-gonic/gin"
)

// Partition 分页请求参数
type Partition struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"pageSize" form:"pageSize"`
}

func (h *Handler) registerGet(c *gin.Context) {
	message.Respond(c, h.svc.Registered())
}

func (h *Handler) registerPost(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeUserBadRequest))
		return
	}

	message.Respond(c, h.svc.Register(user))
}

func (h *Handler) loginPost(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeUserBadRequest))
		return
	}

	if user.Username == "" || user.Password == "" {
		logger.Logger.Infof("请求参数缺失, api: %s", c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeUserBadRequest))
		return
	}

	message.Respond(c, h.svc.Login(user))
}

func (h *Handler) menuGet(c *gin.Context) {
	message.Respond(c, h.svc.Menu(middleware.Actor(c)))
}

func (h *Handler) basePost(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeUserBadRequest))
		return
	}

	message.Respond(c, h.svc.CreateUser(user))
}

func (h *Handler) baseGet(c *gin.Context) {
	message.Respond(c, h.svc.Base(middleware.Actor(c)))
}

func (h *Handler) basePut(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeUserBadRequest))
		return
	}

	message.Respond(c, h.svc.UpdateUser(user))
}

func (h *Handler) baseDelete(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeUserBadRequest))
		return
	}

	message.Respond(c, h.svc.DeleteUser(user))
}

func (h *Handler) userListGet(c *gin.Context) {
	type ReqForm struct {
		Partition
		Q string `json:"q" form:"q"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Result{Code: constant.CodeUserBadRequest, Data: dao.PaginatedResult[models.User]{}})
		return
	}

	message.Respond(c, h.svc.UserList(middleware.Actor(c), reqForm.Q, reqForm.Page, reqForm.PageSize))
}

func (h *Handler) myselfPut(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeUserBadRequest))
		return
	}

	message.Respond(c, h.svc.UpdateMyself(middleware.Actor(c), user))
}

// revokePost 管理员撤销指定用户的所有 token
func (h *Handler) revokePost(c *gin.Context) {
	var req struct {
		Username string `json:"username" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeUserBadRequest))
		return
	}

	message.Respond(c, h.svc.Revoke(req.Username))
}
