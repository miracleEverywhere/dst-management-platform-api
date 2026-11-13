package cloud

import (
	"dst-management-platform-api/cache"
	"dst-management-platform-api/constant"
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/scheduler"
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
