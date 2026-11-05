package cloud

import (
	"dst-management-platform-api/app/room"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/utils"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var workshopPattern = regexp.MustCompile(`workshop-\d+`)

var errArchiveTooLarge = errors.New("cloud archive too large")

type archiveUploadReq struct {
	RoomID int `json:"roomID"`
}

func (h *Handler) archiveUploadPost(c *gin.Context) {
	var req archiveUploadReq
	if err := c.ShouldBindJSON(&req); err != nil || req.RoomID == 0 {
		fail(c, 400, message.Get(c, "bad request"))
		return
	}
	if !h.hasRoomPermission(c, req.RoomID) {
		fail(c, 201, message.Get(c, "cloud archive permission"))
		return
	}

	link, okLink := h.requireLink(c)
	if !okLink {
		return
	}
	if err := h.refreshProfile(c, link); err != nil {
		failCloud(c, err)
		return
	}
	if link.Plan != "ultimate" {
		fail(c, 201, message.Get(c, "cloud archive ultimate only"))
		return
	}

	zipPath, remark, cleanup, err := h.prepareArchive(c, req.RoomID)
	if err != nil {
		logger.Logger.Errorf("打包云存档失败, roomID: %d, err: %v", req.RoomID, err)
		if errors.Is(err, errArchiveTooLarge) {
			fail(c, 201, message.Get(c, "cloud archive too large"))
			return
		}
		fail(c, 201, message.Get(c, "cloud archive build fail"))
		return
	}
	defer cleanup()

	roomInfo, worlds, roomSetting, err := dao.FetchGameInfo(req.RoomID)
	if err != nil {
		logger.Logger.Errorf("查询房间失败, roomID: %d, err: %v", req.RoomID, err)
		fail(c, 500, message.Get(c, "database error"))
		return
	}
	metadata, err := buildArchiveMetadata(roomInfo, worlds, roomSetting)
	if err != nil {
		logger.Logger.Errorf("生成云存档元数据失败, roomID: %d, err: %v", req.RoomID, err)
		fail(c, 201, message.Get(c, "cloud archive build fail"))
		return
	}

	archive, err := h.client.UploadArchive(c.Request.Context(), link, zipPath, roomInfo.GameName, remark, metadata)
	if err != nil {
		failCloud(c, err)
		return
	}
	if err = h.saveLink(link); err != nil {
		logger.Logger.Errorf("保存云账号 token 失败, err: %v", err)
	}

	ok(c, message.Get(c, "cloud archive upload success"), archive)
}

func (h *Handler) archivesGet(c *gin.Context) {
	link, okLink := h.requireLink(c)
	if !okLink {
		return
	}

	result, err := h.client.ListArchives(c.Request.Context(), link)
	if err != nil {
		failCloud(c, err)
		return
	}
	if err = h.saveLink(link); err != nil {
		logger.Logger.Errorf("保存云账号 token 失败, err: %v", err)
	}

	ok(c, "success", result)
}

func (h *Handler) archiveDownloadGet(c *gin.Context) {
	id := strings.TrimSpace(c.Param("id"))
	if !validCloudID(id) {
		fail(c, 400, message.Get(c, "cloud archive id invalid"))
		return
	}

	link, okLink := h.requireLink(c)
	if !okLink {
		return
	}

	body, fileName, size, err := h.client.DownloadArchive(c.Request.Context(), link, id)
	if err != nil {
		failCloud(c, err)
		return
	}
	defer body.Close()

	if err = h.saveLink(link); err != nil {
		logger.Logger.Errorf("保存云账号 token 失败, err: %v", err)
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
	if !validCloudID(id) {
		fail(c, 400, message.Get(c, "cloud archive id invalid"))
		return
	}
	if !h.canCreateRoom(c) {
		fail(c, 201, message.Get(c, "cloud archive create permission"))
		return
	}

	link, okLink := h.requireLink(c)
	if !okLink {
		return
	}

	body, _, _, err := h.client.DownloadArchive(c.Request.Context(), link, id)
	if err != nil {
		failCloud(c, err)
		return
	}
	defer body.Close()

	if err = h.saveLink(link); err != nil {
		logger.Logger.Errorf("保存云账号 token 失败, err: %v", err)
	}

	if h.roomHandler == nil {
		logger.Logger.Error("云存档导入失败：房间导入器未初始化")
		fail(c, 500, message.Get(c, "cloud archive import fail"))
		return
	}

	baseDir := filepath.Join(constant.CloudTempPath, "import")
	if err = os.MkdirAll(baseDir, 0o700); err != nil {
		logger.Logger.Errorf("创建云存档导入目录失败, err: %v", err)
		fail(c, 500, message.Get(c, "cloud archive import fail"))
		return
	}
	tempDir, err := os.MkdirTemp(baseDir, "import-*")
	if err != nil {
		logger.Logger.Errorf("创建云存档导入临时目录失败, err: %v", err)
		fail(c, 500, message.Get(c, "cloud archive import fail"))
		return
	}
	defer func() {
		if removeErr := utils.RemoveDir(tempDir); removeErr != nil {
			logger.Logger.Warnf("清理云存档导入临时目录失败, dir: %s, err: %v", tempDir, removeErr)
		}
	}()

	savePath := filepath.Join(tempDir, "archive.zip")
	file, err := os.OpenFile(savePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		logger.Logger.Errorf("保存云存档失败, err: %v", err)
		fail(c, 500, message.Get(c, "cloud archive import fail"))
		return
	}
	written, copyErr := io.Copy(file, io.LimitReader(body, constant.CloudArchiveMaxBytes+1))
	closeErr := file.Close()
	if copyErr != nil {
		logger.Logger.Errorf("下载云存档失败, err: %v", copyErr)
		fail(c, 201, message.Get(c, "cloud archive download fail"))
		return
	}
	if closeErr != nil {
		logger.Logger.Errorf("保存云存档失败, err: %v", closeErr)
		fail(c, 500, message.Get(c, "cloud archive import fail"))
		return
	}
	if written > constant.CloudArchiveMaxBytes {
		fail(c, 201, message.Get(c, "cloud archive too large"))
		return
	}
	if written == 0 {
		fail(c, 201, message.Get(c, "cloud archive download fail"))
		return
	}

	unzipPath := filepath.Join(tempDir, "unzip") + string(os.PathSeparator)
	roomInfo, code, msgKey, err := h.roomHandler.ImportNewRoomFromZip(c, savePath, unzipPath)
	if err != nil {
		logger.Logger.Errorf("导入云存档失败, archiveID: %s, err: %v", id, err)
		if msgKey == "" {
			msgKey = "cloud archive import fail"
		}
		if code == 0 {
			code = 201
		}
		fail(c, code, message.Get(c, msgKey))
		return
	}

	ok(c, message.Get(c, "cloud archive import success"), roomInfo)
}

// prepareArchive 复制当前存档，清理日志与旧快照后压缩，返回 zip 路径、remark 和清理函数。
func (h *Handler) prepareArchive(c *gin.Context, roomID int) (string, string, func(), error) {
	_, worlds, _, err := dao.FetchGameInfo(roomID)
	if err != nil {
		return "", "", func() {}, err
	}
	if len(*worlds) == 0 {
		return "", "", func() {}, fmt.Errorf("房间没有世界")
	}

	clusterPath := fmt.Sprintf("%s/Cluster_%d", utils.ClusterPath, roomID)
	if !utils.FileDirectoryExists(clusterPath) {
		return "", "", func() {}, fmt.Errorf("存档目录不存在: %s", clusterPath)
	}

	tempBase := constant.CloudTempPath
	if err = os.MkdirAll(tempBase, 0o700); err != nil {
		return "", "", func() {}, err
	}
	tempDir, err := os.MkdirTemp(tempBase, "archive-*")
	if err != nil {
		return "", "", func() {}, err
	}
	cleanup := func() {
		if err := utils.RemoveDir(tempDir); err != nil {
			logger.Logger.Warnf("清理云存档临时目录失败, dir: %s, err: %v", tempDir, err)
		}
	}

	archiveDir := filepath.Join(tempDir, fmt.Sprintf("Cluster_%d", roomID))
	if err = copyDir(clusterPath, archiveDir); err != nil {
		cleanup()
		return "", "", func() {}, err
	}
	if err = removeLogs(archiveDir); err != nil {
		logger.Logger.Warnf("清理云存档日志目录失败, dir: %s, err: %v", archiveDir, err)
	}
	if err = removeLogFiles(archiveDir); err != nil {
		logger.Logger.Warnf("清理云存档日志文件失败, dir: %s, err: %v", archiveDir, err)
	}
	if err = pruneSnapshots(archiveDir, constant.CloudKeepSnapshots); err != nil {
		logger.Logger.Warnf("清理云存档旧快照失败, dir: %s, err: %v", archiveDir, err)
	}

	zipPath := filepath.Join(tempDir, fmt.Sprintf("Cluster_%d_%d.zip", roomID, time.Now().Unix()))
	if err = utils.Zip(archiveDir, zipPath); err != nil {
		cleanup()
		return "", "", func() {}, err
	}

	fileInfo, err := os.Stat(zipPath)
	if err != nil {
		cleanup()
		return "", "", func() {}, err
	}
	if fileInfo.Size() > constant.CloudArchiveMaxBytes {
		cleanup()
		return "", "", func() {}, errArchiveTooLarge
	}

	//game := dst.NewGameController(roomInfo, worlds, roomSetting, c.Request.Header.Get("X-I18n-Lang"))
	remark := buildArchiveRemark(currentPublicIP())

	return zipPath, truncateRunes(remark, 255), cleanup, nil
}

// buildArchiveMetadata 生成云存档元数据，内容是与房间收藏同结构的 XRoomTotalInfo。
// 云端要求该字段必填，用于展示房间名称、世界与模组信息。
func buildArchiveMetadata(roomInfo *models.Room, worlds *[]models.World, roomSetting *models.RoomSetting) (string, error) {
	raw, err := json.Marshal(room.RoomSnapshot(roomInfo, worlds, roomSetting))
	if err != nil {
		return "", err
	}

	return string(raw), nil
}

// func buildArchiveRemark(room *models.Room, worlds *[]models.World, session *dst.RoomSessionInfo, publicIP string) string {
func buildArchiveRemark(s string) string {
	//worldParts := make([]string, 0, len(*worlds))
	//for _, world := range *worlds {
	//	master := ""
	//	if world.IsMaster {
	//		master = "，主世界"
	//	}
	//	worldParts = append(worldParts, fmt.Sprintf("%s(直连 %d%s)", world.WorldName, world.ServerPort, master))
	//}
	//
	//return fmt.Sprintf(
	//	"房间：%s；模组：%d 个；天数：%d；公网IP：%s；世界：%s",
	//	room.GameName,
	//	countMods(room, worlds),
	//	session.Cycles,
	//	publicIP,
	//	strings.Join(worldParts, "、"),
	//)

	return fmt.Sprintf("DMP IP: %s", s)
}

func countMods(room *models.Room, worlds *[]models.World) int {
	mods := make(map[string]struct{})
	for _, mod := range workshopPattern.FindAllString(room.ModData, -1) {
		mods[mod] = struct{}{}
	}
	for _, world := range *worlds {
		for _, mod := range workshopPattern.FindAllString(world.ModData, -1) {
			mods[mod] = struct{}{}
		}
	}

	return len(mods)
}

func currentPublicIP() string {
	if cache.InternetIP != "" {
		return cache.InternetIP
	}
	if ip, err := scheduler.GetInternetIP1(); err == nil && ip != "" {
		cache.InternetIP = ip
		return ip
	}
	if ip, err := scheduler.GetInternetIP2(); err == nil && ip != "" {
		cache.InternetIP = ip
		return ip
	}

	return "未知"
}

func copyDir(source, target string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)

		if info.IsDir() {
			mode := info.Mode().Perm() | 0o700
			if err = os.MkdirAll(destination, mode); err != nil {
				return err
			}
			_ = os.Chtimes(destination, info.ModTime(), info.ModTime())
			return nil
		}
		if !info.Mode().IsRegular() {
			return nil
		}

		if err = os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}

		mode := info.Mode().Perm() | 0o600
		dst, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
		if err != nil {
			_ = src.Close()
			return err
		}
		if _, err = io.Copy(dst, src); err != nil {
			_ = dst.Close()
			_ = src.Close()
			return err
		}
		_ = src.Close()
		if err = dst.Close(); err != nil {
			return err
		}

		_ = os.Chtimes(destination, info.ModTime(), info.ModTime())
		return nil
	})
}

func removeLogs(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		lower := strings.ToLower(info.Name())
		if lower == "server_log" || lower == "server_chat_log" {
			if err = os.RemoveAll(path); err != nil {
				return err
			}
			return filepath.SkipDir
		}

		return nil
	})
}

func pruneSnapshots(root string, keep int) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() || strings.ToLower(info.Name()) != "session" {
			return nil
		}

		entries, err := os.ReadDir(path)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			if err = pruneSessionDir(filepath.Join(path, entry.Name()), keep); err != nil {
				logger.Logger.Warnf("清理快照失败, dir: %s, err: %v", entry.Name(), err)
			}
		}

		return filepath.SkipDir
	})
}

func pruneSessionDir(dir string, keep int) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}

	type snapshot struct {
		name    string
		modTime time.Time
	}
	snapshots := make([]snapshot, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.HasSuffix(strings.ToLower(entry.Name()), ".meta") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		snapshots = append(snapshots, snapshot{name: entry.Name(), modTime: info.ModTime()})
	}

	sort.Slice(snapshots, func(i, j int) bool {
		if snapshots[i].modTime.Equal(snapshots[j].modTime) {
			return snapshots[i].name > snapshots[j].name
		}
		return snapshots[i].modTime.After(snapshots[j].modTime)
	})

	for index := keep; index < len(snapshots); index++ {
		name := snapshots[index].name
		if err = os.Remove(filepath.Join(dir, name)); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err = os.Remove(filepath.Join(dir, name+".meta")); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

// 日志清理还会删除根目录下常见的 .log / 含 log 的 .txt 文件。
func removeLogFiles(root string) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		name := strings.ToLower(info.Name())
		if strings.HasSuffix(name, ".log") || (strings.HasSuffix(name, ".txt") && strings.Contains(name, "log")) {
			if err = os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}

		return nil
	})
}
