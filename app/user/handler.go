package user

import (
	"dst-management-platform-api/database/dao"
	"dst-management-platform-api/database/db"
	"dst-management-platform-api/database/models"
	"dst-management-platform-api/logger"
	"dst-management-platform-api/utils"
	"github.com/gin-gonic/gin"
	"net/http"
)

type Handler struct {
	userDao   *dao.UserDAO
	systemDao *dao.SystemDAO
}

func NewUserHandler(userDao *dao.UserDAO, systemDao *dao.SystemDAO) *Handler {
	return &Handler{
		userDao:   userDao,
		systemDao: systemDao,
	}
}

func (h *Handler) registerPost(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message(c, "bad request"), "data": nil})
		return
	}

	num, err := h.userDao.Count(nil)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message(c, "register fail"), "data": nil})
		return
	}

	if num != 0 {
		logger.Logger.Info("创建用户失败，用户已存在", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message(c, "user exist"), "data": nil})
		return
	}

	if errCreate := h.userDao.Create(&user); errCreate != nil {
		logger.Logger.Error("创建用户失败", "err", errCreate)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message(c, "register fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message(c, "register success"), "data": nil})
	return
}

func (h *Handler) loginPost(c *gin.Context) {
	var user models.User
	if err := c.ShouldBindJSON(&user); err != nil {
		logger.Logger.Info("请求参数错误", "err", err, "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message(c, "bad request"), "data": nil})
		return
	}
	if user.Username == "" || user.Password == "" {
		logger.Logger.Info("请求参数缺失", "api", c.Request.URL.Path)
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": message(c, "bad request"), "data": nil})
		return
	}

	dbUser, err := h.userDao.GetUserByUsername(user.Username)
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message(c, "login fail"), "data": nil})
		return
	}

	if dbUser.Username == "" {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message(c, "user not exist"), "data": nil})
		return
	}

	if dbUser.Password != user.Password {
		c.JSON(http.StatusOK, gin.H{"code": 201, "message": message(c, "wrong password"), "data": nil})
		return
	}

	token, err := utils.GenerateJWT(*dbUser, []byte(db.JwtSecret), 24)
	if err != nil {
		logger.Logger.Error("生成jwt失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message(c, "login fail"), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": message(c, "login success"), "data": token})
}

func (h *Handler) menuGet(c *gin.Context) {
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
		}, // 个人中心
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
		}, // 设置
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
		}, // 房间
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
		}, // 玩家
		{
			MenuId:      10103,
			MenuName:    "导入",
			EnName:      "Import",
			ParentId:    101,
			MenuType:    "2",
			Path:        "/settings/import",
			Name:        "settingsImport",
			Component:   "settings/import",
			Icon:        "sc-icon-UninstallFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		}, // 导入
		{
			MenuId:      10104,
			MenuName:    "模组",
			EnName:      "Mod",
			ParentId:    101,
			MenuType:    "2",
			Path:        "/settings/mod",
			Name:        "settingsMod",
			Component:   "settings/mod",
			Icon:        "sc-icon-FileSettingsFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		}, // 模组
		{
			MenuId:      10105,
			MenuName:    "系统",
			EnName:      "System",
			ParentId:    101,
			MenuType:    "2",
			Path:        "/settings/system",
			Name:        "settingsSystem",
			Component:   "settings/system",
			Icon:        "sc-icon-SystemFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		}, // 系统
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
		}, // 工具
		{
			MenuId:      10202,
			MenuName:    "备份管理",
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
		}, // 备份管理 10202
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
		}, // 定时通知 10203
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
		}, // 安装游戏 10205
		{
			MenuId:      10206,
			MenuName:    "玩家统计",
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
		}, // 玩家统计 10206
		{
			MenuId:      10207,
			MenuName:    "世界预览",
			EnName:      "Location",
			ParentId:    102,
			MenuType:    "2",
			Path:        "/tools/location",
			Name:        "toolsLocation",
			Component:   "tools/location",
			Icon:        "sc-icon-PinDistanceFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		}, // 世界统计 10207
		{
			MenuId:      10208,
			MenuName:    "系统监控",
			EnName:      "Metrics",
			ParentId:    102,
			MenuType:    "2",
			Path:        "/tools/metrics",
			Name:        "toolsMetrics",
			Component:   "tools/metrics",
			Icon:        "Histogram",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		}, // 系统监控 10208
		{
			MenuId:      10209,
			MenuName:    "创建令牌",
			EnName:      "Token",
			ParentId:    102,
			MenuType:    "2",
			Path:        "/tools/token",
			Name:        "toolsToken",
			Component:   "tools/token",
			Icon:        "sc-icon-Lock2Fill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		}, // 创建令牌 10209
		{
			MenuId:      10210,
			MenuName:    "远程终端",
			EnName:      "WebSSH",
			ParentId:    102,
			MenuType:    "2",
			Path:        "/tools/webssh",
			Name:        "toolsWebSSH",
			Component:   "tools/webssh",
			Icon:        "sc-icon-TerminalBoxFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		}, // 远程终端 10210
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
		}, // 日志
		{
			MenuId:      10301,
			MenuName:    "世界日志",
			EnName:      "World",
			ParentId:    103,
			MenuType:    "2",
			Path:        "/logs/world",
			Name:        "logsWorld",
			Component:   "logs/world",
			Icon:        "sc-icon-EarthFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "1",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		}, // 世界日志
		{
			MenuId:      10303,
			MenuName:    "聊天日志",
			EnName:      "Chat",
			ParentId:    103,
			MenuType:    "2",
			Path:        "/logs/chat",
			Name:        "logsChat",
			Component:   "logs/chat",
			Icon:        "sc-icon-ChatSmileFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "1",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		}, // 聊天日志
		{
			MenuId:      10304,
			MenuName:    "请求日志",
			EnName:      "Access",
			ParentId:    103,
			MenuType:    "2",
			Path:        "/logs/access",
			Name:        "logsAccess",
			Component:   "logs/access",
			Icon:        "sc-icon-CodeBoxFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "1",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		}, // 请求日志
		{
			MenuId:      10304,
			MenuName:    "平台日志",
			EnName:      "Runtime",
			ParentId:    103,
			MenuType:    "2",
			Path:        "/logs/runtime",
			Name:        "logsRuntime",
			Component:   "logs/runtime",
			Icon:        "sc-icon-CpuFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "1",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		}, // 平台日志
		{
			MenuId:      10305,
			MenuName:    "Steam日志",
			EnName:      "Steam",
			ParentId:    103,
			MenuType:    "2",
			Path:        "/logs/steam",
			Name:        "logsSteam",
			Component:   "logs/steam",
			Icon:        "sc-icon-SteamFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "1",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		}, // Steam日志
		{
			MenuId:      10306,
			MenuName:    "清理日志",
			EnName:      "Clean",
			ParentId:    103,
			MenuType:    "2",
			Path:        "/logs/clean",
			Name:        "logsClean",
			Component:   "logs/clean",
			Icon:        "sc-icon-FileShredFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "1",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "",
			ActiveMenu:  nil,
		}, // 清理日志
		{
			MenuId:      104,
			MenuName:    "用户管理",
			EnName:      "Users",
			ParentId:    0,
			MenuType:    "2",
			Path:        "/users",
			Name:        "Users",
			Component:   "users/index",
			Icon:        "sc-icon-UserSettingsFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "/users",
			ActiveMenu:  nil,
		}, // 用户管理
		{
			MenuId:      105,
			MenuName:    "集群管理",
			EnName:      "Clusters",
			ParentId:    0,
			MenuType:    "2",
			Path:        "/clusters",
			Name:        "Cluster",
			Component:   "clusters/index",
			Icon:        "sc-icon-AppsFill",
			IsHide:      "1",
			IsLink:      "",
			IsKeepAlive: "0",
			IsFull:      "1",
			IsAffix:     "1",
			Redirect:    "/clusters",
			ActiveMenu:  nil,
		}, // 集群管理
		{
			MenuId:      106,
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
		}, // 帮助
	}

	// 非管理员拥有权限的菜单
	nonAdminID := []int{
		100,
		101, 10101, 10102, 10103, 10104, 10105,
		102, 10202, 10203, 10206, 10207, 10208, 10209,
		103, 10301, 10302, 10303, 10304, 10305,
		106,
	}

	var response Response

	role, exist := c.Get("role")
	if exist && role == "admin" {
		response = Response{
			Code:    200,
			Message: "success",
			Data:    menuItems,
		}
	} else {
		var nonAdminMenu []MenuItem
		for _, i := range nonAdminID {
			for _, item := range menuItems {
				if i == item.MenuId {
					nonAdminMenu = append(nonAdminMenu, item)
				}
			}
		}
		response = Response{
			Code:    200,
			Message: "success",
			Data:    nonAdminMenu,
		}
	}

	// 返回 JSON 响应
	c.JSON(http.StatusOK, response)
}

func (h *Handler) userInfo(c *gin.Context) {
	username, _ := c.Get("username")
	dbUser, err := h.userDao.GetUserByUsername(username.(string))
	if err != nil {
		logger.Logger.Error("查询数据库失败", "err", err)
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": message(c, "login fail"), "data": nil})
		return
	}
	dbUser.Password = ""

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": dbUser})
}
