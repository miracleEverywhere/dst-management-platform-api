package setting

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
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
	err = utils.WriteConfig(config)
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	err = saveSetting(config)
	if err != nil {
		utils.Logger.Error("房间配置保存失败", "err", err)
	}
	err = dstModsSetup()
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
	err = dstModsSetup()
	if err != nil {
		utils.Logger.Error("mod配置保存失败", "err", err)
	}
	restartWorld(c, config, langStr)

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
	err = dstModsSetup()
	if err != nil {
		utils.Logger.Error("mod配置保存失败", "err", err)
	}
	generateWorld(c, config, langStr)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("generateSuccess", langStr), "data": nil})

}

func handlePlayerListGet(c *gin.Context) {
	type PlayerList struct {
		Players   []utils.Players `json:"players"`
		AdminList []string        `json:"adminList"`
		BlockList []string        `json:"blockList"`
		WhiteList []string        `json:"whiteList"`
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

	var playList PlayerList
	//playList.Players = config.Players
	playList.Players = utils.STATISTICS[len(utils.STATISTICS)-1].Players
	playList.AdminList = adminList
	playList.BlockList = blockList
	playList.WhiteList = whiteList

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": playList})
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
	err = writeDatabase()
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("uploadSuccess", langStr), "data": nil})
}
