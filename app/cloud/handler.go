package cloud

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/middleware"
	"dst-management-platform-api/service"
	"fmt"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type cloudLoginReq struct {
	Username string `json:"username"`
	Identity string `json:"identity"`
	Password string `json:"password"`
}

func (r cloudLoginReq) identity() string {
	identity := strings.TrimSpace(r.Identity)
	if identity == "" {
		identity = strings.TrimSpace(r.Username)
	}

	return identity
}

// loginPost 云账号登录。
// 数据库无用户时创建首个本地管理员；已有用户时必须先绑定云账号。
// 云服务网络错误/超时/5xx 时回退到本地账号校验。
func (h *Handler) loginPost(c *gin.Context) {
	var req cloudLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		message.Respond(c, service.Fail(constant.CodeCloudBadRequest))
		return
	}

	message.Respond(c, h.svc.Login(c.Request.Context(), middleware.Actor(c), req.identity(), req.Password))
}

type linkReq struct {
	Username string `json:"username"`
	Identity string `json:"identity"`
	Password string `json:"password"`
}

func (r linkReq) identity() string {
	identity := strings.TrimSpace(r.Identity)
	if identity == "" {
		identity = strings.TrimSpace(r.Username)
	}

	return identity
}

// linkPost 把当前本地账号绑定到云账号。绑定不修改本地用户名和本地密码。
func (h *Handler) linkPost(c *gin.Context) {
	var req linkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		message.Respond(c, service.Fail(constant.CodeCloudBadRequest))
		return
	}

	message.Respond(c, h.svc.Link(c.Request.Context(), middleware.Actor(c), req.identity(), req.Password))
}

type unlinkReq struct {
	Password string `json:"password"`
}

// unlinkPost 解除云账号绑定。需要本地密码或云账号密码确认，防止会话被劫持后直接解绑。
func (h *Handler) unlinkPost(c *gin.Context) {
	var req unlinkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		message.Respond(c, service.Fail(constant.CodeCloudBadRequest))
		return
	}

	message.Respond(c, h.svc.Unlink(middleware.Actor(c), req.Password))
}

// statusGet 返回当前本地账号的云绑定状态。
func (h *Handler) statusGet(c *gin.Context) {
	message.Respond(c, h.svc.Status(middleware.Actor(c)))
}

func (h *Handler) favoritesGet(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	message.Respond(c, h.svc.Favorites(c.Request.Context(), middleware.Actor(c), page, pageSize, c.Query("keyword")))
}

func (h *Handler) tokensGet(c *gin.Context) {
	message.Respond(c, h.svc.Tokens(c.Request.Context(), middleware.Actor(c)))
}

func (h *Handler) gameEntriesGet(c *gin.Context, kind string) {
	message.Respond(c, h.svc.GameEntries(c.Request.Context(), middleware.Actor(c), kind))
}

func (h *Handler) gameAdminsGet(c *gin.Context) {
	h.gameEntriesGet(c, "admin")
}

func (h *Handler) gameSlotsGet(c *gin.Context) {
	h.gameEntriesGet(c, "slot")
}

// ===== 云同步 =====

type syncReq struct {
	Direction string `json:"direction"`
}

func (h *Handler) syncPost(c *gin.Context) {
	var req syncReq
	if err := c.ShouldBindJSON(&req); err != nil {
		message.Respond(c, service.Fail(constant.CodeCloudBadRequest))
		return
	}

	message.Respond(c, h.svc.Sync(c.Request.Context(), middleware.Actor(c), req.Direction))
}

// ===== 云存档 =====

type archiveUploadReq struct {
	RoomID int `json:"roomID"`
}

func (h *Handler) archiveUploadPost(c *gin.Context) {
	var req archiveUploadReq
	if err := c.ShouldBindJSON(&req); err != nil || req.RoomID == 0 {
		message.Respond(c, service.Fail(constant.CodeCloudBadRequest))
		return
	}

	message.Respond(c, h.svc.ArchiveUpload(c.Request.Context(), middleware.Actor(c), req.RoomID))
}

func (h *Handler) archivesGet(c *gin.Context) {
	message.Respond(c, h.svc.Archives(c.Request.Context(), middleware.Actor(c)))
}

func (h *Handler) archiveDownloadGet(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))

	body, fileName, size, result := h.svc.ArchiveDownload(c.Request.Context(), middleware.Actor(c), id)
	if body != nil {
		defer body.Close()
	}
	if result.Code != 200 {
		message.Respond(c, result)
		return
	}

	c.Header("Content-Disposition", attachmentHeader(fileName))
	c.Header("X-Content-Type-Options", "nosniff")
	if size <= 0 {
		size = -1
	}
	c.DataFromReader(200, size, "application/zip", body, nil)
}

// archiveImportPost 下载云存档并导入为一个新房间。
// 只有拥有创建房间权限（管理员或 roomCreation=true）的用户可以执行。
func (h *Handler) archiveImportPost(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	message.Respond(c, h.svc.ArchiveImport(c.Request.Context(), middleware.Actor(c), id))
}

// attachmentHeader 生成兼容中文文件名的下载头。
func attachmentHeader(fileName string) string {
	name := strings.TrimSpace(fileName)
	if name == "" {
		name = "archive.zip"
	}
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, `"`, "_")
	name = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return '_'
		}
		return r
	}, name)
	if filepath.Ext(name) == "" {
		name += ".zip"
	}
	fallback := make([]rune, 0, len(name))
	for _, r := range name {
		if r < 0x20 || r > 0x7e || r == '"' || r == '\\' {
			fallback = append(fallback, '_')
		} else {
			fallback = append(fallback, r)
		}
	}

	return fmt.Sprintf(`attachment; filename="%s"; filename*=UTF-8''%s`, string(fallback), url.PathEscape(name))
}
