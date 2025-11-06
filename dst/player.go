package dst

import (
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"strings"
)

type playerSaveData struct {
	whitelist     []string
	blocklist     []string
	adminlist     []string
	whitelistPath string
	blocklistPath string
	adminlistPath string
}

func getPlayerList(filepath string) []string {
	// 预留位 黑名单 管理员
	err := utils.EnsureFileExists(filepath)
	if err != nil {
		logger.Logger.Error("创建文件失败", "err", err, "file", filepath)
		return []string{}
	}
	al, err := utils.ReadLinesToSlice(filepath)
	if err != nil {
		logger.Logger.Error("读取文件失败", "err", err, "file", filepath)
		return []string{}
	}
	var uidList []string
	for _, uid := range al {
		if !(uid == "" || strings.HasPrefix(uid, " ")) {
			uidList = append(uidList, uid)
		}
	}

	return uidList
}
