package home

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"time"
)

func handleRoomInfoGet(c *gin.Context) {

	type Data struct {
		RoomSettingBase utils.RoomSettingBase `json:"roomSettingBase"`
		SeasonInfo      metaInfo              `json:"seasonInfo"`
		ModsCount       int                   `json:"modsCount"`
		PlayerNum       int                   `json:"playerNum"`
	}
	type Response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    Data   `json:"data"`
	}

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("读取配置文件失败", "err", err)
		utils.RespondWithError(c, 500, "zh")
		return
	}
	modsCount, err := countMods(config.RoomSetting.Mod)
	if err != nil {
		utils.Logger.Error("读取mod数量失败", "err", err)
	}

	var filePath string

	if config.RoomSetting.Ground != "" {
		filePath, err = FindLatestMetaFile(utils.MasterMetaPath)
	} else {
		filePath, err = FindLatestMetaFile(utils.CavesMetaPath)
	}

	if err != nil {
		utils.Logger.Error("查询session-meta文件失败", "err", err)
	}

	var seasonInfo metaInfo
	if err != nil {
		seasonInfo, err = getMetaInfo("")
		utils.Logger.Error("获取meta文件内容失败", "err", err)
	} else {
		seasonInfo, err = getMetaInfo(filePath)
		if err != nil {
			utils.Logger.Error("获取meta文件内容失败", "err", err)
		}
	}

	var playerNum int
	if len(utils.STATISTICS) > 0 {
		players := utils.STATISTICS[len(utils.STATISTICS)-1].Players
		playerNum = len(players)
	} else {
		playerNum = 0
	}

	data := Data{
		RoomSettingBase: config.RoomSetting.Base,
		SeasonInfo:      seasonInfo,
		ModsCount:       modsCount,
		PlayerNum:       playerNum,
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
		Master int     `json:"master"`
		Caves  int     `json:"caves"`
	}
	type Response struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Data    Data   `json:"data"`
	}
	var err error
	var response Response
	response.Code = 200
	response.Message = "success"
	response.Data.Cpu, err = utils.CpuUsage()
	if err != nil {
		utils.Logger.Error("获取Cpu使用率失败", "err", err)
	}
	response.Data.Memory, err = utils.MemoryUsage()
	if err != nil {
		utils.Logger.Error("获取内存使用率失败", "err", err)
	}
	response.Data.Master = GetProcessStatus(utils.MasterScreenName)
	response.Data.Caves = GetProcessStatus(utils.CavesScreenName)
	c.JSON(http.StatusOK, response)
}

func handleExecPost(c *gin.Context) {
	type ExecForm struct {
		Type string `json:"type"`
		Info int    `json:"info"`
	}
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	var execFrom ExecForm
	if err := c.ShouldBindJSON(&execFrom); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	switch execFrom.Type {
	case "startup":
		err := utils.BashCMD(utils.KillDST)
		if err != nil {
			utils.Logger.Error("BashCMD执行失败", "err", err, "cmd", utils.KillDST)
		}
		err = utils.BashCMD(utils.ClearScreenCMD)
		if err != nil {
			utils.Logger.Error("BashCMD执行失败", "err", err, "cmd", utils.ClearScreenCMD)
		}
		masterStatus := GetProcessStatus(utils.MasterScreenName)
		cavesStatus := GetProcessStatus(utils.CavesScreenName)

		config, err := utils.ReadConfig()
		if err != nil {
			utils.Logger.Error("读取配置文件失败", "err", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}

		if config.RoomSetting.Ground != "" {
			if masterStatus == 0 {
				if config.Platform == "darwin" {
					err = utils.BashCMD(utils.MacStartMasterCMD)
					if err != nil {
						utils.Logger.Error("BashCMD执行失败", "err", err, "cmd", utils.MacStartMasterCMD)
					}
				} else {
					var cmd string
					if config.Bit64 {
						cmd = utils.StartMaster64CMD
					} else {
						cmd = utils.StartMasterCMD
					}
					err = utils.BashCMD(cmd)
					if err != nil {
						utils.Logger.Error("BashCMD执行失败", "err", err, "cmd", cmd)
					}
				}
			}
		}

		if config.RoomSetting.Cave != "" {
			if cavesStatus == 0 {
				if config.RoomSetting.Cave != "" {
					if config.Platform == "darwin" {
						err = utils.BashCMD(utils.MacStartCavesCMD)
						if err != nil {
							utils.Logger.Error("BashCMD执行失败", "err", err, "cmd", utils.MacStartCavesCMD)
						}
					} else {
						var cmd string
						if config.Bit64 {
							cmd = utils.StartCaves64CMD
						} else {
							cmd = utils.StartCavesCMD
						}
						err = utils.BashCMD(cmd)
						if err != nil {
							utils.Logger.Error("BashCMD执行失败", "err", err, "cmd", cmd)
						}
					}
				}
			}
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("startupSuccess", langStr), "data": nil})

	case "rollback":
		cmd := "c_rollback(" + strconv.Itoa(execFrom.Info) + ")"
		err := utils.ScreenCMD(cmd, utils.MasterName)
		if err != nil {
			utils.Logger.Error("ScreenCMD执行失败", "err", err, "cmd", utils.MasterName)
			utils.RespondWithError(c, 511, langStr)
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("rollbackSuccess", langStr), "data": nil})

	case "shutdown":
		err := utils.StopGame()
		if err != nil {
			utils.Logger.Error("关闭游戏失败", "err", err)
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("shutdownSuccess", langStr), "data": nil})

	case "restart":
		err := utils.StopGame()
		if err != nil {
			utils.Logger.Error("关闭游戏失败", "err", err)
		}
		err = utils.StartGame()
		if err != nil {
			utils.Logger.Error("启动游戏失败", "err", err)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("restartFail", langStr), "data": nil})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("restartSuccess", langStr), "data": nil})

	case "update":
		err := utils.StopGame()
		if err != nil {
			utils.Logger.Error("关闭游戏失败", "err", err)
		}

		go func() {
			err = utils.BashCMD(utils.UpdateGameCMD)
			if err != nil {
				utils.Logger.Error("BashCMD执行失败", "err", err, "cmd", utils.UpdateGameCMD)
			}
			err = utils.StartGame()
			if err != nil {
				utils.Logger.Error("启动游戏失败", "err", err)
			}
		}()

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("updating", langStr), "data": nil})

	case "reset":
		cmd := "c_regenerateworld()"
		err := utils.ScreenCMD(cmd, utils.MasterName)
		if err != nil {
			utils.Logger.Error("ScreenCMD执行失败", "err", err, "cmd", cmd)
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("execFail", langStr), "data": nil})
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("resetSuccess", langStr), "data": nil})

	case "delete":
		err := utils.StopGame()
		if err != nil {
			utils.Logger.Error("关闭游戏失败", "err", err)
		}

		time.Sleep(2 * time.Second)

		config, err := utils.ReadConfig()
		if err != nil {
			utils.Logger.Error("读取配置文件失败", "err", err)
			utils.RespondWithError(c, 500, langStr)
			return
		}

		var (
			cmd    string
			master bool
			caves  bool
		)
		if config.RoomSetting.Ground != "" {
			cmd = "rm -rf " + utils.MasterPath + "/*"
			err = utils.BashCMD(cmd)
			if err != nil {
				utils.Logger.Error("删除地面失败", "err", err)
			}
			master = true
		}
		if config.RoomSetting.Cave != "" {
			cmd = "rm -rf " + utils.CavesPath + "/*"
			err = utils.BashCMD(cmd)
			if err != nil {
				utils.Logger.Error("删除洞穴失败", "err", err)
			}
			caves = true
		}

		if !master && !caves {
			c.JSON(http.StatusOK, gin.H{
				"code":    201,
				"message": response("deleteFail", langStr),
				"data":    nil,
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"code":    200,
				"message": response("deleteSuccess", langStr),
				"data":    nil,
			})
		}

	case "masterSwitch":
		if execFrom.Info == 0 {
			cmd := "c_shutdown()"
			err := utils.ScreenCMD(cmd, utils.MasterName)
			if err != nil {
				utils.Logger.Error("ScreenCMD执行失败", "err", err, "cmd", cmd)
			}
			time.Sleep(2 * time.Second)
			err = utils.BashCMD(utils.StopMasterCMD)
			if err != nil {
				utils.Logger.Error("BashCMD执行失败", "err", err, "cmd", utils.StopMasterCMD)
			}
			err = utils.BashCMD(utils.ClearScreenCMD)
			if err != nil {
				utils.Logger.Error("BashCMD执行失败", "err", err, "cmd", utils.ClearScreenCMD)
			}

			c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("shutdownSuccess", langStr), "data": nil})
		} else {
			config, err := utils.ReadConfig()
			if err != nil {
				utils.Logger.Error("读取配置文件失败", "err", err)
				utils.RespondWithError(c, 500, langStr)
				return
			}
			if config.RoomSetting.Ground == "" {
				c.JSON(http.StatusOK, gin.H{"code": 200, "message": "no master", "data": nil})
				return
			}
			//开启服务器
			err = utils.BashCMD(utils.ClearScreenCMD)
			if err != nil {
				utils.Logger.Error("BashCMD执行失败", "err", err, "cmd", utils.ClearScreenCMD)
			}
			time.Sleep(1 * time.Second)
			var cmd string
			if config.Platform == "darwin" {
				cmd = utils.MacStartMasterCMD
			} else {
				if config.Bit64 {
					cmd = utils.StartMaster64CMD
				} else {
					cmd = utils.StartMasterCMD
				}
			}
			err = utils.BashCMD(cmd)
			if err != nil {
				utils.Logger.Error("启动游戏失败", "err", err, "cmd", cmd)
				c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("startupFail", langStr), "data": nil})
			}
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("startupSuccess", langStr), "data": nil})
		}

	case "cavesSwitch":
		if execFrom.Info == 0 {
			cmd := "c_shutdown()"
			err := utils.ScreenCMD(cmd, utils.CavesName)
			if err != nil {
				utils.Logger.Error("ScreenCMD执行失败", "err", err, "cmd", cmd)
			}
			time.Sleep(2 * time.Second)
			err = utils.BashCMD(utils.StopCavesCMD)
			if err != nil {
				utils.Logger.Error("BashCMD执行失败", "err", err, "cmd", utils.StopCavesCMD)
			}
			err = utils.BashCMD(utils.ClearScreenCMD)
			if err != nil {
				utils.Logger.Error("BashCMD执行失败", "err", err, "cmd", utils.ClearScreenCMD)
			}

			c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("shutdownSuccess", langStr), "data": nil})
		} else {
			config, err := utils.ReadConfig()
			if err != nil {
				utils.Logger.Error("读取配置文件失败", "err", err)
				utils.RespondWithError(c, 500, langStr)
				return
			}
			if config.RoomSetting.Cave == "" {
				c.JSON(http.StatusOK, gin.H{"code": 200, "message": "no caves", "data": nil})
				return
			}
			//开启服务器
			err = utils.BashCMD(utils.ClearScreenCMD)
			if err != nil {
				utils.Logger.Error("BashCMD执行失败", "err", err, "cmd", utils.ClearScreenCMD)
			}
			time.Sleep(1 * time.Second)

			var cmd string
			if config.Platform == "darwin" {
				cmd = utils.MacStartCavesCMD
			} else {
				if config.Bit64 {
					cmd = utils.StartCaves64CMD
				} else {
					cmd = utils.StartCavesCMD
				}
			}
			err = utils.BashCMD(cmd)
			if err != nil {
				utils.Logger.Error("启动游戏失败", "err", err)
				c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("startupFail", langStr), "data": nil})
				return
			}
			c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("startupSuccess", langStr), "data": nil})
		}

	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
	}
}

func handleAnnouncementPost(c *gin.Context) {
	type AnnouncementForm struct {
		Message string `json:"message"`
	}
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}

	var announcementForm AnnouncementForm
	if err := c.ShouldBindJSON(&announcementForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cmd := "c_announce('" + announcementForm.Message + "')"

	config, err := utils.ReadConfig()
	if err != nil {
		utils.Logger.Error("配置文件读取失败", "err", err)
		utils.RespondWithError(c, 500, langStr)
		return
	}

	var cmdErr error
	if config.RoomSetting.Ground != "" {
		cmdErr = utils.ScreenCMD(cmd, utils.MasterName)
	} else {
		cmdErr = utils.ScreenCMD(cmd, utils.CavesName)
	}

	if cmdErr != nil {
		utils.Logger.Error("ScreenCMD执行失败", "err", cmdErr, "cmd", cmd)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("announceFail", langStr), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("announceSuccess", langStr), "data": nil})
}

func handleConsolePost(c *gin.Context) {
	type ConsoleForm struct {
		CMD   string `json:"cmd"`
		World string `json:"world"`
	}
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var consoleForm ConsoleForm
	if err := c.ShouldBindJSON(&consoleForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	cmd := consoleForm.CMD
	if consoleForm.World == "master" {
		err := utils.ScreenCMD(cmd, utils.MasterName)
		if err != nil {
			utils.Logger.Error("ScreenCMD执行失败", "err", err, "cmd", cmd)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("execFail", langStr), "data": nil})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("execSuccess", langStr), "data": nil})
		return
	}
	if consoleForm.World == "caves" {
		err := utils.ScreenCMD(cmd, utils.CavesName)
		if err != nil {
			utils.Logger.Error("ScreenCMD执行失败", "err", err, "cmd", cmd)
			c.JSON(http.StatusOK, gin.H{"code": 201, "message": response("execFail", langStr), "data": nil})
			return
		}

		c.JSON(http.StatusOK, gin.H{"code": 200, "message": response("execSuccess", langStr), "data": nil})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
}
