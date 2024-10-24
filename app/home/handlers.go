package home

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

func handleRoomInfoGet(c *gin.Context) {
	type Data struct {
		RoomSettingBase utils.RoomSettingBase `json:"roomSettingBase"`
		SeasonInfo      metaInfo              `json:"seasonInfo"`
	}
	type Response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    Data   `json:"data"`
	}
	filePath, err := findLatestMetaFile(utils.MetaPath)
	if err != nil {
		return
	}

	seasonInfo := getMetaInfo(filePath)
	config, _ := utils.ReadConfig()

	data := Data{
		RoomSettingBase: config.RoomSetting.Base,
		SeasonInfo:      seasonInfo,
	}

	response := Response{
		Code:    200,
		Message: "success",
		Data:    data,
	}

	c.JSON(http.StatusOK, response)
}

func handleSystemInfoGet(c *gin.Context) {
	type Data struct {
		Cpu    float64 `json:"cpu"`
		Memory float64 `json:"memory"`
	}
	type Response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    Data   `json:"data"`
	}
	var response Response
	response.Code = 200
	response.Message = "success"
	response.Data.Cpu = utils.CpuUsage()
	response.Data.Memory = utils.MemoryUsage()
	c.JSON(http.StatusOK, response)
}
