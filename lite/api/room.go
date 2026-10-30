package api

import (
	"dst-management-platform-api/lite/utils"
	"net/http"
)

func roomGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, msgSuccess, utils.DB.Room)
}

// roomPut 整体替换房间配置，请求体需要提交完整对象
func roomPut(w http.ResponseWriter, r *http.Request) {
	var room utils.Room
	if err := bindJSON(r, &room); err != nil || !validateRoom(room) {
		writeJSON(w, 400, msgBadRequest, nil)
		return
	}

	if err := utils.SaveRoomFile(room); err != nil {
		msg := "保存文件失败, " + err.Error()
		utils.Logger.Error(msg)

		writeJSON(w, 201, msg, nil)
	}

	utils.DB.Room = room

	writeJSON(w, 200, msgUpdateSuccess, nil)
}

// validateRoom 复刻原 gin binding:"required" 的必填校验。
func validateRoom(room utils.Room) bool {
	return room.GameName != "" &&
		room.GameMode != "" &&
		room.Token != "" &&
		room.MasterIP != "" &&
		room.MasterPort != 0 &&
		room.ClusterKey != ""
}
