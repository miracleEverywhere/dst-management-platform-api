package cloud

import (
	"context"
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/service"
	roomsvc "dst-management-platform-api/service/room"
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
)

// ===== 云存档 =====

var cloudIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

func validCloudID(id string) bool {
	return cloudIDPattern.MatchString(id)
}

func truncateRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}

	return string(runes[:max])
}

// ArchiveUpload 打包当前房间存档并上传到云端
func (s *Service) ArchiveUpload(ctx context.Context, actor service.Actor, roomID int) service.Result {
	if !s.hasRoomPermission(actor, roomID) {
		return service.Fail(constant.CodeCloudArchiveNoPermission)
	}

	link, failed := s.requireLink(actor)
	if failed != nil {
		return *failed
	}
	if err := s.refreshProfile(ctx, link); err != nil {
		if errors.Is(err, errLinkChanged) {
			return service.Fail(constant.CodeCloudLinkNotFound)
		}
		return failCloud(actor, err)
	}
	if link.Plan != "ultimate" {
		return service.Fail(constant.CodeCloudArchiveUltimateOnly)
	}

	zipPath, remark, cleanup, err := s.prepareArchive(roomID)
	if err != nil {
		logger.Logger.Errorf("打包云存档失败, roomID: %d, err: %v", roomID, err)
		if errors.Is(err, errArchiveTooLarge) {
			return service.Fail(constant.CodeCloudArchiveTooLarge)
		}
		return service.Fail(constant.CodeCloudArchiveBuildFail)
	}
	defer cleanup()

	roomInfo, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("查询房间失败, roomID: %d, err: %v", roomID, err)
		return service.Fail(constant.CodeCloudDBError)
	}
	metadata, err := buildArchiveMetadata(roomInfo, worlds, roomSetting)
	if err != nil {
		logger.Logger.Errorf("生成云存档元数据失败, roomID: %d, err: %v", roomID, err)
		return service.Fail(constant.CodeCloudArchiveBuildFail)
	}

	archive, err := s.client.UploadArchive(ctx, link, zipPath, roomInfo.GameName, remark, metadata)
	if err != nil {
		return failCloud(actor, err)
	}
	if err = s.saveLinkTokens(link); err != nil {
		logger.Logger.Errorf("保存云账号 token 失败, err: %v", err)
	}

	return service.Msg("cloud archive upload success", archive)
}

// Archives 云存档列表
func (s *Service) Archives(ctx context.Context, actor service.Actor) service.Result {
	link, failed := s.requireLink(actor)
	if failed != nil {
		return *failed
	}

	result, err := s.client.ListArchives(ctx, link)
	if err != nil {
		return failCloud(actor, err)
	}
	if err = s.saveLinkTokens(link); err != nil {
		logger.Logger.Errorf("保存云账号 token 失败, err: %v", err)
	}

	return service.Success(result)
}

// ArchiveDownload 打开云存档下载流，由 handler 负责写响应头与 body
func (s *Service) ArchiveDownload(ctx context.Context, actor service.Actor, id string) (io.ReadCloser, string, int64, service.Result) {
	if !validCloudID(id) {
		return nil, "", 0, service.Fail(constant.CodeCloudArchiveIDInvalid)
	}

	link, failed := s.requireLink(actor)
	if failed != nil {
		return nil, "", 0, *failed
	}

	body, fileName, size, err := s.client.DownloadArchive(ctx, link, id)
	if err != nil {
		return nil, "", 0, failCloud(actor, err)
	}

	if err = s.saveLinkTokens(link); err != nil {
		logger.Logger.Errorf("保存云账号 token 失败, err: %v", err)
	}

	return body, fileName, size, service.Result{Code: 200}
}

// ArchiveImport 下载云存档、校验大小并导入为一个新房间。
// handler 只负责提取路径参数和输出结果，临时目录、落盘与清理都在此完成。
func (s *Service) ArchiveImport(ctx context.Context, actor service.Actor, id string) service.Result {
	if !validCloudID(id) {
		return service.Fail(constant.CodeCloudArchiveIDInvalid)
	}
	if !s.canCreateRoom(actor) {
		return service.Fail(constant.CodeCloudArchiveCreateNoPermission)
	}
	if s.roomSvc == nil {
		logger.Logger.Error("云存档导入失败：房间导入器未初始化")
		return service.Fail(constant.CodeCloudArchiveImportFail)
	}

	link, failed := s.requireLink(actor)
	if failed != nil {
		return *failed
	}

	body, _, _, err := s.client.DownloadArchive(ctx, link, id)
	if err != nil {
		return failCloud(actor, err)
	}
	defer body.Close()

	if err = s.saveLinkTokens(link); err != nil {
		logger.Logger.Errorf("保存云账号 token 失败, err: %v", err)
	}

	baseDir := filepath.Join(constant.CloudTempPath, "import")
	if err = os.MkdirAll(baseDir, 0o700); err != nil {
		logger.Logger.Errorf("创建云存档导入目录失败, err: %v", err)
		return service.Fail(constant.CodeCloudArchiveImportFail)
	}
	tempDir, err := os.MkdirTemp(baseDir, "import-*")
	if err != nil {
		logger.Logger.Errorf("创建云存档导入临时目录失败, err: %v", err)
		return service.Fail(constant.CodeCloudArchiveImportFail)
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
		return service.Fail(constant.CodeCloudArchiveImportFail)
	}
	written, copyErr := io.Copy(file, io.LimitReader(body, constant.CloudArchiveMaxBytes+1))
	closeErr := file.Close()
	if copyErr != nil {
		logger.Logger.Errorf("下载云存档失败, err: %v", copyErr)
		return service.Fail(constant.CodeCloudArchiveDownloadFail)
	}
	if closeErr != nil {
		logger.Logger.Errorf("保存云存档失败, err: %v", closeErr)
		return service.Fail(constant.CodeCloudArchiveImportFail)
	}
	if written > constant.CloudArchiveMaxBytes {
		return service.Fail(constant.CodeCloudArchiveTooLarge)
	}
	if written == 0 {
		return service.Fail(constant.CodeCloudArchiveDownloadFail)
	}

	unzipPath := filepath.Join(tempDir, "unzip") + string(os.PathSeparator)
	roomInfo, code, err := s.roomSvc.ImportNewRoomFromZip(actor, savePath, unzipPath)
	if err != nil {
		logger.Logger.Errorf("导入云存档失败, archiveID: %s, err: %v", id, err)
		if code == 0 {
			code = constant.CodeCloudArchiveImportFail
		}
		return service.Fail(code)
	}

	return service.Msg("cloud archive import success", roomInfo)
}

var workshopPattern = regexp.MustCompile(`workshop-\d+`)

var errArchiveTooLarge = errors.New("cloud archive too large")

// prepareArchive 复制当前存档，清理日志与旧快照后压缩，返回 zip 路径、remark 和清理函数。
func (s *Service) prepareArchive(roomID int) (string, string, func(), error) {
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

	remark := buildArchiveRemark(currentPublicIP())

	return zipPath, truncateRunes(remark, 255), cleanup, nil
}

// buildArchiveMetadata 生成云存档元数据，内容是与房间收藏同结构的 XRoomTotalInfo。
// 云端要求该字段必填，用于展示房间名称、世界与模组信息。
func buildArchiveMetadata(roomInfo *models.Room, worlds *[]models.World, roomSetting *models.RoomSetting) (string, error) {
	raw, err := json.Marshal(roomsvc.RoomSnapshot(roomInfo, worlds, roomSetting))
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
