package platform

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/service"
	platformsvc "dst-management-platform-api/service/platform"
	"encoding/json"
	nethttp "net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Partition 分页请求参数
type Partition struct {
	Page     int `json:"page" form:"page"`
	PageSize int `json:"pageSize" form:"pageSize"`
}

func (h *Handler) readmeGet(c *gin.Context) {
	message.Respond(c, h.svc.ReadmeGet())
}

func (h *Handler) readmePost(c *gin.Context) {
	message.Respond(c, h.svc.ReadmePost())
}

func (h *Handler) overviewGet(c *gin.Context) {
	message.Respond(c, h.svc.Overview())
}

func (h *Handler) gameVersionGet(c *gin.Context) {
	message.Respond(c, h.svc.GameVersion())
}

// gameInstallStreamGet 以 SSE 推送游戏安装/更新的实时输出。
// 传输层留在 handler，脚本执行、输出清洗与任务状态都在 service 层：
//   - action 为空表示只订阅已有任务（页面刷新后重连），不启动新任务
//   - action=install|update 表示先启动任务再订阅
//   - after 是客户端已收到的输出偏移，用于断线续传，首次订阅传 0
func (h *Handler) gameInstallStreamGet(c *gin.Context) {
	type ReqForm struct {
		Action string `json:"action" form:"action"`
		After  int64  `json:"after" form:"after"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	actor := middleware.Actor(c)
	if reqForm.Action != "" {
		// 启动失败（任务忙、参数非法、脚本缺失）走普通 JSON 响应，前端按 content-type 区分
		if result := h.svc.GameTaskStart(actor, reqForm.Action); result.Code != nethttp.StatusOK {
			message.Respond(c, result)
			return
		}
	}

	ch, result := h.svc.GameTaskStream(c.Request.Context(), actor, reqForm.After)
	if result.Code != nethttp.StatusOK {
		message.Respond(c, result)
		return
	}

	header := c.Writer.Header()
	header.Set("Content-Type", "text/event-stream; charset=utf-8")
	// no-transform 让反向代理不要缓冲或改写流式响应，与日志流、诊断流保持一致
	header.Set("Cache-Control", "no-cache, no-transform")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	c.Writer.WriteHeader(nethttp.StatusOK)
	flusher := c.Writer.(nethttp.Flusher)
	_, _ = c.Writer.WriteString("retry: 3000\n\n")
	flusher.Flush()

	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			h.writeGameTaskEvent(c, flusher, event)
			// done 是最后一条事件，写完就结束响应
			if event.Type == platformsvc.GameTaskEventDone {
				return
			}
		case <-heartbeat.C:
			_, _ = c.Writer.WriteString(": ping\n\n")
			flusher.Flush()
		}
	}
}

// writeGameTaskEvent 把一条任务事件写成 SSE 帧。
// 输出内容必须 JSON 编码成单行：安装日志里有大量单独的 \r（进度条原地刷新），
// 而 SSE 把 CR 当行结束符，直接透传会被拆成错误的帧。
func (h *Handler) writeGameTaskEvent(c *gin.Context, flusher nethttp.Flusher, event platformsvc.GameTaskEvent) {
	payload, err := json.Marshal(event.Data)
	if err != nil {
		logger.Logger.Warnf("序列化安装事件失败: %v", err)
		return
	}

	_, _ = c.Writer.WriteString("id: " + strconv.FormatInt(event.ID, 10) + "\n")
	_, _ = c.Writer.WriteString("event: " + event.Type + "\n")
	_, _ = c.Writer.WriteString("data: " + string(payload) + "\n\n")
	flusher.Flush()
}

func (h *Handler) gameTaskStateGet(c *gin.Context) {
	message.Respond(c, h.svc.GameTaskState())
}

func (h *Handler) osInfoGet(c *gin.Context) {
	message.Respond(c, h.svc.OSInfoGet())
}

func (h *Handler) metricsGet(c *gin.Context) {
	type ReqForm struct {
		TimeRange int `json:"timeRange" form:"timeRange"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.Metrics(reqForm.TimeRange))
}

func (h *Handler) globalSettingsGet(c *gin.Context) {
	message.Respond(c, h.svc.GlobalSettingsGet())
}

func (h *Handler) globalSettingsPost(c *gin.Context) {
	var reqForm models.GlobalSetting
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.GlobalSettingsPost(middleware.Actor(c), reqForm))
}

func (h *Handler) screenRunningGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.ScreenRunning(middleware.Actor(c), reqForm.RoomID))
}

func (h *Handler) screenKillPost(c *gin.Context) {
	type ReqForm struct {
		ScreenName string `json:"screenName"`
	}

	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.KillScreen(middleware.Actor(c), reqForm.ScreenName))
}

func (h *Handler) webhookTestPost(c *gin.Context) {
	var reqForm struct {
		URL    string `json:"url" binding:"required"`
		Secret string `json:"secret"`
	}
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.WebhookTest(middleware.Actor(c), reqForm.URL, reqForm.Secret))
}

func (h *Handler) webhookEventsGet(c *gin.Context) {
	message.Respond(c, h.svc.WebhookEvents())
}

func (h *Handler) pluginListGet(c *gin.Context) {
	type ReqForm struct {
		Partition
		Q string `json:"q" form:"q"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.PluginList(reqForm.Page, reqForm.PageSize, reqForm.Q))
}

func (h *Handler) pluginInstallPost(c *gin.Context) {
	var reqForm struct {
		Name       string `json:"name" binding:"required"`
		Proxy      string `json:"proxy"`
		ImageParse *bool  `json:"imageParse"`
	}
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.PluginInstall(middleware.Actor(c), reqForm.Name, reqForm.Proxy, reqForm.ImageParse))
}

func (h *Handler) pluginActionPost(c *gin.Context) {
	var reqForm struct {
		Name string `json:"name" binding:"required"`
		Type string `json:"type"`
	}
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.PluginAction(middleware.Actor(c), reqForm.Name, reqForm.Type))
}

func (h *Handler) pluginStatusGet(c *gin.Context) {
	var reqForm struct {
		Name string `json:"name" form:"name" binding:"required"`
	}
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodePlatformBadRequest))
		return
	}

	message.Respond(c, h.svc.PluginStatus(reqForm.Name))
}
