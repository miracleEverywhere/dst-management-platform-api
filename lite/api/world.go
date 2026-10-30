package api

import (
	"dst-management-platform-api/lite/utils"
	"net/http"
)

func groundGet(w http.ResponseWriter, r *http.Request) {
	screenName := "DMP_LITE_" + utils.DB.Ground.Name
	utils.DB.Ground.Status = utils.ScreenAlive(screenName)

	writeJSON(w, 200, msgSuccess, utils.DB.Ground)
}

func groundPut(w http.ResponseWriter, r *http.Request) {
	var ground utils.World
	if err := bindJSON(r, &ground); err != nil || !validateWorld(ground) {
		writeJSON(w, 400, msgBadRequest, nil)
		return
	}

	if err := utils.SaveWorldFile(ground); err != nil {
		msg := "保存文件失败, " + err.Error()
		utils.Logger.Error(msg)

		writeJSON(w, 201, msg, nil)
	}

	utils.DB.Ground = ground

	writeJSON(w, 200, msgUpdateSuccess, nil)
}

func caveGet(w http.ResponseWriter, r *http.Request) {
	screenName := "DMP_LITE_" + utils.DB.Cave.Name
	utils.DB.Cave.Status = utils.ScreenAlive(screenName)

	writeJSON(w, 200, msgSuccess, utils.DB.Cave)
}

func cavePut(w http.ResponseWriter, r *http.Request) {
	var cave utils.World
	if err := bindJSON(r, &cave); err != nil || !validateWorld(cave) {
		writeJSON(w, 400, msgBadRequest, nil)
		return
	}

	if err := utils.SaveWorldFile(cave); err != nil {
		msg := "保存文件失败, " + err.Error()
		utils.Logger.Error(msg)

		writeJSON(w, 201, msg, nil)
	}

	utils.DB.Cave = cave

	writeJSON(w, 200, msgUpdateSuccess, nil)
}

// validateWorld 复刻原 gin binding:"required" 的必填校验。
// IsMaster 是 bool：原 required 会把 false 当成“缺失”，导致洞穴世界保存必然失败，这里不再校验。
func validateWorld(world utils.World) bool {
	return world.ID != 0 &&
		world.ServerPort != 0 &&
		world.MasterServerPort != 0 &&
		world.AuthenticationPort != 0
}
