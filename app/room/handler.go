package room

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/service"
	roomsvc "dst-management-platform-api/service/room"
	"dst-management-platform-api/utils"
	"fmt"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func (h *Handler) roomPost(c *gin.Context) {
	// 原实现先校验房间创建权限（含数据库异常）再解析请求体，保持该顺序
	if failed := h.svc.CreateGuard(middleware.Actor(c)); failed != nil {
		message.Respond(c, *failed)
		return
	}

	var reqForm roomsvc.XRoomTotalInfo
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeRoomBadRequest))
		return
	}

	message.Respond(c, h.svc.Create(middleware.Actor(c), reqForm))
}

// roomPut 修改房间
func (h *Handler) roomPut(c *gin.Context) {
	var reqForm roomsvc.XRoomTotalInfo
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeRoomBadRequest))
		return
	}

	message.Respond(c, h.svc.Update(middleware.Actor(c), reqForm))
}

// listGet 按分页获取集群信息，并附带对应世界信息
func (h *Handler) listGet(c *gin.Context) {
	type ReqForm struct {
		roomsvc.Partition
		GameName string `json:"gameName" form:"gameName"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		// 原实现在绑定失败时返回零值分页结构
		message.Respond(c, service.Result{Code: constant.CodeRoomBadRequest, Data: roomsvc.RoomWorldPage{}})
		return
	}

	message.Respond(c, h.svc.List(middleware.Actor(c), reqForm.GameName, reqForm.Page, reqForm.PageSize))
}

// roomGet 返回房间、世界、房间设置等所有信息
func (h *Handler) roomGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"id" form:"id"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeRoomBadRequest))
		return
	}

	message.Respond(c, h.svc.Get(middleware.Actor(c), reqForm.RoomID))
}

// roomMetricsGet 获取房间内各世界的资源监控数据，数据来源于内存缓存
func (h *Handler) roomMetricsGet(c *gin.Context) {
	type ReqForm struct {
		RoomID    int `json:"roomID" form:"roomID"`
		TimeRange int `json:"timeRange" form:"timeRange"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeRoomBadRequest))
		return
	}

	message.Respond(c, h.svc.Metrics(middleware.Actor(c), reqForm.RoomID, reqForm.TimeRange))
}

// factorGet 前端自动分配端口
func (h *Handler) factorGet(c *gin.Context) {
	message.Respond(c, h.svc.Factor())
}

// allRoomBasicGet 获取room基本信息 name和id
func (h *Handler) allRoomBasicGet(c *gin.Context) {
	message.Respond(c, h.svc.AllBasic())
}

func (h *Handler) allRoomPermittedBasicGet(c *gin.Context) {
	message.Respond(c, h.svc.PermittedBasic(middleware.Actor(c)))
}

func (h *Handler) roomWorldsGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeRoomBadRequest))
		return
	}

	message.Respond(c, h.svc.Worlds(middleware.Actor(c), reqForm.RoomID))
}

func (h *Handler) uploadPost(c *gin.Context) {
	roomIDStr := c.PostForm("roomID")

	// 权限校验必须先于文件读取，保持与原始实现一致的响应顺序
	roomID, newRoom, failed := h.svc.UploadTarget(middleware.Actor(c), roomIDStr)
	if failed != nil {
		message.Respond(c, *failed)
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeRoomBadRequest))
		return
	}

	currentTS := utils.GetTimestamp()

	// 创建上传文件保存目录
	uploadPath := fmt.Sprintf("%s/upload/%d", constant.DmpFiles, currentTS)
	if err = utils.EnsureDirExists(uploadPath); err != nil {
		logger.Logger.Errorf("创建上传目录失败, err: %v", err)
		message.Respond(c, service.Fail(constant.CodeRoomUploadSaveFail))
		return
	}

	defer func() {
		if errRemove := utils.RemoveDir(uploadPath); errRemove != nil {
			logger.Logger.Errorf("清理上传文件失败, err: %v", errRemove)
		}
	}()

	//保存上传的文件
	unzipPath := fmt.Sprintf("%s/", uploadPath)
	filename := "room.zip"
	savePath := filepath.Join(uploadPath, filename)
	if err = c.SaveUploadedFile(file, savePath); err != nil {
		logger.Logger.Errorf("文件保存失败, err: %v", err)
		message.Respond(c, service.Fail(constant.CodeRoomUploadSaveFail))
		return
	}

	message.Respond(c, h.svc.UploadImport(middleware.Actor(c), savePath, unzipPath, roomID, newRoom))
}

func (h *Handler) deactivatePost(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeRoomBadRequest))
		return
	}

	message.Respond(c, h.svc.Deactivate(middleware.Actor(c), reqForm.RoomID))
}

func (h *Handler) activatePost(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeRoomBadRequest))
		return
	}

	message.Respond(c, h.svc.Activate(middleware.Actor(c), reqForm.RoomID))
}

func (h *Handler) roomDelete(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeRoomBadRequest))
		return
	}

	message.Respond(c, h.svc.Delete(middleware.Actor(c), reqForm.RoomID))
}

func (h *Handler) webhookTestPost(c *gin.Context) {
	var reqForm struct {
		URL    string `json:"url" binding:"required"`
		Secret string `json:"secret"`
	}
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeRoomBadRequest))
		return
	}

	message.Respond(c, h.svc.WebhookTest(middleware.Actor(c), reqForm.URL, reqForm.Secret))
}

func (h *Handler) webhookEventsGet(c *gin.Context) {
	message.Respond(c, h.svc.WebhookEvents())
}

func (h *Handler) worldCustomStartupCmdPut(c *gin.Context) {
	var reqForm struct {
		RoomID int                             `json:"roomID" form:"roomID"`
		Worlds []roomsvc.WorldCustomStartupCmd `json:"worldCustomStartupCmd" form:"worldCustomStartupCmd"`
	}
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeRoomBadRequest))
		return
	}

	message.Respond(c, h.svc.CustomStartupCmdPut(middleware.Actor(c), reqForm.RoomID, reqForm.Worlds))
}

// favoritePost 收藏已有房间的 room/world/roomSetting 配置
func (h *Handler) favoritePost(c *gin.Context) {
	type ReqForm struct {
		RoomID int    `json:"roomID" binding:"required"`
		Remark string `json:"remark"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeRoomBadRequest))
		return
	}

	message.Respond(c, h.svc.FavoriteCreate(middleware.Actor(c), reqForm.RoomID, reqForm.Remark))
}

// favoriteGet 分页查询当前用户的 dst 收藏
func (h *Handler) favoriteGet(c *gin.Context) {
	var req roomsvc.Partition
	if err := c.ShouldBindQuery(&req); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeRoomBadRequest))
		return
	}

	message.Respond(c, h.svc.FavoriteList(middleware.Actor(c), req.Page, req.PageSize))
}

// favoriteDetailGet 查询当前用户的单条收藏
func (h *Handler) favoriteDetailGet(c *gin.Context) {
	message.Respond(c, h.svc.FavoriteDetail(middleware.Actor(c), c.Param("id")))
}

// favoritePut 修改收藏备注
func (h *Handler) favoritePut(c *gin.Context) {
	type ReqForm struct {
		ID     string  `json:"id" binding:"required"`
		Remark *string `json:"remark" binding:"required"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeRoomBadRequest))
		return
	}

	message.Respond(c, h.svc.FavoriteUpdate(middleware.Actor(c), reqForm.ID, reqForm.Remark))
}

// favoriteDelete 删除当前用户的收藏
func (h *Handler) favoriteDelete(c *gin.Context) {
	type ReqForm struct {
		ID string `json:"id" binding:"required"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeRoomBadRequest))
		return
	}

	message.Respond(c, h.svc.FavoriteDelete(middleware.Actor(c), reqForm.ID))
}

// favoriteApplyPost 使用收藏创建新房间
func (h *Handler) favoriteApplyPost(c *gin.Context) {
	type ReqForm struct {
		ID       string `json:"id" binding:"required"`
		GameName string `json:"gameName"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeRoomBadRequest))
		return
	}

	message.Respond(c, h.svc.FavoriteApply(middleware.Actor(c), reqForm.ID, reqForm.GameName))
}
