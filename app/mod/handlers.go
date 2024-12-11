package mod

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func handleModSettingGet(c *gin.Context) {
	luaScript, _ := utils.GetFileAllContent(utils.MasterModPath)
	a := utils.ModOverridesToStruct(luaScript)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": a})
}

func handleModInfoGet(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	type ModConfig struct {
		ID            int                         `json:"id"`
		ConfigOptions []utils.ConfigurationOption `json:"configOptions"`
	}
	var modConfig ModConfig
	modID := 1216718131
	modInfoLuaFile := utils.ModUgcPath + "/" + strconv.Itoa(modID) + "/modinfo.lua"
	isUgcMod, err := utils.FileDirectoryExists(modInfoLuaFile)
	if err != nil {
		utils.RespondWithError(c, 500, langStr)
		return
	}

	if !isUgcMod {
		modInfoLuaFile = utils.ModNoUgcPath + "/workshop-" + strconv.Itoa(modID) + "/modinfo.lua"
	}

	luaScript, _ := utils.GetFileAllContent(modInfoLuaFile)
	modConfig.ID = modID
	modConfig.ConfigOptions = utils.GetModConfigOptions(luaScript)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": modConfig})
}
