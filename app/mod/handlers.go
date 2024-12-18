package mod

import (
	"dst-management-platform-api/app/externalApi"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
)

func handleModSettingFormatGet(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	luaScript, _ := utils.GetFileAllContent(utils.MasterModPath)

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
	var modConfig ModConfig
	modID := modConfigurationsForm.ID
	modInfoLuaFile := utils.ModUgcPath + "/" + strconv.Itoa(modID) + "/modinfo.lua"
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
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("needDownload", langStr), "data": nil})
			return
		}
	}

	luaScript, _ := utils.GetFileAllContent(modInfoLuaFile)
	modConfig.ID = modID
	modConfig.ConfigOptions = utils.GetModConfigOptions(luaScript, langStr)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": modConfig})
}

func test(c *gin.Context) {
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

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": nil})
}
