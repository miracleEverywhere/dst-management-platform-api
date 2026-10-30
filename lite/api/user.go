package api

import (
	"crypto/subtle"
	"dst-management-platform-api/lite/utils"
	"net/http"
)

func registerPost(w http.ResponseWriter, r *http.Request) {
	// 用户名是唯一标识，只按用户名判断是否已注册
	if utils.DB.User.Username != "" {
		writeJSON(w, 201, msgUserExist, nil)
		return
	}

	var user utils.User
	if err := bindJSON(r, &user); err != nil || !validateUser(user) {
		writeJSON(w, 400, msgBadRequest, nil)
		return
	}

	hash, err := utils.HashPassword(user.Password)
	if err != nil {
		utils.Logger.Error("生成密码哈希失败: " + err.Error())
		writeJSON(w, 500, msgRegisterFail, nil)
		return
	}
	user.Password = hash

	utils.DB.User = user

	writeJSON(w, 200, msgRegisterSuccess, nil)
}

func registerGet(w http.ResponseWriter, r *http.Request) {
	// 与主项目语义保持一致：true 表示还没有用户、需要注册
	needRegister := utils.DB.User.Username == ""
	writeJSON(w, 200, msgSuccess, needRegister)
}

func loginPost(w http.ResponseWriter, r *http.Request) {
	var form utils.User
	if err := bindJSON(r, &form); err != nil || !validateUser(form) {
		writeJSON(w, 400, msgBadRequest, nil)
		return
	}

	dbUser := utils.DB.User
	if dbUser.Username == "" || subtle.ConstantTimeCompare([]byte(dbUser.Username), []byte(form.Username)) != 1 {
		writeJSON(w, 201, msgUserNotExist, nil)
		return
	}

	// 与主项目保持一致：用户名正确但密码不匹配时返回单独的提示
	if !utils.ValidatePassword(form.Password, dbUser.Password) {
		writeJSON(w, 201, msgWrongPassword, nil)
		return
	}

	// 密钥为空时拒绝签发，避免用空密钥签出的 token 可被任何人伪造
	secret := utils.DB.System.JwtSecret
	if secret == "" {
		utils.Logger.Error("JWT密钥为空，拒绝签发token")
		writeJSON(w, 500, msgLoginFail, nil)
		return
	}

	token, err := utils.GenerateJWT(utils.User{Username: dbUser.Username}, []byte(secret), utils.JwtExpirationHours)
	if err != nil {
		utils.Logger.Error("生成JWT失败: " + err.Error())
		writeJSON(w, 500, msgLoginFail, nil)
		return
	}

	writeJSON(w, 200, msgLoginSuccess, token)
}

func menuGet(w http.ResponseWriter, r *http.Request) {
	type menuItem struct {
		ID        int        `json:"id"`
		Type      string     `json:"type"`
		Section   string     `json:"section"`
		Title     string     `json:"title"`
		To        string     `json:"to"`
		Component string     `json:"component"`
		Icon      string     `json:"icon"`
		Links     []menuItem `json:"links"`
	}
	menu := []menuItem{
		{
			ID:        1,
			Type:      "link",
			Section:   "",
			Title:     "dashboard",
			To:        "/dashboard",
			Component: "lite/dashboard/index",
			Icon:      "ri-function-ai-line",
			Links:     nil,
		},
		{
			ID:        2,
			Type:      "link",
			Section:   "",
			Title:     "room",
			To:        "/room",
			Component: "lite/room/index",
			Icon:      "ri-gamepad-line",
			Links:     nil,
		},
		{
			ID:        3,
			Type:      "link",
			Section:   "",
			Title:     "worldGround",
			To:        "/world/ground",
			Component: "lite/world/ground",
			Icon:      "ri-sun-line",
			Links:     nil,
		},
		{
			ID:        4,
			Type:      "link",
			Section:   "",
			Title:     "worldCave",
			To:        "/world/cave",
			Component: "lite/world/cave",
			Icon:      "ri-moon-line",
			Links:     nil,
		},
		{
			ID:        5,
			Type:      "link",
			Section:   "",
			Title:     "worldMod",
			To:        "/world/mod",
			Component: "lite/world/mod",
			Icon:      "ri-rocket-2-line",
			Links:     nil,
		},
		{
			ID:        6,
			Type:      "link",
			Section:   "",
			Title:     "logGame",
			To:        "/log/game",
			Component: "lite/log/game",
			Icon:      "ri-sword-line",
			Links:     nil,
		},
		{
			ID:        7,
			Type:      "link",
			Section:   "",
			Title:     "logChat",
			To:        "/log/chat",
			Component: "lite/log/chat",
			Icon:      "ri-chat-smile-3-line",
			Links:     nil,
		},
	}

	writeJSON(w, 200, msgSuccess, menu)
}

// validateUser 复刻原 gin binding:"required" 的必填校验：用户名与密码不能为空。
func validateUser(user utils.User) bool {
	return user.Username != "" && user.Password != ""
}
