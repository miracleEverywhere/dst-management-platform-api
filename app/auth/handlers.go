package auth

import (
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

type LoginForm struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type JsonBody struct {
	LoginForm LoginForm `json:"loginForm"`
}

type UpdatePasswordForm struct {
	Password string `json:"password"`
}

func handleLogin(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var loginForm JsonBody
	if err := c.ShouldBindJSON(&loginForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config, _ := utils.ReadConfig()
	// 校验用户名和密码
	if loginForm.LoginForm.Username != config.Username {
		utils.RespondWithError(c, 421, langStr)
		return
	}
	if loginForm.LoginForm.Password != config.Password {
		utils.RespondWithError(c, 422, langStr)
		return
	}

	jwtSecret := []byte(config.JwtSecret)
	token, _ := utils.GenerateJWT(config.Username, jwtSecret, 12)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": Success("loginSuccess", langStr), "data": gin.H{"token": token}})
}

func handleUserinfo(c *gin.Context) {
	config, _ := utils.ReadConfig()
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": gin.H{
		"username": config.Username,
		"nickname": config.Nickname,
	}})
}

func handleMenu(c *gin.Context) {
	type MenuItem struct {
		MenuId      int    `json:"menuId"`
		MenuName    string `json:"menuName"`
		EnName      string `json:"enName"`
		ParentId    int    `json:"parentId"`
		MenuType    string `json:"menuType"`
		Path        string `json:"path"`
		Name        string `json:"name"`
		Component   string `json:"component"`
		Icon        string `json:"icon"`
		IsHide      string `json:"isHide"`
		IsLink      string `json:"isLink"`
		IsKeepAlive string `json:"isKeepAlive"`
		IsFull      string `json:"isFull"`
		IsAffix     string `json:"isAffix"`
		Redirect    string `json:"redirect"`
		ActiveMenu  *int   `json:"activeMenu"`
	}

	type Response struct {
		Code    int        `json:"code"`
		Message string     `json:"message"`
		Data    []MenuItem `json:"data"`
	}
	menuItems := []MenuItem{
		{
			MenuId:      100,
			MenuName:    "个人中心",
			EnName:      "Profile",
			ParentId:    0,
			MenuType:    "2",
			Path:        "/profile",
			Name:        "profile",
			Component:   "profile/index",
			Icon:        "User",
			IsHide:      "0",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		},
		{
			MenuId:      101,
			MenuName:    "设置",
			EnName:      "Settings",
			ParentId:    0,
			MenuType:    "1",
			Path:        "/settings",
			Name:        "settings",
			Component:   "",
			Icon:        "Tools",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "/settings/room",
			ActiveMenu:  nil,
		},
		{
			MenuId:      10101,
			MenuName:    "房间",
			EnName:      "Room",
			ParentId:    101,
			MenuType:    "2",
			Path:        "/settings/room",
			Name:        "settingsRoom",
			Component:   "settings/room",
			Icon:        "sc-icon-Game",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		},
		{
			MenuId:      10102,
			MenuName:    "玩家",
			EnName:      "Player",
			ParentId:    101,
			MenuType:    "2",
			Path:        "/settings/player",
			Name:        "settingsPlayer",
			Component:   "settings/player",
			Icon:        "Avatar",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		},
		{
			MenuId:      102,
			MenuName:    "工具",
			EnName:      "Tools",
			ParentId:    0,
			MenuType:    "1",
			Path:        "/tools",
			Name:        "tools",
			Component:   "",
			Icon:        "sc-icon-ToolsFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "/tools/update",
			ActiveMenu:  nil,
		},
		{
			MenuId:      10201,
			MenuName:    "定时更新",
			EnName:      "Update",
			ParentId:    102,
			MenuType:    "2",
			Path:        "/tools/update",
			Name:        "toolsUpdate",
			Component:   "tools/update",
			Icon:        "sc-icon-DownloadCloudFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		},
		{
			MenuId:      10202,
			MenuName:    "定时备份",
			EnName:      "Backup",
			ParentId:    102,
			MenuType:    "2",
			Path:        "/tools/backup",
			Name:        "toolsBackup",
			Component:   "tools/backup",
			Icon:        "sc-icon-SaveFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		},
		{
			MenuId:      10203,
			MenuName:    "定时通知",
			EnName:      "Announce",
			ParentId:    102,
			MenuType:    "2",
			Path:        "/tools/announce",
			Name:        "toolsAnnounce",
			Component:   "tools/announce",
			Icon:        "sc-icon-NotificationFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		},
		{
			MenuId:      10204,
			MenuName:    "自动保活",
			EnName:      "Keepalive",
			ParentId:    102,
			MenuType:    "2",
			Path:        "/tools/keepalive",
			Name:        "toolsKeepalive",
			Component:   "tools/keepalive",
			Icon:        "sc-icon-PulseFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		},
		{
			MenuId:      10205,
			MenuName:    "安装游戏",
			EnName:      "Install",
			ParentId:    102,
			MenuType:    "2",
			Path:        "/tools/install",
			Name:        "toolsInstall",
			Component:   "tools/install",
			Icon:        "sc-icon-InstallFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		},
		//{
		//	MenuId:      10205,
		//	MenuName:    "模组安装",
		//	EnName:      "Mod",
		//	ParentId:    102,
		//	MenuType:    "2",
		//	Path:        "/tools/mod",
		//	Name:        "toolsMod",
		//	Component:   "tools/mod",
		//	Icon:        "sc-icon-DashboardFill",
		//	IsHide:      "1",
		//	IsLink:      "",
		//	IsKeepAlive: "0",
		//	IsFull:      "1",
		//	IsAffix:     "1",
		//	Redirect:    "",
		//	ActiveMenu:  nil,
		//},
		{
			MenuId:      10206,
			MenuName:    "数据统计",
			EnName:      "Statistics",
			ParentId:    102,
			MenuType:    "2",
			Path:        "/tools/statistics",
			Name:        "toolsStatistics",
			Component:   "tools/statistics",
			Icon:        "sc-icon-LineChartFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		},
		{
			MenuId:      103,
			MenuName:    "日志",
			EnName:      "Logs",
			ParentId:    0,
			MenuType:    "1",
			Path:        "/logs",
			Name:        "logs",
			Component:   "",
			Icon:        "sc-icon-FileListFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "/logs/ground",
			ActiveMenu:  nil,
		},
		{
			MenuId:      10301,
			MenuName:    "地面",
			EnName:      "Ground",
			ParentId:    103,
			MenuType:    "2",
			Path:        "/logs/ground",
			Name:        "logsGround",
			Component:   "logs/ground",
			Icon:        "sc-icon-SunFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "1",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		},
		{
			MenuId:      10302,
			MenuName:    "洞穴",
			EnName:      "Cave",
			ParentId:    103,
			MenuType:    "2",
			Path:        "/logs/cave",
			Name:        "logsCave",
			Component:   "logs/cave",
			Icon:        "sc-icon-TyphoonFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "1",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		},
		{
			MenuId:      10303,
			MenuName:    "聊天",
			EnName:      "Chat",
			ParentId:    103,
			MenuType:    "2",
			Path:        "/logs/chat",
			Name:        "logsChat",
			Component:   "logs/chat",
			Icon:        "sc-icon-MessageFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "1",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		},
		{
			MenuId:      10304,
			MenuName:    "平台",
			EnName:      "DMP",
			ParentId:    103,
			MenuType:    "2",
			Path:        "/logs/dmp",
			Name:        "logsDMP",
			Component:   "logs/dmp",
			Icon:        "sc-icon-CodeBoxFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "1",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		},
		{
			MenuId:      104,
			MenuName:    "帮助",
			EnName:      "Help",
			ParentId:    0,
			MenuType:    "2",
			Path:        "/help",
			Name:        "help",
			Component:   "help/index",
			Icon:        "sc-icon-HeartFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "/help",
			ActiveMenu:  nil,
		},
	}
	response := Response{
		Code:    200,
		Message: "success",
		Data:    menuItems,
	}

	// 返回 JSON 响应
	c.JSON(http.StatusOK, response)
}

func handleUpdatePassword(c *gin.Context) {
	lang, _ := c.Get("lang")
	langStr := "zh" // 默认语言
	if strLang, ok := lang.(string); ok {
		langStr = strLang
	}
	var updatePasswordForm UpdatePasswordForm
	if err := c.ShouldBindJSON(&updatePasswordForm); err != nil {
		// 如果绑定失败，返回 400 错误
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	config, _ := utils.ReadConfig()
	config.Password = updatePasswordForm.Password
	utils.WriteConfig(config)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": Success("updatePassword", langStr), "data": nil})
}
