package setting

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func handleRoomSettingGet(c *gin.Context) {
	config, _ := utils.ReadConfig()
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
	config, _ := utils.ReadConfig()
	config.RoomSetting = roomSetting
	utils.WriteConfig(config)

	saveSetting(config)
	dstModsSetup()

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
	config, _ := utils.ReadConfig()
	config.RoomSetting = roomSetting
	utils.WriteConfig(config)

	saveSetting(config)
	dstModsSetup()
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
	config, _ := utils.ReadConfig()
	config.RoomSetting = roomSetting
	utils.WriteConfig(config)

	saveSetting(config)
	dstModsSetup()
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

	config, _ := utils.ReadConfig()
	adminList := getList(utils.AdminListPath)
	blockList := getList(utils.BlockListPath)
	whiteList := getList(utils.WhiteListPath)

	var playList PlayerList
	playList.Players = config.Players
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
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("kickFail", langStr), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("kickSuccess", langStr), "data": nil})
}
