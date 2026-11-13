package cloud

import (
	"context"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"errors"
	"io"
	"os"
	"path/filepath"
	"regexp"
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
