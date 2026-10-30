package api

import (
	"dst-management-platform-api/lite/utils"
	"net/http"
)

func modGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, msgSuccess, utils.DB.Mod)
}

// modPut 只更新请求体中出现的字段，不带 mod 字段时保持原值
func modPut(w http.ResponseWriter, r *http.Request) {
	var form struct {
		Mod *string `json:"mod"`
	}
	if err := bindJSON(r, &form); err != nil {
		writeJSON(w, 400, msgBadRequest, nil)
		return
	}

	if form.Mod != nil {
		if err := utils.SaveModFile(*form.Mod, []utils.World{utils.DB.Ground, utils.DB.Cave}); err != nil {
			msg := "保存文件失败, " + err.Error()
			utils.Logger.Error(msg)

			writeJSON(w, 201, msg, nil)
		}
		utils.DB.Mod = *form.Mod
	}

	writeJSON(w, 200, msgUpdateSuccess, nil)
}
