package api

import (
	"dst-management-platform-api/lite/utils"
	"net/http"
)

// playersGet 返回后台任务缓存的在线玩家昵称。
// 缓存由 lite/utils 的定时任务维护，这里只读内存，不读取日志也不操作游戏。
func playersGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, 200, msgSuccess, utils.GetOnlinePlayers())
}
