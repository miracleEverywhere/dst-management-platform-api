package setting

import (
	"dst-management-platform-api/app/externalApi"
	"dst-management-platform-api/scheduler"
	"dst-management-platform-api/utils"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/tealeg/xlsx"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func handleRoomSettingGet(c *gin.Context) {
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}
	type Response struct {
		Code    int               `json:"code"`
		Message string            `json:"message"`
		Data    utils.RoomSetting `json:"data"`
	}
	response := Response{
		Code:    200,
		Message: "success",
		Data:    config.RoomSetting,
	}
	c.JSON(http.StatusOK, response)
}

func handleRoomSettingSavePost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var roomSetting utils.RoomSetting
	if err := c.ShouldBindJSON(&roomSetting); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	config.RoomSetting = roomSetting

	// 配置单服务器节点数据库
	if !config.MultiHost {
		config.RoomSetting.Base.ShardMasterIp = "127.0.0.1"
		config.RoomSetting.Base.ShardMasterPort = 10888
		config.RoomSetting.Base.ClusterKey = "supersecretkey"
		config.RoomSetting.Base.SteamMasterPort = 27018
		config.RoomSetting.Base.SteamAuthenticationPort = 8768
	}

	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	err = saveSetting(config)
	if err != nil {
		utils.Logger.Error("房间配置保存失败", "err", err)
	}
	err = DstModsSetup()
	if err != nil {
		utils.Logger.Error("mod配置保存失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("saveFail", langStr), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("saveSuccess", langStr), "data": nil})
}

func handleRoomSettingSaveAndRestartPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var roomSetting utils.RoomSetting
	if err := c.ShouldBindJSON(&roomSetting); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	config.RoomSetting = roomSetting

	// 配置单服务器节点数据库
	if !config.MultiHost {
		config.RoomSetting.Base.ShardMasterIp = "127.0.0.1"
		config.RoomSetting.Base.ShardMasterPort = 10888
		config.RoomSetting.Base.ClusterKey = "supersecretkey"
		config.RoomSetting.Base.SteamMasterPort = 27018
		config.RoomSetting.Base.SteamAuthenticationPort = 8768
	}

	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	err = saveSetting(config)
	if err != nil {
		utils.Logger.Error("房间配置保存失败", "err", err)
	}
	err = DstModsSetup()
	if err != nil {
		utils.Logger.Error("mod配置保存失败", "err", err)
	}

	err = utils.StopGame()
	if err != nil {
		utils.Logger.Error("关闭游戏失败", "err", err)
	}
	err = utils.StartGame()
	if err != nil {
		utils.Logger.Error("启动游戏失败", "err", err)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("restartSuccess", langStr), "data": nil})
}

func handleRoomSettingSaveAndGeneratePost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var roomSetting utils.RoomSetting
	if err := c.ShouldBindJSON(&roomSetting); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	config.RoomSetting = roomSetting

	// 配置单服务器节点数据库
	if !config.MultiHost {
		config.RoomSetting.Base.ShardMasterIp = "127.0.0.1"
		config.RoomSetting.Base.ShardMasterPort = 10888
		config.RoomSetting.Base.ClusterKey = "supersecretkey"
		config.RoomSetting.Base.SteamMasterPort = 27018
		config.RoomSetting.Base.SteamAuthenticationPort = 8768
	}

	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	err = saveSetting(config)
	if err != nil {
		utils.Logger.Error("房间配置保存失败", "err", err)
	}
	err = DstModsSetup()
	if err != nil {
		utils.Logger.Error("mod配置保存失败", "err", err)
	}
	generateWorld()

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("generateSuccess", langStr), "data": nil})

}

func handlePlayerListGet(c *gin.Context) {
	type PlayersInfo struct {
		UID      string `json:"uid"`
		NickName string `json:"nickName"`
		Prefab   string `json:"prefab"`
		Age      int    `json:"age"`
	}
	type PlayerList struct {
		Players   []PlayersInfo          `json:"players"`
		AdminList []string               `json:"adminList"`
		BlockList []string               `json:"blockList"`
		WhiteList []string               `json:"whiteList"`
		UidMap    map[string]interface{} `json:"uidMap"`
	}

	//config, err := utils.ReadConfig()
	//if err != nil {
	//	utils.Logger.Error("配置文件读取失败", "err", err)
	//	utils.RespondWithError(c, 500, "zh")
	//	return
	//}
	adminList := getList(utils.AdminListPath)
	blockList := getList(utils.BlockListPath)
	whiteList := getList(utils.WhiteListPath)

	uidMap, _ := utils.ReadUidMap()

	var (
		playList PlayerList
		players  []utils.Players
	)

	//playList.Players = config.Players
	if len(utils.STATISTICS) > 0 {
		players = utils.STATISTICS[len(utils.STATISTICS)-1].Players
	}
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("读取配置文件失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	var world string

	if config.RoomSetting.Ground != "" {
		world = "Master"
	} else {
		world = "Caves"
	}

	userPathEncode, _ := GetUserDataEncodeStatus("KU_12345678", world)

	for _, player := range players {
		uid := player.UID
		age, _, err := GetPlayerAgePrefab(uid, world, userPathEncode)
		if err != nil {
			utils.Logger.Error("玩家游戏时长获取失败")
		}
		var playerInfo PlayersInfo
		playerInfo.UID = uid
		playerInfo.NickName = player.NickName
		playerInfo.Prefab = player.Prefab
		playerInfo.Age = age

		playList.Players = append(playList.Players, playerInfo)
	}

	playList.AdminList = adminList
	playList.BlockList = blockList
	playList.WhiteList = whiteList
	playList.UidMap = uidMap

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": playList})
}

func handleHistoryPlayerGet(c *gin.Context) {
	type Player struct {
		UID      string      `json:"uid"`
		Nickname interface{} `json:"nickname"`
		Prefab   string      `json:"prefab"`
		Age      int         `json:"age"`
	}

	uidMap, _ := utils.ReadUidMap()
	if len(uidMap) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": []Player{}})
		return
	}

	var world string

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("读取配置文件失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}
	if config.RoomSetting.Ground != "" {
		world = "Master"
	} else {
		world = "Caves"
	}

	userPathEncode, _ := GetUserDataEncodeStatus("KU_12345678", world)

	var playerList []Player
	for uid, nickname := range uidMap {
		age, prefab, err := GetPlayerAgePrefab(uid, world, userPathEncode)
		if err != nil {
			utils.Logger.Error("获取历史玩家信息失败", "err", err, "UID", uid)
		}
		var player Player
		player.UID = uid
		player.Nickname = nickname
		player.Age = age
		player.Prefab = prefab
		playerList = append(playerList, player)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": playerList})
}

func handleAdminAddPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var uidFrom UIDForm
	if err := c.ShouldBindJSON(&uidFrom); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := addList(uidFrom.UID, utils.AdminListPath)
	if err != nil {
		utils.Logger.Error("添加管理员失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("addAdminFail", langStr), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("addAdmin", langStr), "data": nil})
}

func handleBlockAddPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var uidFrom UIDForm
	if err := c.ShouldBindJSON(&uidFrom); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := addList(uidFrom.UID, utils.BlockListPath)
	if err != nil {
		utils.Logger.Error("添加黑名单失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("addBlockFail", langStr), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("addBlock", langStr), "data": nil})
}

func handleBlockUpload(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("uploadFail", langStr), "data": nil})
		return
	}
	//保存文件
	savePath := utils.ImportFileUploadPath + file.Filename
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		utils.Logger.Error("文件保存失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("uploadFail", langStr), "data": nil})
		return
	}

	// 打开Excel文件
	xlsFile, err := xlsx.OpenFile(savePath)
	if err != nil {
		utils.Logger.Error("无法打开文件: %s", err)
	}

	blockList := getList(utils.BlockListPath)

	// 遍历所有工作表
	for _, sheet := range xlsFile.Sheets {
		// 遍历工作表中的所有行
		for _, row := range sheet.Rows {
			// 获取A列（索引为0）的单元格
			if len(row.Cells) > 0 {
				cell := row.Cells[0]
				// 将单元格的值添加到字符串切片中
				blockList = append(blockList, cell.String())
			}
		}
	}

	blockList = utils.UniqueSliceKeepOrderString(blockList)
	str := strings.Join(blockList, "\n")
	err = utils.TruncAndWriteFile(utils.BlockListPath, str)

	_ = utils.BashCMD("rm -f " + savePath)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("uploadSuccess", langStr), "data": nil})
}

func handleWhiteAddPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var uidFrom UIDForm
	if err := c.ShouldBindJSON(&uidFrom); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := addList(uidFrom.UID, utils.WhiteListPath)
	if err != nil {
		utils.Logger.Error("添加白名单失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("addWhiteFail", langStr), "data": nil})
		return
	}
	err = changeWhitelistSlots()
	if err != nil {
		utils.Logger.Error("添加白名单失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("addWhiteFail", langStr), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("addWhite", langStr), "data": nil})
}

func handleAdminDeletePost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var uidFrom UIDForm
	if err := c.ShouldBindJSON(&uidFrom); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := deleteList(uidFrom.UID, utils.AdminListPath)
	if err != nil {
		utils.Logger.Error("删除管理员失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("deleteAdminFail", langStr), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("deleteAdmin", langStr), "data": nil})
}

func handleBlockDeletePost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var uidFrom UIDForm
	if err := c.ShouldBindJSON(&uidFrom); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := deleteList(uidFrom.UID, utils.BlockListPath)
	if err != nil {
		utils.Logger.Error("删除黑名单失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("deleteBlockFail", langStr), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("deleteBlock", langStr), "data": nil})
}

func handleWhiteDeletePost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var uidFrom UIDForm
	if err := c.ShouldBindJSON(&uidFrom); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := deleteList(uidFrom.UID, utils.WhiteListPath)
	if err != nil {
		utils.Logger.Error("删除白名单失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("deleteWhiteFail", langStr), "data": nil})
		return
	}
	err = changeWhitelistSlots()
	if err != nil {
		utils.Logger.Error("删除白名单失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("addWhiteFail", langStr), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("deleteWhite", langStr), "data": nil})
}

func handleKick(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var uidFrom UIDForm
	if err := c.ShouldBindJSON(&uidFrom); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	errMaster := kick(uidFrom.UID, utils.MasterName)
	errCaves := kick(uidFrom.UID, utils.CavesName)

	if errMaster != nil && errCaves != nil {
		utils.Logger.Error("踢出玩家失败", "errMaster", errMaster, "errCaves", errCaves)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("kickFail", langStr), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("kickSuccess", langStr), "data": nil})
}

func handleImportFileUploadPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("uploadFail", langStr), "data": nil})
		return
	}
	//保存文件
	savePath := utils.ImportFileUploadPath + file.Filename
	if err := c.SaveUploadedFile(file, savePath); err != nil {
		utils.Logger.Error("文件保存失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("uploadFail", langStr), "data": nil})
		return
	}
	//检查导入文件是否合法
	result, err := checkZipFile(file.Filename)
	if err != nil {
		utils.Logger.Error("检查导入文件失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("wrongUploadFile", langStr), "data": nil})
		return
	}
	if !result {
		utils.Logger.Error("导入文件校验失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("wrongUploadFile", langStr), "data": nil})
		return
	}
	//关闭服务器
	_ = utils.StopGame()
	//备份服务器
	err = utils.BackupGame()
	if err != nil {
		utils.Logger.Warn("游戏备份失败", "err", err)
	}
	//删除旧服务器文件
	err = utils.BashCMD("rm -rf " + utils.ServerPath + "*")
	if err != nil {
		utils.Logger.Error("删除旧服务器文件失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("deleteOldServerFail", langStr), "data": nil})
		return
	}
	//创建新服务器文件
	err = utils.BashCMD("mv " + utils.ImportFileUnzipPath + "* " + utils.ServerPath)
	if err != nil {
		utils.Logger.Error("创建新服务器文件失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("createNewServerFail", langStr), "data": nil})
		return
	}
	//写入数据库
	err = WriteDatabase()
	if err != nil {
		utils.Logger.Error("导入文件写入数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("writeToDBFail", langStr), "data": nil})
		return
	}
	//清理上传的文件
	clearUpZipFile()
	// 写入dedicated_server_mods_setup.lua
	err = DstModsSetup()
	if err != nil {
		utils.Logger.Error("mod配置保存失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("saveFail", langStr), "data": nil})
		return
	}
	err = changeWhitelistSlots()
	if err != nil {
		utils.Logger.Error("配置白名单失败", "err", err)
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("uploadSuccess", langStr), "data": nil})
}

func handleModSettingFormatGet(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	if config.RoomSetting.Ground == "" && config.RoomSetting.Cave == "" {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": nil})
		return
	}

	var luaScript string

	if config.RoomSetting.Ground == "" {
		luaScript, _ = utils.GetFileAllContent(utils.CavesModPath)
	} else {
		luaScript, _ = utils.GetFileAllContent(utils.MasterModPath)
	}

	modInfo, err := externalApi.GetModsInfo(luaScript, langStr)
	if err != nil {
		utils.RespondWithError(c, 500, langStr)
		return
	}

	var responseData []utils.ModFormattedData
	for _, i := range utils.ModOverridesToStruct(luaScript) {
		item := utils.ModFormattedData{
			ID: i.ID,
			Name: func() string {
				for _, j := range modInfo {
					if i.ID == j.ID {
						return j.Name
					}
				}
				return ""
			}(),
			Enable:               i.Enabled,
			ConfigurationOptions: i.ConfigurationOptions,
			FileUrl: func() string {
				for _, j := range modInfo {
					if i.ID == j.ID {
						return j.FileUrl
					}
				}
				return ""
			}(),
			PreviewUrl: func() string {
				for _, j := range modInfo {
					if i.ID == j.ID {
						return j.PreviewUrl
					}
				}
				return ""
			}(),
		}
		responseData = append(responseData, item)
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": responseData})
}

func handleModConfigOptionsGet(c *gin.Context) {
	type ModConfigurationsForm struct {
		ID int `form:"id" json:"id"`
	}
	var modConfigurationsForm ModConfigurationsForm
	if err := c.ShouldBindQuery(&modConfigurationsForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	type ModConfig struct {
		ID            int                         `json:"id"`
		ConfigOptions []utils.ConfigurationOption `json:"configOptions"`
	}

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	var (
		modConfig      ModConfig
		modInfoLuaFile string
	)

	modID := modConfigurationsForm.ID

	if modID == 1 {
		// 禁用客户端模组配置
		modConfig.ID = 1
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": modConfig})
		return
	}

	if config.RoomSetting.Ground != "" {
		modInfoLuaFile = utils.MasterModUgcPath + "/" + strconv.Itoa(modID) + "/modinfo.lua"
	} else {
		modInfoLuaFile = utils.CavesModUgcPath + "/" + strconv.Itoa(modID) + "/modinfo.lua"
	}

	isUgcMod, err := utils.FileDirectoryExists(modInfoLuaFile)
	if err != nil {
		utils.RespondWithError(c, 500, langStr)
		return
	}

	if !isUgcMod {
		modInfoLuaFile = utils.ModNoUgcPath + "/workshop-" + strconv.Itoa(modID) + "/modinfo.lua"
		exist, err := utils.FileDirectoryExists(modInfoLuaFile)
		if err != nil {
			utils.RespondWithError(c, 500, langStr)
			return
		}
		if !exist {
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("needDownload", langStr), "data": nil})
			return
		}
	}

	luaScript, _ := utils.GetFileAllContent(modInfoLuaFile)
	modConfig.ID = modID
	modConfig.ConfigOptions = utils.GetModConfigOptions(luaScript, langStr)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": modConfig})
}

func handleModConfigChangePost(c *gin.Context) {
	type ModFormattedDataForm struct {
		ModFormattedData []utils.ModFormattedData `json:"modFormattedData"`
	}
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	var modFormattedDataForm ModFormattedDataForm
	if err := c.ShouldBindJSON(&modFormattedDataForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	luaString := utils.ParseToLua(modFormattedDataForm.ModFormattedData)

	config.RoomSetting.Mod = luaString
	// Master/modoverrides.lua
	err = utils.TruncAndWriteFile(utils.MasterModPath, config.RoomSetting.Mod)
	if err != nil {
		utils.Logger.Error("MOD配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	if config.RoomSetting.Cave != "" {
		//Caves/modoverrides.lua
		err = utils.TruncAndWriteFile(utils.CavesModPath, config.RoomSetting.Mod)
		if err != nil {
			utils.Logger.Error("MOD配置文件写入失败", "err", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}
	}

	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("configUpdateSuccess", langStr), "data": nil})
}

func handleModDownloadPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	type ModDownloadForm struct {
		ID      int    `json:"id"`
		FileURL string `json:"file_url"`
	}
	var modDownloadForm ModDownloadForm
	if err := c.ShouldBindJSON(&modDownloadForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	go func() {
		if modDownloadForm.FileURL == "" {
			cmd := utils.GenerateModDownloadCMD(modDownloadForm.ID)
			err := utils.BashCMD(cmd)
			if err != nil {
				utils.Logger.Error("MOD下载失败", "err", err)
			}
		} else {
			err := externalApi.DownloadMod(modDownloadForm.FileURL, modDownloadForm.ID)
			if err != nil {
				utils.Logger.Error("MOD下载失败", "err", err)
			}
		}
	}()

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("downloading", langStr), "data": nil})
}

func handleSyncModPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	err := utils.SyncMods()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("syncModFail", langStr), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("syncModSuccess", langStr), "data": nil})
}

func handleDeleteDownloadedModPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	type DeleteForm struct {
		ISUGC bool `json:"isUgc"`
		ID    int  `json:"id"`
	}

	var deleteForm DeleteForm
	if err := c.ShouldBindJSON(&deleteForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := utils.DeleteDownloadedMod(deleteForm.ISUGC, deleteForm.ID)
	if err != nil {
		utils.Logger.Error("删除已下载的MOD失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("deleteModFail", langStr), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("deleteModSuccess", langStr), "data": nil})
}

func handleEnableModPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	type EnableForm struct {
		ISUGC bool `json:"isUgc"`
		ID    int  `json:"id"`
	}

	var enableForm EnableForm
	if err := c.ShouldBindJSON(&enableForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		utils.Logger.Error("无法获取 home 目录", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	// 读取modinfo.lua
	var (
		modInfoLuaFile   string
		modDirPath       string
		modFormattedData []utils.ModFormattedData
	)

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	if config.RoomSetting.Base.Name == "" {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("gameServerNotCreated", langStr), "data": []string{}})
		return
	}

	// 复制mod文件至指定的dst目录
	if enableForm.ISUGC {
		modDirPath = homeDir + "/" + utils.ModDownloadPath + "/steamapps/workshop/content/322330/" + strconv.Itoa(enableForm.ID)
		modInfoLuaFile = modDirPath + "/modinfo.lua"
		// MacOS 不执行复制
		if config.Platform != "darwin" {
			if config.RoomSetting.Ground != "" {
				err = utils.RemoveDir(utils.MasterModUgcPath + "/" + strconv.Itoa(enableForm.ID))
				if err != nil {
					utils.Logger.Error("删除旧MOD文件失败", "err", err)
				}
				cmdMaster := "cp -r " + modDirPath + " " + utils.MasterModUgcPath + "/"
				err := utils.BashCMD(cmdMaster)
				if err != nil {
					utils.Logger.Error("复制MOD文件失败", "err", err, "cmd", cmdMaster)
				}
			}
			if config.RoomSetting.Cave != "" {
				err = utils.RemoveDir(utils.CavesModUgcPath + "/" + strconv.Itoa(enableForm.ID))
				if err != nil {
					utils.Logger.Error("删除旧MOD文件失败", "err", err)
				}
				cmdCaves := "cp -r " + modDirPath + " " + utils.CavesModUgcPath + "/"
				err = utils.BashCMD(cmdCaves)
				if err != nil {
					utils.Logger.Error("复制MOD文件失败", "err", err, "cmd", cmdCaves)
				}
			}
		}

	} else {
		modDirPath = homeDir + "/" + utils.ModDownloadPath + "/not_ugc/" + strconv.Itoa(enableForm.ID)
		modInfoLuaFile = modDirPath + "/modinfo.lua"
		// MacOS 不执行复制
		if config.Platform != "darwin" {
			err = utils.RemoveDir(utils.ModNoUgcPath + "/workshop-" + strconv.Itoa(enableForm.ID))
			if err != nil {
				utils.Logger.Error("删除旧MOD文件失败", "err", err, "cmd", enableForm.ID)
			}
			cmd := "cp -r " + modDirPath + " " + utils.ModNoUgcPath + "/workshop-" + strconv.Itoa(enableForm.ID)
			err = utils.BashCMD(cmd)
			if err != nil {
				utils.Logger.Error("复制MOD文件失败", "err", err, "cmd", cmd)
			}
		}
	}

	luaScript, _ := utils.GetFileAllContent(modInfoLuaFile)

	// 获取新modoverrides.lua
	modOverrides := utils.AddModDefaultConfig(luaScript, enableForm.ID, langStr)
	for _, mod := range modOverrides {
		modFormattedData = append(modFormattedData, utils.ModFormattedData{
			ID:                   mod.ID,
			Enable:               mod.Enabled,
			ConfigurationOptions: mod.ConfigurationOptions,
		})
	}

	// 需要转一次json，否则会出现新mod的default变量无法添加
	a, _ := json.Marshal(modFormattedData)
	var b []utils.ModFormattedData
	_ = json.Unmarshal(a, &b)
	modOverridesLua := utils.ParseToLua(b)

	// 写入数据库
	config.RoomSetting.Mod = modOverridesLua
	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	if config.RoomSetting.Ground != "" {
		//Master/modoverrides.lua
		err = utils.TruncAndWriteFile(utils.MasterModPath, config.RoomSetting.Mod)
		if err != nil {
			utils.Logger.Error("地面modoverrides.lua写入失败", "err", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}
	}

	if config.RoomSetting.Cave != "" {
		//Caves/modoverrides.lua
		err = utils.TruncAndWriteFile(utils.CavesModPath, config.RoomSetting.Mod)
		if err != nil {
			utils.Logger.Error("洞穴modoverrides.lua写入失败", "err", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}
	}

	// MacOS 不修改mod自动下载配置
	if config.Platform == "darwin" {
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("enableModSuccess", langStr), "data": nil})
		return
	}

	err = DstModsSetup()
	if err != nil {
		utils.RespondWithError(c, 500, langStr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("enableModSuccess", langStr), "data": nil})
}

func handleDisableModPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	type DisableForm struct {
		ISUGC bool `json:"isUgc"`
		ID    int  `json:"id"`
	}

	var disableForm DisableForm
	if err := c.ShouldBindJSON(&disableForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 读取modinfo.lua
	luaScript, _ := utils.GetFileAllContent(utils.MasterModPath)
	modOverrides := utils.ModOverridesToStruct(luaScript)

	var newModOverrides []utils.ModOverrides
	for _, mod := range modOverrides {
		if mod.ID != disableForm.ID {
			newModOverrides = append(newModOverrides, mod)
		}
	}

	// 需要转一次json，否则会出现新mod的default变量无法添加
	a, _ := json.Marshal(newModOverrides)
	var b []utils.ModOverrides
	_ = json.Unmarshal(a, &b)
	var modFormattedData []utils.ModFormattedData
	for _, mod := range b {
		modFormattedData = append(modFormattedData, utils.ModFormattedData{
			ID:                   mod.ID,
			Enable:               mod.Enabled,
			ConfigurationOptions: mod.ConfigurationOptions,
		})
	}
	newModOverridesLua := utils.ParseToLua(modFormattedData)

	// 写入数据库
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	config.RoomSetting.Mod = newModOverridesLua
	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	//Master/modoverrides.lua
	err = utils.TruncAndWriteFile(utils.MasterModPath, config.RoomSetting.Mod)
	if err != nil {
		utils.Logger.Error("地面modoverrides.lua写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	if config.RoomSetting.Cave != "" {
		//Caves/modoverrides.lua
		err = utils.TruncAndWriteFile(utils.CavesModPath, config.RoomSetting.Mod)
		if err != nil {
			utils.Logger.Error("洞穴modoverrides.lua写入失败", "err", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}
	}

	err = DstModsSetup()
	if err != nil {
		utils.RespondWithError(c, 500, langStr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("deleteModSuccess", langStr), "data": newModOverridesLua})
}

func handleGetMultiHostGet(c *gin.Context) {
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": config.MultiHost})
}

func handleChangeMultiHostPost(c *gin.Context) {
	type MultiHostForm struct {
		MultiHost bool `json:"multiHost"`
	}

	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	var multiHostForm MultiHostForm
	if err := c.ShouldBindJSON(&multiHostForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config.MultiHost = multiHostForm.MultiHost
	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("configUpdateSuccess", langStr), "data": nil})
}

func handleSystemSettingGet(c *gin.Context) {
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	var data SystemSettingForm
	data.SysMetricsGet = config.SysSetting.SchedulerSetting.SysMetricsGet
	data.KeepaliveDisable = !config.Keepalive.Enable
	data.PlayerGetFrequency = config.SysSetting.SchedulerSetting.PlayerGetFrequency
	data.UIDMaintain = config.SysSetting.SchedulerSetting.UIDMaintain
	data.KeepaliveFrequency = config.Keepalive.Frequency
	data.Bit64 = config.Bit64
	data.TickRate = config.TickRate
	data.EncodeUserPath = config.EncodeUserPath

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": data})
}

func handleSystemSettingPut(c *gin.Context) {
	defer scheduler.ReloadScheduler()
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	var systemSettingForm SystemSettingForm
	if err := c.ShouldBindJSON(&systemSettingForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	config.SysSetting.SchedulerSetting.SysMetricsGet.Disable = systemSettingForm.SysMetricsGet.Disable
	config.SysSetting.SchedulerSetting.UIDMaintain.Frequency = systemSettingForm.UIDMaintain.Frequency
	config.SysSetting.SchedulerSetting.UIDMaintain.Disable = systemSettingForm.UIDMaintain.Disable
	config.SysSetting.SchedulerSetting.PlayerGetFrequency = systemSettingForm.PlayerGetFrequency
	config.Keepalive.Frequency = systemSettingForm.KeepaliveFrequency
	config.Keepalive.Enable = !systemSettingForm.KeepaliveDisable

	if config.TickRate != systemSettingForm.TickRate {
		config.TickRate = systemSettingForm.TickRate
		err = saveSetting(config)
		if err != nil {
			utils.Logger.Error("设置Tick Rate失败", "err", err)
		}
	}

	if config.SysSetting.SchedulerSetting.SysMetricsGet.Disable {
		utils.SYS_METRICS = []utils.SysMetrics{}
	}

	if config.Bit64 != systemSettingForm.Bit64 {
		config.Bit64 = systemSettingForm.Bit64
		if config.Bit64 {
			// 安装64位依赖
			go utils.ExecBashScript("tmp.sh", utils.Install64Dependency)
		} else {
			// 安装32位依赖
			go utils.ExecBashScript("tmp.sh", utils.Install32Dependency)
		}
	}

	if config.EncodeUserPath.Ground != systemSettingForm.EncodeUserPath.Ground {
		config.EncodeUserPath.Ground = systemSettingForm.EncodeUserPath.Ground
		err = saveSetting(config)
		if err != nil {
			utils.Logger.Error("生成游戏配置文件失败", "err", err)
		}
	}
	if config.EncodeUserPath.Cave != systemSettingForm.EncodeUserPath.Cave {
		config.EncodeUserPath.Cave = systemSettingForm.EncodeUserPath.Cave
		err = saveSetting(config)
		if err != nil {
			utils.Logger.Error("生成游戏配置文件失败", "err", err)
		}
	}

	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("配置文件写入失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("configUpdateSuccess", langStr), "data": nil})
}

func handleMacOSModExportPost(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		utils.Logger.Error("无法获取 home 目录", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	err = utils.RemoveDir(utils.MacModExportPath)
	if err != nil {
		utils.Logger.Error("删除目录失败", "err", err, "dir", utils.MacModExportPath)
	}

	var cpCmds []string

	modPathUgc := homeDir + "/" + utils.ModDownloadPath + "/steamapps/workshop/content/322330"
	modsUgc, err := utils.GetDirs(modPathUgc)
	if err != nil {
		utils.Logger.Error("无法获取已下载的UGC MOD目录", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	for _, i := range modsUgc {
		cmd := "cp -r " + modPathUgc + "/" + i + " " + utils.MacModExportPath + "/workshop-" + i
		cpCmds = append(cpCmds, cmd)
	}

	modPathNotUgc := homeDir + "/" + utils.ModDownloadPath + "/not_ugc"
	modsNotUgc, err := utils.GetDirs(modPathNotUgc)
	if err != nil {
		utils.Logger.Error("无法获取已下载的非UGC MOD目录", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	for _, i := range modsNotUgc {
		cmd := "cp -r " + modPathNotUgc + "/" + i + " " + utils.MacModExportPath + "/workshop-" + i
		cpCmds = append(cpCmds, cmd)
	}

	err = utils.BashCMD("mkdir -p " + utils.MacModExportPath)
	if err != nil {
		utils.Logger.Error("创建mod导出目录失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}
	for _, cmd := range cpCmds {
		err = utils.BashCMD(cmd)
		if err != nil {
			utils.Logger.Error("复制mod失败", "err", err)
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("exportSuccess", langStr), "data": nil})
}

func handleModUpdatePost(c *gin.Context) {
	// 同步阻塞接口，耗时较长
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	type UpdateForm struct {
		ID      int    `json:"id"`
		ISUGC   bool   `json:"isUgc"`
		FileURL string `json:"fileURL"`
	}

	var updateForm UpdateForm
	if err := c.ShouldBindJSON(&updateForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		utils.Logger.Error("无法获取 home 目录", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	// 删除，非UGC会在下载前自动删除
	var modDirPath string
	if updateForm.ISUGC {
		modDirPath = homeDir + "/" + utils.ModDownloadPath + "/steamapps/workshop/content/322330/" + strconv.Itoa(updateForm.ID)
	}

	err = utils.RemoveDir(modDirPath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("deleteModFail", langStr), "data": nil})
		return
	}

	// 下载
	if updateForm.ISUGC {
		cmd := utils.GenerateModDownloadCMD(updateForm.ID)
		err := utils.BashCMD(cmd)
		if err != nil {
			utils.Logger.Error("MOD下载失败，MOD更新终止", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("updateModFail", langStr), "data": nil})
			return
		}
	} else {
		err := externalApi.DownloadMod(updateForm.FileURL, updateForm.ID)
		if err != nil {
			utils.Logger.Error("MOD下载失败，MOD更新终止", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("updateModFail", langStr), "data": nil})
			return
		}
	}

	// 删除 dst mod，复制新 mod 文件
	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败，MOD更新终止", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("updateModFail", langStr), "data": nil})
		return
	}
	if updateForm.ISUGC {
		if config.RoomSetting.Ground != "" {
			err = utils.RemoveDir(utils.MasterModUgcPath + "/" + strconv.Itoa(updateForm.ID))
			if err != nil {
				utils.Logger.Error("删除旧MOD文件失败", "err", err)
				c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("updateModFail", langStr), "data": nil})
				return
			}
			cmdMaster := "cp -r " + modDirPath + " " + utils.MasterModUgcPath + "/"
			err := utils.BashCMD(cmdMaster)
			if err != nil {
				utils.Logger.Error("复制MOD文件失败", "err", err, "cmd", cmdMaster)
				c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("updateModFail", langStr), "data": nil})
				return
			}
		}
		if config.RoomSetting.Cave != "" {
			err = utils.RemoveDir(utils.CavesModUgcPath + "/" + strconv.Itoa(updateForm.ID))
			if err != nil {
				utils.Logger.Error("删除旧MOD文件失败", "err", err)
				c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("updateModFail", langStr), "data": nil})
				return
			}
			cmdCaves := "cp -r " + modDirPath + " " + utils.CavesModUgcPath + "/"
			err = utils.BashCMD(cmdCaves)
			if err != nil {
				utils.Logger.Error("复制MOD文件失败", "err", err, "cmd", cmdCaves)
				c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("updateModFail", langStr), "data": nil})
				return
			}
		}
	} else {
		modDirPath = homeDir + "/" + utils.ModDownloadPath + "/not_ugc/" + strconv.Itoa(updateForm.ID)
		err = utils.RemoveDir(utils.ModNoUgcPath + "/workshop-" + strconv.Itoa(updateForm.ID))
		if err != nil {
			utils.Logger.Error("删除旧MOD文件失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("updateModFail", langStr), "data": nil})
			return
		}
		cmd := "cp -r " + modDirPath + " " + utils.ModNoUgcPath + "/workshop-" + strconv.Itoa(updateForm.ID)
		err = utils.BashCMD(cmd)
		if err != nil {
			utils.Logger.Error("复制MOD文件失败", "err", err, "cmd", cmd)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("updateModFail", langStr), "data": nil})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("updateModSuccess", langStr), "data": nil})

}

func handleAddClientModsDisabledConfig(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	if config.RoomSetting.Base.Name == "" {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("gameServerNotCreated", langStr), "data": nil})
		return
	}

	if config.RoomSetting.Ground != "" {
		//Master/modoverrides.lua
		modFileLines, err := readLines(utils.MasterModPath)
		if err != nil {
			utils.Logger.Error("地面modoverrides.lua读取失败", "err", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}

		var newModFileLines []string
		newModFileLines = append(newModFileLines, modFileLines[0])
		newModFileLines = append(newModFileLines, "  client_mods_disabled={configuration_options={}, enabled=true},")
		newModFileLines = append(newModFileLines, modFileLines[1:]...)

		config.RoomSetting.Mod = strings.Join(newModFileLines, "\n")

		err = utils.TruncAndWriteFile(utils.MasterModPath, config.RoomSetting.Mod)
		if err != nil {
			utils.Logger.Error("地面modoverrides.lua写入失败", "err", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}

		err = utils.WriteConfig(config)
		if err != nil {
			utils.Logger.Error("配置文件写入失败", "err", err)
			utils.RespondWithError(c, 500, "zh")
			return
		}
	}

	if config.RoomSetting.Cave != "" {
		//Caves/modoverrides.lua
		modFileLines, err := readLines(utils.CavesModPath)
		if err != nil {
			utils.Logger.Error("洞穴modoverrides.lua读取失败", "err", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}

		var newModFileLines []string
		newModFileLines = append(newModFileLines, modFileLines[0])
		newModFileLines = append(newModFileLines, "  client_mods_disabled={configuration_options={}, enabled=true},")
		newModFileLines = append(newModFileLines, modFileLines[1:]...)

		config.RoomSetting.Mod = strings.Join(newModFileLines, "\n")

		err = utils.TruncAndWriteFile(utils.CavesModPath, config.RoomSetting.Mod)
		if err != nil {
			utils.Logger.Error("洞穴modoverrides.lua写入失败", "err", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}

		err = utils.WriteConfig(config)
		if err != nil {
			utils.Logger.Error("配置文件写入失败", "err", err)
			utils.RespondWithError(c, 500, "zh")
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("enableModSuccess", langStr), "data": nil})
}

func handleDeleteClientModsDisabledConfig(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}

	if config.RoomSetting.Base.Name == "" {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("gameServerNotCreated", langStr), "data": nil})
		return
	}
	// 定义正则表达式来匹配目标内容
	re := regexp.MustCompile(`\s*client_mods_disabled=\s*\{(\s*configuration_options=\s*\{(\s*)*\},?\s*enabled=true\s*)\},?`)

	if config.RoomSetting.Ground != "" {
		//Master/modoverrides.lua
		luaScript, err := utils.GetFileAllContent(utils.MasterModPath)
		if err != nil {
			utils.Logger.Error("获取地面模组配置文件失败", "err", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}
		// 删除匹配到的内容
		luaScript = re.ReplaceAllString(luaScript, "")

		config.RoomSetting.Mod = luaScript
		err = utils.TruncAndWriteFile(utils.MasterModPath, config.RoomSetting.Mod)
		if err != nil {
			utils.Logger.Error("地面modoverrides.lua写入失败", "err", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}

		err = utils.WriteConfig(config)
		if err != nil {
			utils.Logger.Error("配置文件写入失败", "err", err)
			utils.RespondWithError(c, 500, "zh")
			return
		}
	}

	if config.RoomSetting.Cave != "" {
		//Caves/modoverrides.lua
		luaScript, err := utils.GetFileAllContent(utils.CavesModPath)
		if err != nil {
			utils.Logger.Error("获取洞穴模组配置文件失败", "err", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}
		// 删除匹配到的内容
		luaScript = re.ReplaceAllString(luaScript, "")

		config.RoomSetting.Mod = luaScript

		err = utils.TruncAndWriteFile(utils.CavesModPath, config.RoomSetting.Mod)
		if err != nil {
			utils.Logger.Error("洞穴modoverrides.lua写入失败", "err", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}

		err = utils.WriteConfig(config)
		if err != nil {
			utils.Logger.Error("配置文件写入失败", "err", err)
			utils.RespondWithError(c, 500, "zh")
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("disableModSuccess", langStr), "data": nil})

}
