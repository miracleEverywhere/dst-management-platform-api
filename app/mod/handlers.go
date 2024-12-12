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
	type ResponseData struct {
		ID                   int                    `json:"id"`
		Name                 string                 `json:"name"`
		Enable               bool                   `json:"enable"`
		ConfigurationOptions map[string]interface{} `json:"configurationOptions"`
		PreviewUrl           string                 `json:"preview_url"`
	}

	modInfo, err := externalApi.GetModsInfo(luaScript)
	if err != nil {
		utils.RespondWithError(c, 500, langStr)
		return
	}

	var responseData []ResponseData
	for _, i := range utils.ModOverridesToStruct(luaScript) {
		item := ResponseData{
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
	modConfig.ConfigOptions = utils.GetModConfigOptions(luaScript)

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": modConfig})
}
