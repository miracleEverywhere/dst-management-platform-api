package api

import (
	"dst-management-platform-api/lite/utils"
	"net/http"
	"time"
)

const (
	apiPrefix = "/v3"
)

// RegisterRoutes 注册全部路由。
func RegisterRoutes(mux *http.ServeMux) {
	// 公开接口
	mux.HandleFunc("GET "+apiPrefix+"/register", withReadLock(registerGet))
	mux.HandleFunc("POST "+apiPrefix+"/register", withWriteLock(registerPost))
	mux.HandleFunc("POST "+apiPrefix+"/login", withReadLock(loginPost))

	// 需要 token 的接口
	mux.HandleFunc("GET "+apiPrefix+"/menu", tokenCheck(menuGet))
	mux.HandleFunc("GET "+apiPrefix+"/log", tokenCheck(logGet))
	mux.HandleFunc("GET "+apiPrefix+"/sys", tokenCheck(sysInfoGet))
	mux.HandleFunc("GET "+apiPrefix+"/platform/game_version", tokenCheck(gameVersionGet))

	// 需要 token 和 写db 的接口
	mux.HandleFunc("GET "+apiPrefix+"/room", tokenCheck(withReadLock(roomGet)))
	mux.HandleFunc("PUT "+apiPrefix+"/room", tokenCheck(withWriteLock(roomPut)))
	mux.HandleFunc("GET "+apiPrefix+"/ground", tokenCheck(withReadLock(groundGet)))
	mux.HandleFunc("PUT "+apiPrefix+"/ground", tokenCheck(withWriteLock(groundPut)))
	mux.HandleFunc("GET "+apiPrefix+"/cave", tokenCheck(withReadLock(caveGet)))
	mux.HandleFunc("PUT "+apiPrefix+"/cave", tokenCheck(withWriteLock(cavePut)))
	mux.HandleFunc("GET "+apiPrefix+"/mod", tokenCheck(withReadLock(modGet)))
	mux.HandleFunc("PUT "+apiPrefix+"/mod", tokenCheck(withWriteLock(modPut)))
	mux.HandleFunc("GET "+apiPrefix+"/cc", tokenCheck(withWriteLock(connectCodeGet)))
	mux.HandleFunc("POST "+apiPrefix+"/exec", tokenCheck(withWriteLock(execPost)))
}

// withWriteLock 让写接口独占 DB：请求期间持有写锁，结束后若 DB 有变化就自动落盘。
//
// 注意：
//   - handler 内不要再调用 DB.Read() 或自行操作 DBMu，sync.RWMutex 不可重入，会死锁；
//   - db 必须保持可比较（全部是 string/int/bool），新增 map/slice 字段时这里要改成序列化比较；
//   - handler 内不要做长时间阻塞操作，否则会挡住其它写请求。
func withWriteLock(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		utils.DBMu.Lock()
		defer utils.DBMu.Unlock()

		before := utils.DB
		next(w, r)

		if utils.DB == before {
			return
		}

		if err := utils.DB.SaveLocked(); err != nil {
			// 响应已写出时无法再改状态码，只能记录日志
			utils.Logger.Error("保存数据库失败: " + err.Error())
		}
	}
}

// withReadLock 让只读接口共享读锁：彼此可并发，只与写请求互斥。
func withReadLock(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		utils.DBMu.RLock()
		defer utils.DBMu.RUnlock()

		next(w, r)
	}
}

// tokenCheck 校验 X-DMP-TOKEN 或 ?token=，剩余有效期不足一半时刷新并写入响应头。
func tokenCheck(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-DMP-TOKEN")
		if token == "" {
			token = r.URL.Query().Get("token")
		}
		if token == "" {
			writeJSON(w, 420, msgTokenFail, nil)
			return
		}

		// 只在读取密钥的瞬间持有读锁，避免和写请求嵌套加锁
		utils.DBMu.RLock()
		secret := utils.DB.System.JwtSecret
		utils.DBMu.RUnlock()

		// 密钥为空时拒绝通过，避免空密钥签出的 token 可被伪造
		if secret == "" {
			utils.Logger.Error("JWT密钥为空，拒绝校验token")
			writeJSON(w, 420, msgTokenFail, nil)
			return
		}

		claims, err := utils.ValidateJWT(token, []byte(secret))
		if err != nil {
			writeJSON(w, 420, msgTokenFail, nil)
			return
		}

		if claims.ExpiresAt != nil && shouldRefreshToken(claims.ExpiresAt.Time) {
			newToken, err := utils.GenerateJWT(utils.User{Username: claims.Username}, []byte(secret), utils.JwtExpirationHours)
			if err != nil {
				utils.Logger.Error("刷新JWT失败: " + err.Error())
			} else {
				w.Header().Set("X-DMP-NEW-TOKEN", newToken)
			}
		}

		next(w, r)
	}
}

func shouldRefreshToken(exp time.Time) bool {
	remainingTime := time.Until(exp)
	// 当剩余时间小于总有效期的 1/2 时刷新
	return remainingTime > 0 && remainingTime < time.Duration(utils.JwtExpirationHours)*time.Hour/2
}
