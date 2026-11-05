package cloud

import (
	"dst-management-platform-api/app/room"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	userDao      *dao.UserDAO
	cloudLinkDao *dao.CloudLinkDAO
	cloudMapDao  *dao.CloudFavoriteMapDAO
	favoriteDao  *dao.FavoriteDAO
	client       *Client
	roomHandler  *room.Handler
}

func NewHandler(userDao *dao.UserDAO, cloudLinkDao *dao.CloudLinkDAO, cloudMapDao *dao.CloudFavoriteMapDAO, favoriteDao *dao.FavoriteDAO, roomHandler *room.Handler) *Handler {
	return &Handler{
		userDao:      userDao,
		cloudLinkDao: cloudLinkDao,
		cloudMapDao:  cloudMapDao,
		favoriteDao:  favoriteDao,
		client:       NewClient(),
		roomHandler:  roomHandler,
	}
}

func ok(c *gin.Context, msg string, data any) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": msg, "data": data})
}

func fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, gin.H{"code": code, "message": msg, "data": nil})
}

// failCloud 把云客户端错误转换为 DMP 业务响应，HTTP 状态保持 200。
func failCloud(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrRelink):
		fail(c, 201, message.Get(c, "cloud relink required"))
	case IsUnavailable(err):
		logger.Logger.Warnf("云服务不可用, api: %s, err: %v", c.Request.URL.Path, err)
		fail(c, 201, message.Get(c, "cloud unavailable"))
	default:
		if msg := MessageOf(err); msg != "" {
			logger.Logger.Warnf("云服务请求失败, api: %s, err: %v", c.Request.URL.Path, err)
			fail(c, 201, msg)
			return
		}
		logger.Logger.Errorf("云服务请求失败, api: %s, err: %v", c.Request.URL.Path, err)
		fail(c, 201, message.Get(c, "cloud unavailable"))
	}
}

func nowSeconds() int64 {
	return time.Now().Unix()
}

func currentUsername(c *gin.Context) string {
	return c.GetString("username")
}

func (h *Handler) getLink(username string) (*models.CloudLink, error) {
	return h.cloudLinkDao.GetByUsername(username)
}

func (h *Handler) saveLink(link *models.CloudLink) error {
	link.UpdatedAt = nowSeconds()
	if link.DeviceID == "" {
		link.DeviceID = uuid.NewString()
	}
	if link.LinkedAt == 0 {
		link.LinkedAt = nowSeconds()
	}

	return h.cloudLinkDao.SaveLink(link)
}

func (h *Handler) getLocalUser(username string) (*models.User, error) {
	return h.userDao.GetUserByUsername(username)
}

// applyAuthResult 把云登录 / 云绑定返回的账号信息写入本地绑定表。
func (h *Handler) applyAuthResult(link *models.CloudLink, result *AuthResult, password string) error {
	plan := result.Account.Plan
	if plan == "" {
		plan = result.Plan.Code
	}
	if plan == "" {
		plan = "basic"
	}
	favoriteQuota, tokenQuota, adminQuota, slotQuota, archiveQuota := QuotasForPlan(plan)

	username := result.User.Username
	if username == "" {
		username = result.Account.Username
	}
	email := result.User.Email
	if email == "" {
		email = result.Account.Email
	}

	passwordHash := link.CloudPassword
	if password != "" {
		hash, _, err := hashCloudPassword(password)
		if err != nil {
			return err
		}
		passwordHash = hash
	}

	link.CloudAccountID = result.Account.ID
	link.CloudUsername = username
	link.CloudEmail = email
	link.CloudPassword = passwordHash
	link.Plan = plan
	link.FavoriteQuota = favoriteQuota
	link.TokenQuota = tokenQuota
	link.GameAdminQuota = adminQuota
	link.GameSlotQuota = slotQuota
	link.ArchiveQuota = archiveQuota
	if link.ID == 0 {
		link.AutoSync = true
	}
	if link.LinkedAt == 0 {
		link.LinkedAt = nowSeconds()
	}
	link.UpdatedAt = nowSeconds()
	if link.DeviceID == "" {
		link.DeviceID = uuid.NewString()
	}

	return applyTokenPair(link, result.Tokens)
}

// applyMeResult 刷新套餐与配额，不修改 token。
func applyMeResult(link *models.CloudLink, result *MeResult) {
	plan := result.Account.Plan
	if plan == "" {
		plan = result.Plan.Code
	}
	if plan == "" {
		plan = "basic"
	}
	favoriteQuota, tokenQuota, adminQuota, slotQuota, archiveQuota := QuotasForPlan(plan)

	link.Plan = plan
	link.FavoriteQuota = favoriteQuota
	link.TokenQuota = tokenQuota
	link.GameAdminQuota = adminQuota
	link.GameSlotQuota = slotQuota
	link.ArchiveQuota = archiveQuota
	link.UpdatedAt = nowSeconds()
}

func (h *Handler) refreshProfile(c *gin.Context, link *models.CloudLink) error {
	me, err := h.client.Me(c.Request.Context(), link)
	if err != nil {
		return err
	}
	applyMeResult(link, me)
	if err = h.saveLink(link); err != nil {
		return err
	}

	return nil
}

// hashCloudPassword 生成云密码哈希，并返回本地 users 表应使用的密码版本。
// bcrypt 只接受 72 字节以内的密码，超长密码退化为 SHA-256 + bcrypt。
func hashCloudPassword(password string) (string, string, error) {
	if len([]byte(password)) <= 72 {
		hash, err := utils.GenerateBcryptPassword(password)
		return hash, models.PasswordVersionBcrypt, err
	}
	hash, err := utils.GenerateBcryptSha256Password(password)

	return hash, models.PasswordVersionBcryptSha256, err
}

func validateCloudPassword(password, hash string) bool {
	return utils.ValidatePassword(password, hash) || utils.ValidateBcryptSha256Password(password, hash)
}

func (h *Handler) issueLocalToken(c *gin.Context, user *models.User) (string, error) {
	if user.Disabled {
		return "", errors.New("disabled")
	}

	token, err := utils.GenerateJWT(*user, []byte(cache.JwtSecret), constant.JwtExpirationHours)
	if err != nil {
		return "", err
	}
	cache.SetTokenVersion(user.Username, user.TokenVersion)

	return token, nil
}

func linkView(link *models.CloudLink) gin.H {
	if link == nil || link.ID == 0 {
		return gin.H{"linked": false}
	}

	return gin.H{
		"linked":         true,
		"username":       link.Username,
		"cloudAccountId": link.CloudAccountID,
		"cloudUsername":  link.CloudUsername,
		"cloudEmail":     link.CloudEmail,
		"plan":           link.Plan,
		"planName":       PlanName(link.Plan),
		"favoriteQuota":  link.FavoriteQuota,
		"tokenQuota":     link.TokenQuota,
		"gameAdminQuota": link.GameAdminQuota,
		"gameSlotQuota":  link.GameSlotQuota,
		"archiveQuota":   link.ArchiveQuota,
		"lastSyncAt":     link.LastSyncAt,
		"autoSync":       link.AutoSync,
		"linkedAt":       link.LinkedAt,
		"updatedAt":      link.UpdatedAt,
	}
}

type syncSummaryView struct {
	Direction     string     `json:"direction"`
	ServerTime    int64      `json:"serverTime"`
	Uploaded      int        `json:"uploaded"`
	Updated       int        `json:"updated"`
	Deleted       int        `json:"deleted"`
	Skipped       int        `json:"skipped"`
	QuotaExceeded int        `json:"quotaExceeded"`
	Conflicts     int        `json:"conflicts"`
	Downloaded    int        `json:"downloaded"`
	FavoriteQuota int        `json:"favoriteQuota"`
	FavoriteUsed  int        `json:"favoriteUsed"`
	SkippedItems  []SyncSkip `json:"skippedItems"`
	LocalCount    int64      `json:"localCount"`
}

func newSyncSummaryView(direction string, result *SyncResult, localCount int64) syncSummaryView {
	skipped := result.Skipped
	if skipped == nil {
		skipped = []SyncSkip{}
	}

	return syncSummaryView{
		Direction:     direction,
		ServerTime:    result.ServerTime,
		Uploaded:      result.Summary.Uploaded,
		Updated:       result.Summary.Updated,
		Deleted:       result.Summary.Deleted,
		Skipped:       result.Summary.Skipped,
		QuotaExceeded: result.Summary.QuotaExceeded,
		Conflicts:     result.Summary.Conflicts,
		Downloaded:    result.Summary.Downloaded,
		FavoriteQuota: result.FavoriteQuota.Limit,
		FavoriteUsed:  result.FavoriteQuota.Used,
		SkippedItems:  skipped,
		LocalCount:    localCount,
	}
}

func (h *Handler) localFavoriteCount(username string) int64 {
	count, err := h.favoriteDao.Count("username = ? AND type = ? AND deleted_at = 0", username, "dst")
	if err != nil {
		logger.Logger.Errorf("统计本地收藏失败, username: %s, err: %v", username, err)
		return 0
	}

	return count
}

func (h *Handler) canCreateRoom(c *gin.Context) bool {
	if c.GetString("role") == "admin" {
		return true
	}
	user, err := h.getLocalUser(currentUsername(c))
	if err != nil || user.Username == "" {
		return false
	}

	return user.RoomCreation
}

func (h *Handler) hasRoomPermission(c *gin.Context, roomID int) bool {
	if c.GetString("role") == "admin" {
		return true
	}
	user, err := h.getLocalUser(currentUsername(c))
	if err != nil || user.Username == "" {
		return false
	}
	for _, id := range strings.Split(user.Rooms, ",") {
		if id == fmt.Sprintf("%d", roomID) {
			return true
		}
	}

	return false
}

var cloudIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func validCloudID(id string) bool {
	return cloudIDPattern.MatchString(id)
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

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}

	return string(runes[:max])
}
