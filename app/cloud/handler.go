package cloud

import (
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"errors"
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

type cloudLoginView struct {
	Token         string           `json:"token"`
	Username      string           `json:"username"`
	Fallback      bool             `json:"fallback"`
	Plan          string           `json:"plan"`
	FavoriteQuota int              `json:"favoriteQuota"`
	Sync          *syncSummaryView `json:"sync,omitempty"`
	SyncError     string           `json:"syncError,omitempty"`
}

// loginPost 云账号登录。
// 数据库无用户时创建首个本地管理员；已有用户时必须先绑定云账号。
// 云服务网络错误/超时/5xx 时回退到本地账号校验。
func (h *Handler) loginPost(c *gin.Context) {
	var req cloudLoginReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, message.Get(c, "bad request"))
		return
	}
	identity := req.identity()
	if identity == "" || req.Password == "" {
		fail(c, 400, message.Get(c, "bad request"))
		return
	}

	userCount, err := h.userDao.Count(nil)
	if err != nil {
		logger.Logger.Errorf("查询用户数量失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}

	result, err := h.client.Login(c.Request.Context(), identity, req.Password)
	if err != nil {
		if IsUnavailable(err) {
			h.fallbackLocalLogin(c, identity, req.Password)
			return
		}
		failCloud(c, err)
		return
	}

	if userCount == 0 {
		h.bootstrapCloudUser(c, result, req.Password)
		return
	}

	link, err := h.cloudLinkDao.GetByCloudAccountID(result.Account.ID)
	if err != nil {
		logger.Logger.Errorf("查询云账号绑定失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}
	if link.ID == 0 {
		fail(c, 201, message.Get(c, "cloud not linked"))
		return
	}

	user, err := h.getLocalUser(link.Username)
	if err != nil {
		logger.Logger.Errorf("查询本地用户失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}
	if user.Username == "" {
		fail(c, 201, message.Get(c, "cloud local user missing"))
		return
	}
	if user.Disabled {
		fail(c, 201, message.Get(c, "cloud local user disabled"))
		return
	}

	if err = h.applyAuthResult(link, result, req.Password); err != nil {
		logger.Logger.Errorf("保存云账号信息失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}
	if err = h.saveLink(link); err != nil {
		logger.Logger.Errorf("保存云账号绑定失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}

	token, err := h.issueLocalToken(c, user)
	if err != nil {
		fail(c, 500, message.Get(c, "cloud login fail"))
		return
	}

	view := cloudLoginView{
		Token: token, Username: user.Username, Plan: link.Plan, FavoriteQuota: link.FavoriteQuota,
	}
	if link.AutoSync {
		syncView, syncErr := h.autoSync(c, link)
		view.Sync = syncView
		view.SyncError = syncErr
	}

	ok(c, message.Get(c, "cloud login success"), view)
}

// bootstrapCloudUser 数据库无用户时，用云账号创建首个本地管理员。
func (h *Handler) bootstrapCloudUser(c *gin.Context, result *AuthResult, password string) {
	username := result.User.Username
	if username == "" {
		username = result.Account.Username
	}
	if username == "" {
		fail(c, 201, message.Get(c, "cloud login fail"))
		return
	}

	passwordHash, passwordVersion, err := hashCloudPassword(password)
	if err != nil {
		logger.Logger.Errorf("创建bcrypt密码失败, err: %v", err)
		fail(c, 500, message.Get(c, "create fail"))
		return
	}

	user := models.User{
		Username:        username,
		Nickname:        username,
		Role:            "admin",
		Avatar:          "1",
		Password:        passwordHash,
		PasswordVersion: passwordVersion,
		Disabled:        false,
		RoomCreation:    true,
	}
	created := true
	var link *models.CloudLink
	if err = h.userDao.Create(&user); err != nil {
		// 并发首次登录时可能已被另一个请求创建，重新读取并确认云账号归属。
		existing, getErr := h.getLocalUser(username)
		if getErr != nil || existing.Username == "" {
			logger.Logger.Errorf("创建本地用户失败, err: %v", err)
			fail(c, 500, message.Get(c, "create fail"))
			return
		}
		existingLink, linkErr := h.cloudLinkDao.GetByCloudAccountID(result.Account.ID)
		if linkErr != nil {
			logger.Logger.Errorf("查询云账号绑定失败, err: %v", linkErr)
			fail(c, 500, message.Get(c, "database error"))
			return
		}
		if existingLink.ID == 0 || existingLink.Username != existing.Username {
			// 本地账号已被另一个云账号或本地注册占用，必须走绑定流程。
			fail(c, 201, message.Get(c, "cloud not linked"))
			return
		}
		user = *existing
		link = existingLink
		created = false
	}

	if link == nil {
		link = &models.CloudLink{Username: user.Username, AutoSync: true}
	}
	if err = h.applyAuthResult(link, result, password); err != nil {
		if created {
			_ = h.userDao.DeleteUser(user.Username)
		}
		logger.Logger.Errorf("保存云账号信息失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}
	if err = h.saveLink(link); err != nil {
		if created {
			_ = h.userDao.DeleteUser(user.Username)
		}
		logger.Logger.Errorf("保存云账号绑定失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}

	token, err := h.issueLocalToken(c, &user)
	if err != nil {
		fail(c, 500, message.Get(c, "cloud login fail"))
		return
	}

	view := cloudLoginView{
		Token: token, Username: user.Username, Plan: link.Plan, FavoriteQuota: link.FavoriteQuota,
	}
	syncView, syncErr := h.autoSync(c, link)
	view.Sync = syncView
	view.SyncError = syncErr

	ok(c, message.Get(c, "cloud login success"), view)
}

// fallbackLocalLogin 云服务不可用时，按云用户名/邮箱或本地用户名回退到本地校验。
func (h *Handler) fallbackLocalLogin(c *gin.Context, identity, password string) {
	link, err := h.cloudLinkDao.GetByIdentity(identity)
	if err != nil {
		logger.Logger.Errorf("查询云账号绑定失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}

	var user *models.User
	if link.ID != 0 {
		user, err = h.getLocalUser(link.Username)
		if err != nil {
			logger.Logger.Errorf("查询本地用户失败, err: %v", err)
			fail(c, 500, message.Get(c, "database error"))
			return
		}
		if user.Username == "" {
			fail(c, 201, message.Get(c, "cloud local user missing"))
			return
		}
		if !validateCloudPassword(password, link.CloudPassword) && !utils.ValidatePassword(password, user.Password) {
			fail(c, 201, message.Get(c, "cloud password wrong"))
			return
		}
	} else {
		user, err = h.getLocalUser(identity)
		if err != nil {
			logger.Logger.Errorf("查询本地用户失败, err: %v", err)
			fail(c, 500, message.Get(c, "database error"))
			return
		}
		if user.Username == "" || !utils.ValidatePassword(password, user.Password) {
			fail(c, 201, message.Get(c, "cloud unavailable"))
			return
		}
	}

	if user.Disabled {
		fail(c, 201, message.Get(c, "cloud local user disabled"))
		return
	}

	token, err := h.issueLocalToken(c, user)
	if err != nil {
		fail(c, 500, message.Get(c, "cloud login fail"))
		return
	}

	plan := "basic"
	favoriteQuota := 1
	if link.ID != 0 {
		plan = link.Plan
		favoriteQuota = link.FavoriteQuota
	}
	view := cloudLoginView{
		Token: token, Username: user.Username, Fallback: true, Plan: plan, FavoriteQuota: favoriteQuota,
		SyncError: message.Get(c, "cloud unavailable"),
	}

	ok(c, message.Get(c, "cloud local fallback success"), view)
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
		fail(c, 400, message.Get(c, "bad request"))
		return
	}
	identity := req.identity()
	if identity == "" || req.Password == "" {
		fail(c, 400, message.Get(c, "bad request"))
		return
	}

	username := currentUsername(c)
	user, err := h.getLocalUser(username)
	if err != nil {
		logger.Logger.Errorf("查询本地用户失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}
	if user.Username == "" {
		fail(c, 201, message.Get(c, "cloud local user missing"))
		return
	}

	result, err := h.client.Login(c.Request.Context(), identity, req.Password)
	if err != nil {
		failCloud(c, err)
		return
	}

	existing, err := h.cloudLinkDao.GetByCloudAccountID(result.Account.ID)
	if err != nil {
		logger.Logger.Errorf("查询云账号绑定失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}
	if existing.ID != 0 && existing.Username != username {
		fail(c, 201, message.Get(c, "cloud already linked"))
		return
	}

	link, err := h.cloudLinkDao.GetByUsername(username)
	if err != nil {
		logger.Logger.Errorf("查询云账号绑定失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}
	if link.ID == 0 {
		link = &models.CloudLink{Username: username, AutoSync: true}
	} else if link.CloudAccountID != 0 && link.CloudAccountID != result.Account.ID {
		// 切换云账号时清除旧账号的收藏映射，避免把新账号的收藏关联到旧映射。
		if err = h.cloudMapDao.DeleteByUsername(username); err != nil {
			logger.Logger.Errorf("清理云收藏映射失败, err: %v", err)
			fail(c, 500, message.Get(c, "database error"))
			return
		}
	}

	if err = h.applyAuthResult(link, result, req.Password); err != nil {
		logger.Logger.Errorf("保存云账号信息失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}
	if err = h.saveLink(link); err != nil {
		logger.Logger.Errorf("保存云账号绑定失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}

	view := linkView(link)
	view["favoriteCount"] = h.localFavoriteCount(username)
	ok(c, message.Get(c, "cloud link success"), view)
}

type unlinkReq struct {
	Password string `json:"password"`
}

// unlinkPost 解除云账号绑定。需要本地密码或云账号密码确认，防止会话被劫持后直接解绑。
func (h *Handler) unlinkPost(c *gin.Context) {
	var req unlinkReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, message.Get(c, "bad request"))
		return
	}

	username := currentUsername(c)
	unlock := lockUserSync(username)
	defer unlock()

	user, err := h.getLocalUser(username)
	if err != nil {
		logger.Logger.Errorf("查询本地用户失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}
	link, err := h.cloudLinkDao.GetByUsername(username)
	if err != nil {
		logger.Logger.Errorf("查询云账号绑定失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}
	if link.ID == 0 {
		fail(c, 201, message.Get(c, "cloud link not found"))
		return
	}

	passwordOK := validateCloudPassword(req.Password, link.CloudPassword)
	if user.Username != "" {
		passwordOK = passwordOK || utils.ValidatePassword(req.Password, user.Password)
	}
	if !passwordOK {
		fail(c, 201, message.Get(c, "cloud password wrong"))
		return
	}

	if err = h.cloudLinkDao.DeleteByUsername(username); err != nil {
		logger.Logger.Errorf("解除云账号绑定失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}
	if err = h.cloudMapDao.DeleteByUsername(username); err != nil {
		logger.Logger.Errorf("清理云收藏映射失败, err: %v", err)
	}

	ok(c, message.Get(c, "cloud unlink success"), gin.H{"linked": false})
}

// statusGet 返回当前本地账号的云绑定状态。
func (h *Handler) statusGet(c *gin.Context) {
	link, err := h.getLink(currentUsername(c))
	if err != nil {
		logger.Logger.Errorf("查询云账号绑定失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}

	view := linkView(link)
	view["favoriteCount"] = h.localFavoriteCount(currentUsername(c))
	ok(c, "success", view)
}

func (h *Handler) favoritesGet(c *gin.Context) {
	link, okLink := h.requireLink(c)
	if !okLink {
		return
	}

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

	result, err := h.client.ListFavorites(c.Request.Context(), link, page, pageSize, c.Query("keyword"))
	if err != nil {
		failCloud(c, err)
		return
	}
	if err = h.saveLink(link); err != nil {
		logger.Logger.Errorf("保存云账号 token 失败, err: %v", err)
	}

	ok(c, "success", result)
}

func (h *Handler) tokensGet(c *gin.Context) {
	link, okLink := h.requireLink(c)
	if !okLink {
		return
	}

	result, err := h.client.ListTokens(c.Request.Context(), link)
	if err != nil {
		failCloud(c, err)
		return
	}
	if err = h.saveLink(link); err != nil {
		logger.Logger.Errorf("保存云账号 token 失败, err: %v", err)
	}

	ok(c, "success", result)
}

func (h *Handler) gameEntriesGet(c *gin.Context, kind string) {
	link, okLink := h.requireLink(c)
	if !okLink {
		return
	}

	result, err := h.client.ListGameEntries(c.Request.Context(), link, kind)
	if err != nil {
		failCloud(c, err)
		return
	}
	if err = h.saveLink(link); err != nil {
		logger.Logger.Errorf("保存云账号 token 失败, err: %v", err)
	}

	ok(c, "success", result)
}

func (h *Handler) gameAdminsGet(c *gin.Context) {
	h.gameEntriesGet(c, "admin")
}

func (h *Handler) gameSlotsGet(c *gin.Context) {
	h.gameEntriesGet(c, "slot")
}

func (h *Handler) requireLink(c *gin.Context) (*models.CloudLink, bool) {
	link, err := h.getLink(currentUsername(c))
	if err != nil {
		logger.Logger.Errorf("查询云账号绑定失败, err: %v", err)
		fail(c, 500, message.Get(c, "database error"))
		return nil, false
	}
	if link.ID == 0 {
		fail(c, 201, message.Get(c, "cloud link not found"))
		return nil, false
	}

	return link, true
}

type syncReq struct {
	Direction string `json:"direction"`
}

func (h *Handler) syncPost(c *gin.Context) {
	var req syncReq
	if err := c.ShouldBindJSON(&req); err != nil {
		fail(c, 400, message.Get(c, "bad request"))
		return
	}
	direction := strings.TrimSpace(req.Direction)
	if direction == "" {
		direction = "both"
	}
	if direction != "both" && direction != "upload" && direction != "download" {
		fail(c, 201, message.Get(c, "cloud sync invalid direction"))
		return
	}

	link, okLink := h.requireLink(c)
	if !okLink {
		return
	}

	result, localCount, err := h.syncWithLink(c, link, direction)
	if err != nil {
		if errors.Is(err, ErrRelink) || IsUnavailable(err) || MessageOf(err) != "" {
			failCloud(c, err)
			return
		}
		logger.Logger.Errorf("云同步失败, username: %s, err: %v", link.Username, err)
		fail(c, 201, message.Get(c, "cloud sync fail"))
		return
	}

	ok(c, message.Get(c, "cloud sync success"), newSyncSummaryView(direction, result, localCount))
}

// autoSync 在云账号登录后执行一次双向同步，失败不影响登录。
func (h *Handler) autoSync(c *gin.Context, link *models.CloudLink) (*syncSummaryView, string) {
	result, localCount, err := h.syncWithLink(c, link, "both")
	if err != nil {
		logger.Logger.Warnf("云账号登录后自动同步失败, username: %s, err: %v", link.Username, err)
		return nil, message.Get(c, "cloud sync fail")
	}
	view := newSyncSummaryView("both", result, localCount)

	return &view, ""
}
