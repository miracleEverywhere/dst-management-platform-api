package tools

import (
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// BackupList 获取房间的备份文件列表
func (s *Service) BackupList(actor service.Actor, roomID int) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodeToolsBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeToolsNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeToolsDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	backups, err := game.GetBackups()
	if err != nil {
		logger.Logger.Errorf("获取备份文件失败, err: %v", err)
		return service.Fail(constant.CodeToolsGetBackupFail)
	}

	return service.Success(backups)
}

// BackupCreate 创建备份
func (s *Service) BackupCreate(actor service.Actor, roomID int) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodeToolsBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeToolsNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeToolsDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	err = game.Backup()
	if err != nil {
		logger.Logger.Errorf("创建备份文件失败, err: %v", err)
		return service.Fail(constant.CodeToolsCreateBackupFail)
	}

	return service.Msg("create backup success", nil)
}

// BackupDelete 删除备份文件，返回实际删除的数量
func (s *Service) BackupDelete(actor service.Actor, roomID int, filenames []string) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodeToolsBadRequest)
	}

	for _, filename := range filenames {
		if !utils.IsSafePath(filename) {
			logger.Logger.Warnf("检测到穿越攻击: %s, api: %s", filename, actor.Path)
			return service.Fail(constant.CodeToolsBadRequest)
		}
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeToolsNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeToolsDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	count := game.DeleteBackups(filenames)

	return service.Msg("delete success", count)
}

// BackupRestore 恢复备份，并回写房间、世界与房间配置
func (s *Service) BackupRestore(actor service.Actor, roomID int, filename string) service.Result {
	if roomID == 0 {
		return service.Fail(constant.CodeToolsBadRequest)
	}

	if !utils.IsSafeString(filename) {
		return service.Fail(constant.CodeToolsBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return service.Fail(constant.CodeToolsNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeToolsDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	saveData, err := game.Restore(filename)
	if err != nil {
		logger.Logger.Errorf("恢复失败, err: %v", err)
		return service.Fail(constant.CodeToolsRestoreFail)
	}

	err = s.roomDao.UpdateRoom(&saveData.Room)
	if err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		return service.Fail(constant.CodeToolsRestoreFail)
	}

	err = s.worldDao.UpdateWorlds(&saveData.Worlds)
	if err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		return service.Fail(constant.CodeToolsRestoreFail)
	}

	err = s.roomSettingDao.UpdateRoomSetting(&saveData.RoomSetting)
	if err != nil {
		logger.Logger.Errorf("更新房间失败, err: %v", err)
		return service.Fail(constant.CodeToolsRestoreFail)
	}
	if err = s.aiManager.Reload(roomID); err != nil {
		logger.Logger.Errorf("重载房间 AI 对话失败, roomID: %d, err: %v", roomID, err)
	}

	return service.Msg("restore success", nil)
}

// BackupDownload 计算备份下载的文件路径与下载文件名。
// 返回的 result.Code 为 200 表示可以下载，调用方据此写 Content-Disposition 并发送文件；
// 其余 code 表示校验失败，直接把 result 翻译成响应即可。
func (s *Service) BackupDownload(actor service.Actor, roomID int, filename string) (filePath string, fileName string, result service.Result) {
	if roomID == 0 || filename == "" {
		return "", "", service.Fail(constant.CodeToolsBadRequest)
	}

	// 安全验证（防止路径遍历）
	if !utils.IsSafePath(filename) {
		return "", "", service.Fail(constant.CodeToolsBadRequest)
	}

	if !s.hasPermission(actor, strconv.Itoa(roomID)) {
		return "", "", service.Fail(constant.CodeToolsNoPermission)
	}

	filePath = fmt.Sprintf("dmp_files/backup/%d/%s", roomID, filename)
	fileName = filename
	base64Str := strings.TrimSuffix(fileName, ".zip")
	decodeFilename, err := utils.Base64Decode(base64Str)
	if err != nil {
		return filePath, fileName, service.Result{Code: 200}
	}
	// 房间名，房间天数，备份时间
	decodeFilenameParts := strings.Split(decodeFilename, "<-@dmp@->")
	if len(decodeFilenameParts) != 3 {
		return filePath, fileName, service.Result{Code: 200}
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
		return filePath, fileName, service.Result{Code: 200}
	}
	ts := time.UnixMilli(ms)
	tsStr := ts.Format("2006-01-02-15-04")
	fileName = fmt.Sprintf("%s_%s_%s.zip", roomName, cycle, tsStr)

	return filePath, fileName, service.Result{Code: 200}
}
