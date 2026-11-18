package diagnosis

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/service"
	diagnosissvc "dst-management-platform-api/service/diagnosis"
	"encoding/json"
	nethttp "net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func (h *Handler) startPost(c *gin.Context) {
	var reqForm diagnosissvc.StartInput
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeDiagnosisBadRequest))
		return
	}
	message.Respond(c, h.svc.Start(middleware.Actor(c), reqForm))
}

func (h *Handler) getGet(c *gin.Context) {
	message.Respond(c, h.svc.Get(middleware.Actor(c), c.Param("id")))
}

func (h *Handler) cancelDelete(c *gin.Context) {
	message.Respond(c, h.svc.Cancel(middleware.Actor(c), c.Param("id")))
}

func (h *Handler) eventsGet(c *gin.Context) {
	h.stream(c, false)
}

func (h *Handler) applyPost(c *gin.Context) {
	id := c.Param("id")
	var reqForm diagnosissvc.ApplyInput
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeDiagnosisBadRequest))
		return
	}
	message.Respond(c, h.svc.Apply(middleware.Actor(c), id, reqForm))
}

func (h *Handler) applyEventsGet(c *gin.Context) {
	h.stream(c, true)
}

func (h *Handler) historyGet(c *gin.Context) {
	type ReqForm struct {
		Page     int    `json:"page" form:"page"`
		PageSize int    `json:"pageSize" form:"pageSize"`
		Scope    string `json:"scope" form:"scope"`
		Status   string `json:"status" form:"status"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		message.Respond(c, service.Fail(constant.CodeDiagnosisBadRequest))
		return
	}
	message.Respond(c, h.svc.History(middleware.Actor(c), reqForm.Scope, reqForm.Status, reqForm.Page, reqForm.PageSize))
}

func (h *Handler) historyDetailGet(c *gin.Context) {
	message.Respond(c, h.svc.HistoryDetail(middleware.Actor(c), c.Param("id")))
}

func (h *Handler) settingGet(c *gin.Context) {
	message.Respond(c, h.svc.AISettingGet(middleware.Actor(c)))
}

func (h *Handler) settingPut(c *gin.Context) {
	var reqForm models.DiagnosisAISetting
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		message.Respond(c, service.Fail(constant.CodeDiagnosisBadRequest))
		return
	}
	message.Respond(c, h.svc.AISettingPut(middleware.Actor(c), reqForm))
}

func (h *Handler) stream(c *gin.Context, apply bool) {
	id := c.Param("id")
	after := parseLastEventID(c)
	actor := middleware.Actor(c)

	var (
		ch     <-chan diagnosissvc.SSEEvent
		result service.Result
	)
	if apply {
		ch, result = h.svc.ApplyStream(c.Request.Context(), actor, id, after)
	} else {
		ch, result = h.svc.Stream(c.Request.Context(), actor, id, after)
	}
	if result.Code != nethttp.StatusOK {
		message.Respond(c, result)
		return
	}

	header := c.Writer.Header()
	header.Set("Content-Type", "text/event-stream; charset=utf-8")
	header.Set("Cache-Control", "no-cache")
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
			h.writeEvent(c, flusher, event)
		case <-heartbeat.C:
			_, _ = c.Writer.WriteString(": ping\n\n")
			flusher.Flush()
		}
	}
}

func (h *Handler) writeEvent(c *gin.Context, flusher nethttp.Flusher, event diagnosissvc.SSEEvent) {
	data := event.Data
	if event.Type == diagnosissvc.EventError {
		if value, ok := data.(diagnosissvc.ErrorEventData); ok && value.Message == "" {
			value.Message = localizeCode(c, value.Code)
			data = value
		}
	}
	payload, err := json.Marshal(data)
	if err != nil {
		logger.Logger.Warnf("序列化 SSE 事件失败: %v", err)
		return
	}
	_, _ = c.Writer.WriteString("id: " + strconv.FormatInt(event.ID, 10) + "\n")
	_, _ = c.Writer.WriteString("event: " + event.Type + "\n")
	_, _ = c.Writer.WriteString("data: " + string(payload) + "\n\n")
	flusher.Flush()
}

func localizeCode(c *gin.Context, code int) string {
	entry, ok := constant.LookupCode(code)
	if !ok {
		return message.Get(c, "unknown error")
	}
	return message.Get(c, entry.TextKey)
}

func parseLastEventID(c *gin.Context) int64 {
	raw := strings.TrimSpace(c.GetHeader("Last-Event-ID"))
	if raw == "" {
		raw = strings.TrimSpace(c.Query("lastEventID"))
	}
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value < 0 {
		return 0
	}
	return value
}
