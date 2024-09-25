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
		utils.RespondWithError(c, 420, langStr)
		return
	}
	if loginForm.LoginForm.Password != config.Password {
		utils.RespondWithError(c, 421, langStr)
		return
	}

	jwtSecret := []byte(config.JwtSecret)
	token, _ := utils.GenerateJWT(config.Username, jwtSecret, 12)
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "ok", "data": gin.H{"token": token}})
}

func handleUserinfo(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success", "data": gin.H{"username": "admin"}})
}

func handleMenu(c *gin.Context) {
	type BoolOrNil interface{}
	type ChildItem struct {
		ID          int         `json:"id"`
		Name        string      `json:"name"`
		Code        string      `json:"code"`
		Type        string      `json:"type"`
		ParentID    *int        `json:"parentId"` // 允许为 nil
		Path        string      `json:"path"`
		Redirect    *string     `json:"redirect"` // 允许为 nil
		Icon        string      `json:"icon"`
		Component   string      `json:"component"`
		Layout      string      `json:"layout"`
		KeepAlive   BoolOrNil   `json:"keepAlive"`   // 允许为 nil
		Method      *string     `json:"method"`      // 允许为 nil
		Description *string     `json:"description"` // 允许为 nil
		Show        bool        `json:"show"`
		Enable      bool        `json:"enable"`
		Order       int         `json:"order"`
		Children    []ChildItem `json:"children,omitempty"` // 可选字段
	}

	type Response struct {
		Code    int         `json:"code"`
		Message string      `json:"message"`
		Data    []ChildItem `json:"data"`
	}
	data := []ChildItem{
		{
			ID:          100,
			Name:        "首页",
			Code:        "Home",
			Type:        "MENU",
			ParentID:    nil,
			Path:        "/",
			Redirect:    nil,
			Icon:        "i-fe:home",
			Component:   "/src/views/home/index.vue",
			Layout:      "",
			KeepAlive:   true,
			Method:      nil,
			Description: nil,
			Show:        true,
			Enable:      true,
			Order:       0,
		},
		{
			ID:          101,
			Name:        "设置",
			Code:        "Setting",
			Type:        "MENU",
			ParentID:    nil,
			Path:        "",
			Redirect:    nil,
			Icon:        "i-fe:settings",
			Component:   "",
			Layout:      "",
			KeepAlive:   true,
			Method:      nil,
			Description: nil,
			Show:        true,
			Enable:      true,
			Order:       1,
			Children: []ChildItem{
				{
					ID:          10101,
					Name:        "玩家",
					Code:        "Player",
					Type:        "MENU",
					ParentID:    func(i int) *int { return &i }(101),
					Path:        "/setting/player",
					Redirect:    nil,
					Icon:        "i-fe:user",
					Component:   "/src/views/setting/player.vue",
					Layout:      "",
					KeepAlive:   true,
					Method:      nil,
					Description: nil,
					Show:        true,
					Enable:      true,
					Order:       0,
				},
				{
					ID:          10102,
					Name:        "房间",
					Code:        "Room",
					Type:        "MENU",
					ParentID:    func(i int) *int { return &i }(101),
					Path:        "/setting/room",
					Redirect:    nil,
					Icon:        "i-fe:codesandbox",
					Component:   "/src/views/setting/room.vue",
					Layout:      "",
					KeepAlive:   true,
					Method:      nil,
					Description: nil,
					Show:        true,
					Enable:      true,
					Order:       1,
				},
			},
		},
		{
			ID:          102,
			Name:        "工具",
			Code:        "Tools",
			Type:        "MENU",
			ParentID:    nil,
			Path:        "",
			Redirect:    nil,
			Icon:        "i-fe:tool",
			Component:   "",
			Layout:      "",
			KeepAlive:   true,
			Method:      nil,
			Description: nil,
			Show:        true,
			Enable:      true,
			Order:       2,
			Children: []ChildItem{
				{
					ID:          10201,
					Name:        "定时更新",
					Code:        "Update",
					Type:        "MENU",
					ParentID:    func(i int) *int { return &i }(102),
					Path:        "/tools/update",
					Redirect:    nil,
					Icon:        "i-fe:download-cloud",
					Component:   "/src/views/tools/update.vue",
					Layout:      "",
					KeepAlive:   true,
					Method:      nil,
					Description: nil,
					Show:        true,
					Enable:      true,
					Order:       0,
				},
				{
					ID:          10202,
					Name:        "定时备份",
					Code:        "Backup",
					Type:        "MENU",
					ParentID:    func(i int) *int { return &i }(102),
					Path:        "/tools/backup",
					Redirect:    nil,
					Icon:        "i-fe:save",
					Component:   "/src/views/tools/backup.vue",
					Layout:      "",
					KeepAlive:   true,
					Method:      nil,
					Description: nil,
					Show:        true,
					Enable:      true,
					Order:       1,
				},
				{
					ID:          10203,
					Name:        "定时通知",
					Code:        "Announce",
					Type:        "MENU",
					ParentID:    func(i int) *int { return &i }(102),
					Path:        "/tools/announce",
					Redirect:    nil,
					Icon:        "i-fe:send",
					Component:   "/src/views/tools/announce.vue",
					Layout:      "",
					KeepAlive:   true,
					Method:      nil,
					Description: nil,
					Show:        true,
					Enable:      true,
					Order:       2,
				},
			},
		},
		{
			ID:          103,
			Name:        "日志",
			Code:        "Logs",
			Type:        "MENU",
			ParentID:    nil,
			Path:        "",
			Redirect:    nil,
			Icon:        "i-fe:settings",
			Component:   "",
			Layout:      "",
			KeepAlive:   true,
			Method:      nil,
			Description: nil,
			Show:        true,
			Enable:      true,
			Order:       3,
			Children: []ChildItem{
				{
					ID:          10301,
					Name:        "地面",
					Code:        "Ground",
					Type:        "MENU",
					ParentID:    func(i int) *int { return &i }(103),
					Path:        "/logs/ground",
					Redirect:    nil,
					Icon:        "i-fe:sunrise",
					Component:   "/src/views/logs/ground.vue",
					Layout:      "",
					KeepAlive:   true,
					Method:      nil,
					Description: nil,
					Show:        true,
					Enable:      true,
					Order:       0,
				},
				{
					ID:          10302,
					Name:        "洞穴",
					Code:        "Cave",
					Type:        "MENU",
					ParentID:    func(i int) *int { return &i }(103),
					Path:        "/logs/cave",
					Redirect:    nil,
					Icon:        "i-fe:sunset",
					Component:   "/src/views/logs/cave.vue",
					Layout:      "",
					KeepAlive:   true,
					Method:      nil,
					Description: nil,
					Show:        true,
					Enable:      true,
					Order:       1,
				},
				{
					ID:          10303,
					Name:        "聊天",
					Code:        "Chat",
					Type:        "MENU",
					ParentID:    func(i int) *int { return &i }(103),
					Path:        "/logs/chat",
					Redirect:    nil,
					Icon:        "i-fe:message-square",
					Component:   "/src/views/logs/chat.vue",
					Layout:      "",
					KeepAlive:   true,
					Method:      nil,
					Description: nil,
					Show:        true,
					Enable:      true,
					Order:       2,
				},
			},
		},
		{
			ID:          104,
			Name:        "帮助",
			Code:        "Help",
			Type:        "MENU",
			ParentID:    nil,
			Path:        "/help",
			Redirect:    nil,
			Icon:        "i-fe:help-circle",
			Component:   "/src/views/help/index.vue",
			Layout:      "",
			KeepAlive:   true,
			Method:      nil,
			Description: nil,
			Show:        true,
			Enable:      true,
			Order:       4,
		},
	}
	response := Response{
		Code:    200,
		Message: "success",
		Data:    data,
	}

	// 返回 JSON 响应
	c.JSON(http.StatusOK, response)
}
