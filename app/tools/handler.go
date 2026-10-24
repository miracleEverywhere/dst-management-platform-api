package tools

import (
	"context"
	"crypto/sha256"
	"dst-management-platform-api/aichat"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/utils"
	"dst-management-platform-api/webhook"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func (h *Handler) backupGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.RoomID == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	backups, err := game.GetBackups()
	if err != nil {
		logger.Logger.Errorf("获取备份文件失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "get backup fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": backups})
}

func (h *Handler) backupPost(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.RoomID == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	err = game.Backup()
	if err != nil {
		logger.Logger.Errorf("创建备份文件失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "create backup fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "create backup success"), "data": nil})
}

func (h *Handler) backupDelete(c *gin.Context) {
	type ReqForm struct {
		RoomID    int      `json:"roomID"`
		Filenames []string `json:"filenames"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.RoomID == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	for _, filename := range reqForm.Filenames {
		if !utils.IsSafePath(filename) {
			logger.Logger.Warnf("检测到穿越攻击: %s, api: %s", filename, c.Request.URL.Path)
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
			return
		}
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	count := game.DeleteBackups(reqForm.Filenames)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "delete success"), "data": count})
}

func (h *Handler) backupRestorePost(c *gin.Context) {
	type ReqForm struct {
		RoomID   int    `json:"roomID"`
		Filename string `json:"filename"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.RoomID == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !utils.IsSafeString(reqForm.Filename) {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	saveData, err := game.Restore(reqForm.Filename)
	if err != nil {
		logger.Logger.Errorf("恢复失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "restore fail"), "data": nil})
		return
	}

	err = h.roomDao.UpdateRoom(&saveData.Room)
	if err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "restore fail"), "data": nil})
		return
	}

	err = h.worldDao.UpdateWorlds(&saveData.Worlds)
	if err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "restore fail"), "data": nil})
		return
	}

	err = h.roomSettingDao.UpdateRoomSetting(&saveData.RoomSetting)
	if err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "restore fail"), "data": nil})
		return
	}
	if err = h.aiManager.Reload(reqForm.RoomID); err != nil {
		logger.Logger.Errorf("重载房间 AI 对话失败, roomID: %d, err: %v", reqForm.RoomID, err)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "restore success"), "data": nil})
}

func (h *Handler) backupDownloadGet(c *gin.Context) {
	// 1. 获取路径参数
	type ReqForm struct {
		RoomID   int    `json:"roomID" form:"roomID"`
		Filename string `json:"filename" form:"filename"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	// 2. 参数验证
	if reqForm.RoomID == 0 || reqForm.Filename == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	// 3. 安全验证（防止路径遍历）
	if !utils.IsSafePath(reqForm.Filename) {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}
	// 4. 构建文件路径
	filePath := fmt.Sprintf("dmp_files/backup/%d/%s", reqForm.RoomID, reqForm.Filename)
	// 5. 构造文件名
	fileName := reqForm.Filename
	base64Str := strings.TrimSuffix(fileName, ".zip")
	decodeFilename, err := utils.Base64Decode(base64Str)
	if err != nil {
		c.FileAttachment(filePath, fileName)
		return
	}
	// 房间名，房间天数，备份时间
	decodeFilenameParts := strings.Split(decodeFilename, "<-@dmp@->")
	if len(decodeFilenameParts) != 3 {
		c.FileAttachment(filePath, fileName)
		return
	}
	// 文件名安全替换
	cycle := decodeFilenameParts[1] + "天"
	var invalidChars = regexp.MustCompile(`[\x00-\x1f\\/:*?"<>|]`)
	roomName := invalidChars.ReplaceAllString(decodeFilenameParts[0], "")
	if roomName == "" {
		roomName = "存档"
	}
	ms, err := strconv.ParseInt(decodeFilenameParts[2], 10, 64)
	if err != nil {
		fileName = fmt.Sprintf("%s_%s.zip", roomName, cycle)
		c.FileAttachment(filePath, fileName)
		return
	}
	ts := time.UnixMilli(ms)
	tsStr := ts.Format("2006-01-02-15-04")
	fileName = fmt.Sprintf("%s_%s_%s.zip", roomName, cycle, tsStr)
	c.FileAttachment(filePath, fileName)
}

func (h *Handler) announceGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.RoomID == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	roomSetting, err := h.roomSettingDao.GetRoomSettingsByRoomID(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": roomSetting.AnnounceSetting})
}

func (h *Handler) announcePut(c *gin.Context) {
	type ReqForm struct {
		RoomID  int    `json:"roomID"`
		Setting string `json:"setting"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.RoomID == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))

	roomSetting.AnnounceSetting = reqForm.Setting

	err = h.roomSettingDao.UpdateRoomSetting(roomSetting)
	if err != nil {
		logger.Logger.Errorf("更新通知设置失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "update fail"), "data": nil})
		return
	}

	if room.Status {
		// 更新定时任务
		jobNames := scheduler.GetJobsByType(reqForm.RoomID, "Announce")
		logger.Logger.Debug(utils.StructToFlatString(jobNames))
		for _, jobName := range jobNames {
			// 删除所有通知任务
			scheduler.DeleteJob(jobName)
		}
		var announces []scheduler.AnnounceSetting
		if err = json.Unmarshal([]byte(roomSetting.AnnounceSetting), &announces); err != nil {
			logger.Logger.Errorf("获取定时通知设置失败, err: %v", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "update fail"), "data": nil})
			return
		}

		logger.Logger.Debug(utils.StructToFlatString(announces))

		for _, announce := range announces {
			// 创建通知任务
			if announce.Status {
				// 注意，-为分隔符，需要删除uuid中的-
				err = scheduler.UpdateJob(&scheduler.JobConfig{
					Name:     fmt.Sprintf("%d-%s-Announce", room.ID, strings.ReplaceAll(announce.ID, "-", "")),
					Func:     scheduler.Announce,
					Args:     []any{game, announce.Content},
					TimeType: scheduler.SecondType,
					Interval: announce.Interval,
					DayAt:    "",
				})
				if err != nil {
					logger.Logger.Errorf("定时通知定时任务处理失败, err: %v", err)
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "update success"), "data": nil})
}

func tokenPost(c *gin.Context) {
	type ReqForm struct {
		Expiration int `json:"expiration"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.Expiration < 0 {
		logger.Logger.Infof("请求参数错误, api: %s", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	username, _ := c.Get("username")
	nickname, _ := c.Get("nickname")

	cache.TokenVersionCacheLock.RLock()
	cachedVersion, exists := cache.TokenVersionCache[username.(string)]
	cache.TokenVersionCacheLock.RUnlock()
	if !exists {
		cachedVersion = 0
	}

	user := models.User{
		Username:     username.(string),
		Nickname:     nickname.(string),
		Role:         "admin",
		TokenVersion: cachedVersion,
	}

	var expiration int

	if reqForm.Expiration == 0 {
		// 生成永久token 99年
		expiration = 99 * 365 * 24
	} else {
		expiration = reqForm.Expiration
	}

	token, err := utils.GenerateJWT(user, []byte(cache.JwtSecret), expiration)
	if err != nil {
		logger.Logger.Errorf("创建token失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "create fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "create success"), "data": token})
}

func (h *Handler) snapshotGet(c *gin.Context) {
	type ReqForm struct {
		RoomID int `json:"roomID" form:"roomID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.RoomID == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	snapshot, err := game.GetSnapshot()
	if err != nil {
		logger.Logger.Errorf("获取游戏存档文件失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "get snapshot fail"), "data": snapshot})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": snapshot})
}

func (h *Handler) snapshotDelete(c *gin.Context) {
	type ReqForm struct {
		RoomID int    `json:"roomID"`
		Name   string `json:"name"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if reqForm.RoomID == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !utils.IsSafePath(reqForm.Name) {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))

	// 关闭游戏
	err = game.StopAllWorld()
	if err != nil {
		logger.Logger.Warnf("关闭游戏失败：%v，可能是游戏未运行，跳过", err)
	}

	// 删除存档文件
	err = game.DeleteSnapshot(reqForm.Name)
	if err != nil {
		logger.Logger.Errorf("删除游戏存档文件失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "delete fail"), "data": nil})
		return
	}

	// 启动游戏
	err = game.StartAllWorld()
	if err != nil {
		logger.Logger.Errorf("启动游戏失败：%v", err)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "delete success"), "data": nil})
}

func (h *Handler) categoryGet(c *gin.Context) {
	categories, err := h.dstImageDao.Categories()
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": categories})
}

func (h *Handler) categoryItemsGet(c *gin.Context) {
	var reqForm struct {
		Category string `json:"category" form:"category"`
		Page     int    `json:"page" form:"page"`
		PageSize int    `json:"pageSize" form:"pageSize"`
	}
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	images, err := h.dstImageDao.List(reqForm.Category, reqForm.Page, reqForm.PageSize)
	if err != nil {
		logger.Logger.Errorf("查询数据库失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	var needParse []string

	for _, image := range images.Data {
		if image.Image == "" {
			needParse = append(needParse, image.Prefab)
		}
	}

	if len(needParse) > 0 {
		b64Map := pngsToBase64(needParse, reqForm.Page)
		for k, v := range b64Map {
			if v == "" {
				continue
			}
			for index, img := range images.Data {
				if img.Prefab == k {
					images.Data[index].Image = v
					err = h.dstImageDao.UpdateImage(&images.Data[index])
					if err != nil {
						logger.Logger.Errorf("更新图片失败: %v, 跳过", err)
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": images})
}

func (h *Handler) consolePost(c *gin.Context) {
	var reqForm struct {
		RoomID  int    `json:"roomID" binding:"required"`
		WorldID int    `json:"worldID" binding:"required"`
		Type    string `json:"type" binding:"required"`
		Uid     string `json:"uid"`
		Prefab  string `json:"prefab"`
		Num     int    `json:"num"`
	}
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(reqForm.RoomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	err = game.TmiConsoleCmd(reqForm.Type, reqForm.Uid, reqForm.Prefab, reqForm.Num, reqForm.WorldID)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "exec fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "exec success"), "data": nil})
}

func (h *Handler) aiSettingGet(c *gin.Context) {
	var reqForm struct {
		RoomID int `form:"roomID" binding:"required"`
	}
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	if !h.hasPermission(c, strconv.Itoa(reqForm.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}

	setting, err := h.roomAISettingDao.GetByRoomID(reqForm.RoomID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		setting = &models.RoomAISetting{
			RoomID:         reqForm.RoomID,
			MaxResults:     models.DefaultAIWikiMaxResults,
			MaxReplyLength: models.DefaultAIReplyMaxLength,
		}
	} else if err != nil {
		logger.Logger.Errorf("获取房间 AI 配置失败, roomID: %d, err: %v", reqForm.RoomID, err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}
	if setting.MaxResults == 0 {
		setting.MaxResults = models.DefaultAIWikiMaxResults
	}
	if setting.MaxReplyLength < models.MinAIReplyLength {
		setting.MaxReplyLength = models.DefaultAIReplyMaxLength
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": setting})
}

func (h *Handler) aiSettingPut(c *gin.Context) {
	var reqForm struct {
		RoomID         int                   `json:"roomID"`
		Enabled        bool                  `json:"enabled"`
		AllowChat      bool                  `json:"allowChat"`
		Prefix         string                `json:"prefix"`
		MaxResults     int                   `json:"maxResults"`
		MaxReplyLength int                   `json:"maxReplyLength"`
		Setting        *models.RoomAISetting `json:"setting"`
	}
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	setting := models.RoomAISetting{
		RoomID:         reqForm.RoomID,
		Enabled:        reqForm.Enabled,
		AllowChat:      reqForm.AllowChat,
		Prefix:         reqForm.Prefix,
		MaxResults:     reqForm.MaxResults,
		MaxReplyLength: reqForm.MaxReplyLength,
	}
	// 兼容旧版 {setting: {...}} 请求体，嵌套对象也只能解析房间级字段。
	if reqForm.Setting != nil {
		setting = *reqForm.Setting
	}

	if !h.hasPermission(c, strconv.Itoa(setting.RoomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return
	}
	if _, err := h.roomDao.GetRoomByID(setting.RoomID); err != nil {
		logger.Logger.Errorf("获取房间信息失败, roomID: %d, err: %v", setting.RoomID, err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}
	if err := aichat.ValidateRoomSetting(&setting); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": err.Error(), "data": nil})
		return
	}
	if setting.Enabled {
		baseSetting, err := h.systemDao.GetAIBaseSetting()
		if err != nil {
			logger.Logger.Errorf("获取 AI 基础配置失败, err: %v", err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
			return
		}
		if err = aichat.ValidateBaseSetting(baseSetting); err != nil {
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": err.Error(), "data": nil})
			return
		}
	}

	if err := h.roomAISettingDao.UpdateSetting(&setting); err != nil {
		logger.Logger.Errorf("更新房间 AI 配置失败, roomID: %d, err: %v", setting.RoomID, err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	webhook.Snd.Send(webhook.EventAIChatSettingUpdated, setting.RoomID, map[string]interface{}{
		"enabled":        setting.Enabled,
		"allowChat":      setting.AllowChat,
		"prefix":         setting.Prefix,
		"maxResults":     setting.MaxResults,
		"maxReplyLength": setting.MaxReplyLength,
		"username":       c.GetString("username"),
	})
	if err := h.aiManager.Reload(setting.RoomID); err != nil {
		logger.Logger.Errorf("重载房间 AI 对话失败, roomID: %d, err: %v", setting.RoomID, err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "update fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "update success"), "data": nil})
}

func (h *Handler) aiBaseSettingGet(c *gin.Context) {
	setting, err := h.systemDao.GetAIBaseSetting()
	if err != nil {
		logger.Logger.Errorf("获取 AI 基础配置失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	// embeddingIndexDimensions 为磁盘上已有向量索引的维度，没有索引时为 -1。
	response := struct {
		models.AIBaseSetting
		EmbeddingIndexDimensions int `json:"embeddingIndexDimensions"`
	}{
		AIBaseSetting:            *setting,
		EmbeddingIndexDimensions: h.aiManager.GetEmbeddingIndexDimensions(),
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": response})
}

func (h *Handler) aiBaseSettingPut(c *gin.Context) {
	var reqForm models.AIBaseSetting
	if err := c.ShouldBindJSON(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}
	if err := aichat.ValidateBaseSetting(&reqForm); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": err.Error(), "data": nil})
		return
	}
	if err := h.systemDao.UpdateAIBaseSetting(&reqForm); err != nil {
		logger.Logger.Errorf("更新 AI 基础配置失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	webhook.Snd.Send(webhook.EventAIChatBaseSettingUpdated, 0, map[string]interface{}{
		"username":       c.GetString("username"),
		"chatModel":      reqForm.ChatModel,
		"embeddingModel": reqForm.EmbeddingModel,
	})
	if err := h.aiManager.Reload(); err != nil {
		logger.Logger.Errorf("重载 AI 对话服务失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "update fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "update success"), "data": nil})
}

func (h *Handler) aiKeywordIndexReBuild(c *gin.Context) {
	_ = h.aiManager.BuildKeywordIndex(true)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "update success"), "data": nil})
}

func (h *Handler) aiEmbeddingIndexReBuild(c *gin.Context) {
	setting, err := h.systemDao.GetAIBaseSetting()
	if err != nil {
		logger.Logger.Errorf("获取 AI 基础配置失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return
	}

	baseURL := strings.TrimSpace(setting.EmbeddingBaseURL)
	model := strings.TrimSpace(setting.EmbeddingModel)
	if baseURL == "" || setting.EmbeddingApiKey == "" || model == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "embedding config incomplete"), "data": nil})
		return
	}
	if !utils.IsValidURL(baseURL) {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "embedding base url invalid"), "data": nil})
		return
	}

	err = h.aiManager.BuildEmbeddingIndex(aichat.EmbeddingConfig{
		APIURL:     baseURL,
		APIKey:     setting.EmbeddingApiKey,
		Model:      model,
		Dimensions: setting.EmbeddingDimensions,
	}, true)
	if err != nil {
		logger.Logger.Errorf("重建 embedding 索引失败, err: %v", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "update fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "update success"), "data": nil})
}

func (h *Handler) aiEmbeddingIndexCancel(c *gin.Context) {
	if !h.aiManager.CancelEmbeddingIndexBuild() {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "embedding build not running"), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message.Get(c, "embedding build cancel success"), "data": nil})
}

// mapWorldContext 统一地图接口的 roomID/worldID 校验：参数、房间权限、世界归属。
// 校验失败时已经写好响应，调用方直接 return。
func (h *Handler) mapWorldContext(c *gin.Context, roomID, worldID int) (*models.Room, *[]models.World, *models.RoomSetting, string, bool) {
	if roomID <= 0 || worldID <= 0 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return nil, nil, nil, "", false
	}
	if !h.hasPermission(c, strconv.Itoa(roomID)) {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message.Get(c, "permission needed"), "data": nil})
		return nil, nil, nil, "", false
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("地图获取房间信息失败, roomID: %d, err: %v", roomID, err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "database error"), "data": nil})
		return nil, nil, nil, "", false
	}

	for _, world := range *worlds {
		if world.ID == worldID {
			return room, worlds, roomSetting, world.WorldName, true
		}
	}
	c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
	return nil, nil, nil, "", false
}

// mapGet 返回完整世界地图（实时从存档生成）以及可选的实时在线玩家。
func (h *Handler) mapGet(c *gin.Context) {
	type ReqForm struct {
		RoomID  int    `form:"roomID"`
		WorldID int    `form:"worldID"`
		Players string `form:"players"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	probePlayers := true
	if value := strings.TrimSpace(reqForm.Players); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || (parsed != 0 && parsed != 1) {
			c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
			return
		}
		probePlayers = parsed == 1
	}

	room, worlds, roomSetting, worldName, ok := h.mapWorldContext(c, reqForm.RoomID, reqForm.WorldID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), dst.MapRequestTimeoutSeconds*time.Second)
	defer cancel()

	lockCtx, cancelLock := context.WithTimeout(ctx, dst.MapLockWaitSeconds*time.Second)
	release, err := cache.AcquireMap(lockCtx, reqForm.RoomID, reqForm.WorldID)
	cancelLock()
	if err != nil {
		logger.Logger.Warnf("地图等待生成锁超时, roomID: %d, worldID: %d, err: %v", reqForm.RoomID, reqForm.WorldID, err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "map busy"), "data": nil})
		return
	}
	defer release()

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	mapData, err := game.GenerateFullMap(ctx, reqForm.WorldID)
	if err != nil {
		switch {
		case errors.Is(err, dst.ErrMapSaveNotFound):
			logger.Logger.Warnf("地图存档不存在, roomID: %d, worldID: %d, worldName: %s, err: %v", reqForm.RoomID, reqForm.WorldID, worldName, err)
			c.JSON(http.StatusOK, gin.H{"code": 404, "message": message.Get(c, "map save not found"), "data": nil})
		case errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(ctx.Err(), context.Canceled):
			logger.Logger.Warnf("地图生成超时或被取消, roomID: %d, worldID: %d, worldName: %s", reqForm.RoomID, reqForm.WorldID, worldName)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "map busy"), "data": nil})
		default:
			logger.Logger.Errorf("地图生成失败, roomID: %d, worldID: %d, worldName: %s, err: %v", reqForm.RoomID, reqForm.WorldID, worldName, err)
			c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "map parse fail"), "data": nil})
		}
		return
	}

	assetCtx, cancelAssets := context.WithTimeout(ctx, dst.MapAssetFillTimeoutSeconds*time.Second)
	h.fillMapAssets(assetCtx, &mapData)
	cancelAssets()

	if probePlayers {
		mapData.Players = game.PlayerPositions(ctx, reqForm.WorldID)
		if mapData.Players.Error != "" {
			logger.Logger.Warnf("地图在线玩家获取失败, roomID: %d, worldID: %d, err: %s", reqForm.RoomID, reqForm.WorldID, mapData.Players.Error)
		}
	} else {
		mapData.Players = dst.MapPlayers{
			ProbedAt:     time.Now().UTC(),
			WorldRunning: game.WorldUpStatus(reqForm.WorldID),
			List:         []dst.MapPlayer{},
		}
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": mapData})
}

// mapPlayersGet 只刷新在线玩家位置，不重新解析地图。
// 前端拿到地图后可以每 x 秒调用一次，只移动玩家图标。
func (h *Handler) mapPlayersGet(c *gin.Context) {
	type ReqForm struct {
		RoomID  int `form:"roomID"`
		WorldID int `form:"worldID"`
	}
	var reqForm ReqForm
	if err := c.ShouldBindQuery(&reqForm); err != nil {
		logger.Logger.Infof("请求参数错误: %v, api: %s", err, c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message.Get(c, "bad request"), "data": nil})
		return
	}

	room, worlds, roomSetting, worldName, ok := h.mapWorldContext(c, reqForm.RoomID, reqForm.WorldID)
	if !ok {
		return
	}

	ctx, cancel := context.WithTimeout(c.Request.Context(), dst.MapPlayersRequestTimeoutSeconds*time.Second)
	defer cancel()

	lockCtx, cancelLock := context.WithTimeout(ctx, dst.MapLockWaitSeconds*time.Second)
	release, err := cache.AcquireMap(lockCtx, reqForm.RoomID, reqForm.WorldID)
	cancelLock()
	if err != nil {
		logger.Logger.Warnf("地图玩家刷新等待锁超时, roomID: %d, worldID: %d, err: %v", reqForm.RoomID, reqForm.WorldID, err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message.Get(c, "map busy"), "data": nil})
		return
	}
	defer release()

	game := dst.NewGameController(room, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	players := game.PlayerPositions(ctx, reqForm.WorldID)
	if players.Error != "" {
		logger.Logger.Warnf("地图在线玩家刷新失败, roomID: %d, worldID: %d, worldName: %s, err: %s",
			reqForm.RoomID, reqForm.WorldID, worldName, players.Error)
	}

	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": players})
}

// fillMapAssets 把地形贴图直接填进 assets，并把 minimap 图标打包进 iconAtlas。
// 先查 map_images，缺失的再解包并回写；超时或资源缺失时保留空 PNG，由前端回退。
func (h *Handler) fillMapAssets(ctx context.Context, mapData *dst.MapFull) {
	if mapData == nil || h.mapImageDao == nil {
		return
	}

	mapAssetExtractMutex.Lock()
	defer mapAssetExtractMutex.Unlock()

	gameBuild := strings.TrimSpace(mapData.GameBuild)
	if gameBuild == "" {
		gameBuild = "unknown"
	}

	icons := make([]string, 0, len(mapData.Prefabs))
	tiles := make([]string, 0, len(mapData.Assets))
	seen := make(map[string]struct{}, len(mapData.Prefabs)+len(mapData.Assets))
	for _, prefab := range mapData.Prefabs {
		name := prefab.Icon
		if name == "" {
			continue
		}
		if !strings.HasSuffix(name, ".png") {
			name += ".png"
		}
		key := mapImageKey("icon", name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		icons = append(icons, name)
	}
	for _, ref := range mapData.Assets {
		if ref.Name == "" {
			continue
		}
		key := mapImageKey("tile", ref.Name)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		tiles = append(tiles, ref.Name)
	}

	rows := make(map[string]models.MapImage, len(icons)+len(tiles))
	load := func(kind string, names []string) bool {
		if len(names) == 0 {
			return true
		}
		got, err := h.mapImageDao.GetMany(kind, gameBuild, names)
		if err != nil {
			logger.Logger.Warnf("地图图片查询失败, kind: %s, build: %s, err: %v", kind, gameBuild, err)
			return false
		}
		for _, row := range got {
			rows[mapImageKey(kind, row.Name)] = row
		}
		return true
	}
	if !load("icon", icons) || !load("tile", tiles) {
		return
	}

	missing := map[string][]string{"icon": {}, "tile": {}}
	for _, name := range icons {
		if row, ok := rows[mapImageKey("icon", name)]; !ok || row.Image == "" {
			missing["icon"] = append(missing["icon"], name)
		}
	}
	for _, name := range tiles {
		if row, ok := rows[mapImageKey("tile", name)]; !ok || row.Image == "" {
			missing["tile"] = append(missing["tile"], name)
		}
	}

	for _, kind := range []string{"icon", "tile"} {
		names := missing[kind]
		if len(names) == 0 {
			continue
		}
		assets := dst.ExtractMapAssets(ctx, kind, names)
		if len(assets) == 0 {
			continue
		}
		records := make([]models.MapImage, 0, len(assets))
		for _, asset := range assets {
			record := models.MapImage{
				Name:      asset.Name,
				Kind:      asset.Kind,
				GameBuild: gameBuild,
				Version:   models.MapImageVersion,
				Image:     asset.ImageBase64,
				Width:     asset.Width,
				Height:    asset.Height,
				SHA256:    asset.SHA256,
			}
			records = append(records, record)
			rows[mapImageKey(kind, asset.Name)] = record
		}
		if err := h.mapImageDao.UpsertMany(records); err != nil {
			logger.Logger.Warnf("地图图片缓存失败, kind: %s, build: %s, err: %v", kind, gameBuild, err)
		}
	}

	terrain := make([]dst.MapAssetRef, 0, len(tiles))
	normalizedRows := make([]models.MapImage, 0)
	for _, name := range tiles {
		key := mapImageKey("tile", name)
		row, ok := rows[key]
		if !ok || row.Image == "" {
			continue
		}
		// 兼容旧缓存：如果仍是 256px，按当前策略重压后再返回。
		if row.Width > dst.MapTileTextureSide || row.Height > dst.MapTileTextureSide || row.Width == 0 {
			if png, width, height, ok := dst.NormalizeMapAssetBase64("tile", row.Image); ok {
				row.Image = png
				row.Width = width
				row.Height = height
				row.Version = models.MapImageVersion
				if data, err := base64.StdEncoding.DecodeString(png); err == nil {
					row.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
				}
				rows[key] = row
				normalizedRows = append(normalizedRows, row)
			}
		}
		terrain = append(terrain, dst.MapAssetRef{
			Name: name,
			PNG:  row.Image,
		})
	}
	mapData.Assets = terrain
	if len(normalizedRows) > 0 {
		if err := h.mapImageDao.UpsertMany(normalizedRows); err != nil {
			logger.Logger.Warnf("地图旧图片重新压缩失败, build: %s, err: %v", gameBuild, err)
		}
	}

	iconImages := make(map[string]dst.MapAssetData, len(icons))
	for _, name := range icons {
		row, ok := rows[mapImageKey("icon", name)]
		if !ok || row.Image == "" {
			continue
		}
		iconImages[name] = dst.MapAssetData{
			Name:        name,
			Kind:        "icon",
			ImageBase64: row.Image,
			Width:       row.Width,
			Height:      row.Height,
			SHA256:      row.SHA256,
		}
	}
	mapData.IconAtlas = dst.BuildMapIconAtlas(iconImages)
}

func mapImageKey(kind, name string) string {
	return kind + "\x00" + strings.ToLower(name)
}

var mapAssetExtractMutex sync.Mutex
