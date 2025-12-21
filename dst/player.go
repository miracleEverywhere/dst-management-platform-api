package dst

import (
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"fmt"
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
		logger.Logger.Debug(uid)
		if uid == "" || strings.HasPrefix(uid, " ") {
			continue
		}

		uidList = append(uidList, uid)
	}

	logger.Logger.Debug(utils.StructToFlatString(uidList))
	return uidList
}

func (g *Game) savePlayerList() error {
	var err error
	err = utils.WriteLinesFromSlice(g.adminlistPath, g.adminlist)
	if err != nil {
		return err
	}
	err = utils.WriteLinesFromSlice(g.blocklistPath, g.blocklist)
	if err != nil {
		return err
	}
	err = utils.WriteLinesFromSlice(g.whitelistPath, g.whitelist)
	if err != nil {
		return err
	}

	return nil
}

func (g *Game) addPlayerList(uid, listType string) error {
	switch listType {
	case "adminlist":
		if utils.Contains(g.playerSaveData.adminlist, uid) {
			return nil
		}
		g.playerSaveData.adminlist = append(g.playerSaveData.adminlist, uid)
		return g.savePlayerList()
	case "blocklist":
		if utils.Contains(g.playerSaveData.blocklist, uid) {
			return nil
		}
		g.playerSaveData.blocklist = append(g.playerSaveData.blocklist, uid)
		return g.savePlayerList()
	case "whitelist":
		if utils.Contains(g.playerSaveData.whitelist, uid) {
			return nil
		}
		g.playerSaveData.whitelist = append(g.playerSaveData.whitelist, uid)
		err := g.savePlayerList()
		if err != nil {
			return err
		}
		return g.createRoom() // 不统一处理，提升性能
	}

	return fmt.Errorf("类型错误")
}

func (g *Game) removePlayerList(uid, listType string) error {
	switch listType {
	case "adminlist":
		if !utils.Contains(g.playerSaveData.adminlist, uid) {
			return nil
		}
		g.playerSaveData.adminlist = utils.RemoveItem(g.playerSaveData.adminlist, uid)
		return g.savePlayerList()
	case "blocklist":
		if !utils.Contains(g.playerSaveData.blocklist, uid) {
			return nil
		}
		g.playerSaveData.blocklist = utils.RemoveItem(g.playerSaveData.blocklist, uid)
		return g.savePlayerList()
	case "whitelist":
		if !utils.Contains(g.playerSaveData.whitelist, uid) {
			return nil
		}
		g.playerSaveData.whitelist = utils.RemoveItem(g.playerSaveData.whitelist, uid)
		err := g.savePlayerList()
		if err != nil {
			return err
		}
		return g.createRoom() // 不统一处理，提升性能
	}

	return fmt.Errorf("类型错误")
}
