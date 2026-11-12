package logs

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/dst"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/service"
	"dst-management-platform-api/utils"
	"fmt"
	nethttp "net/http"
	"os"
	"path/filepath"
	"strings"
)

// Service 日志模块业务实现，不依赖 gin
type Service struct {
	userDao *dao.UserDAO
}

// New 创建日志模块 service
func New(userDao *dao.UserDAO) *Service {
	return &Service{
		userDao: userDao,
	}
}

// hasPermission 判断当前用户是否有房间权限
func (s *Service) hasPermission(actor service.Actor, roomID string) bool {
	// 管理员直接返回true
	if actor.IsAdmin() {
		return true
	}

	dbUser, err := s.userDao.GetUserByUsername(actor.Username)
	if err != nil {
		logger.Logger.Error("查询数据库失败")
		return false
	}
	roomIDs := strings.Split(dbUser.Rooms, ",")
	for _, id := range roomIDs {
		if id == roomID {
			return true
		}
	}

	return false
}

// Content 读取日志内容：game/chat 走房间日志，其余为系统级日志且仅管理员可见
func (s *Service) Content(actor service.Actor, roomID, worldID int, logType string, lines int) service.Result {
	if !s.hasPermission(actor, fmt.Sprintf("%d", roomID)) {
		return service.Fail(constant.CodeLogsNoPermission)
	}

	if logType == "game" || logType == "chat" {
		if roomID == 0 {
			return service.Fail(constant.CodeLogsBadRequest)
		}

		room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
		if err != nil {
			logger.Logger.Errorf("获取基本信息失败, err: %v", err)
			return service.Fail(constant.CodeLogsDBError)
		}

		game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
		return service.Success(game.LogContent(logType, worldID, lines))
	}

	// 系统级日志（steam、access、runtime）仅允许管理员查看
	if !actor.IsAdmin() {
		return service.Fail(constant.CodeLogsNoPermission)
	}

	var logPath string
	switch logType {
	case "steam":
		if cache.OsType == constant.Darwin {
			logPath = constant.DarwinSteamLogPath
		} else {
			logPath = constant.SteamLogPath
		}
	case "access":
		logPath = "logs/access.log"
	case "runtime":
		logPath = "logs/runtime.log"
	}

	return service.Success(utils.GetFileLastNLines(logPath, lines))
}

// HistoryList 历史日志文件列表
func (s *Service) HistoryList(actor service.Actor, roomID, worldID int, logType string) service.Result {
	if !s.hasPermission(actor, fmt.Sprintf("%d", roomID)) {
		return service.Fail(constant.CodeLogsNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeLogsDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	return service.Success(game.HistoryFileList(logType, worldID))
}

// HistoryContent 历史日志文件内容
func (s *Service) HistoryContent(actor service.Actor, roomID, worldID int, logType, logFile string) service.Result {
	if !s.hasPermission(actor, fmt.Sprintf("%d", roomID)) {
		return service.Fail(constant.CodeLogsNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeLogsDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	return service.Success(game.HistoryFileContent(logType, logFile, worldID))
}

// CleanInfo 日志占用信息
func (s *Service) CleanInfo(actor service.Actor, roomID int) service.Result {
	if !s.hasPermission(actor, fmt.Sprintf("%d", roomID)) {
		return service.Fail(constant.CodeLogsNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeLogsDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	return service.Success(game.LogsInfo())
}

// Clean 清理日志，系统级日志仅管理员可清理
func (s *Service) Clean(actor service.Actor, roomID int, clean dst.CleanLogs) service.Result {
	if !s.hasPermission(actor, fmt.Sprintf("%d", roomID)) {
		return service.Fail(constant.CodeLogsNoPermission)
	}

	if (clean.Steam || clean.Access || clean.Runtime) && !actor.IsAdmin() {
		return service.Fail(constant.CodeLogsNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return service.Fail(constant.CodeLogsDBError)
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	if game.LogsClean(&clean) {
		return service.Msg("delete success", nil)
	}

	return service.Fail(constant.CodeLogsDeleteFail)
}

// Download 打包日志并返回压缩包路径，调用方负责在响应结束后执行 cleanup
func (s *Service) Download(actor service.Actor, roomID int) (zipFile string, cleanup func(), result service.Result) {
	noop := func() {}

	if !s.hasPermission(actor, fmt.Sprintf("%d", roomID)) {
		return "", noop, service.Fail(constant.CodeLogsNoPermission)
	}

	room, worlds, roomSetting, err := dao.FetchGameInfo(roomID)
	if err != nil {
		logger.Logger.Errorf("获取基本信息失败, err: %v", err)
		return "", noop, service.Result{
			Code:   constant.CodeLogsDBError,
			Status: nethttp.StatusInternalServerError,
		}
	}

	game := dst.NewGameController(room, worlds, roomSetting, actor.Lang)
	fileList := game.LogsList(actor.IsAdmin())

	// 每次下载使用独立临时目录：同房间并发下载时不能共用 logs.zip，
	// 否则会互相覆盖/删除，甚至把管理员日志发给普通用户。
	baseDir := fmt.Sprintf("%s/tmp", constant.DmpFiles)
	if errEnsure := utils.EnsureDirExists(baseDir); errEnsure != nil {
		logger.Logger.Errorf("创建目录失败, err: %v", errEnsure)
		return "", noop, service.Result{
			Code:   constant.CodeLogsDownloadFail,
			Status: nethttp.StatusInternalServerError,
		}
	}
	zipDir, errTemp := os.MkdirTemp(baseDir, fmt.Sprintf("%d-*", roomID))
	if errTemp != nil {
		logger.Logger.Errorf("创建临时目录失败, err: %v", errTemp)
		return "", noop, service.Result{
			Code:   constant.CodeLogsDownloadFail,
			Status: nethttp.StatusInternalServerError,
		}
	}
	cleanup = func() {
		if errRemove := utils.RemoveDir(zipDir); errRemove != nil {
			logger.Logger.Error(errRemove.Error())
		}
	}

	zipFile = filepath.Join(zipDir, "logs.zip")
	if errZip := utils.ZipFiles(fileList, zipFile); errZip != nil {
		logger.Logger.Errorf("创建压缩文件失败, err: %v", errZip)
		return "", cleanup, service.Result{
			Code:   constant.CodeLogsDownloadFail,
			Status: nethttp.StatusInternalServerError,
		}
	}

	return zipFile, cleanup, service.Result{Code: 200}
}
