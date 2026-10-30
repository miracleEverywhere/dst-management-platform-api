package api

import (
	"dst-management-platform-api/lite/utils"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// staticCacheSeconds 非 HTML 静态资源的浏览器缓存时间（7 天）。
const staticCacheSeconds = 7 * 24 * 3600

// RegisterStatic 注册前端静态资源路由。
//
// distDir 为 dist-lite 目录（与 dmp-lite 可执行文件同级）。
// 未命中的路径回退到 index.html，交给前端路由处理；
// /v3 下的未知路径不参与回退，避免把前端页面当成接口响应。
func RegisterStatic(mux *http.ServeMux, distDir string) {
	fileServer := http.FileServer(http.Dir(distDir))
	indexPath := filepath.Join(distDir, "index.html")

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}

		if r.URL.Path == "/v3" || strings.HasPrefix(r.URL.Path, "/v3/") {
			http.NotFound(w, r)
			return
		}

		// 先清洗再映射到磁盘，避免 ../ 之类的路径穿越
		name := path.Clean("/" + r.URL.Path)
		diskPath := filepath.Join(distDir, filepath.FromSlash(name))

		info, err := os.Stat(diskPath)
		switch {
		case err == nil && !info.IsDir() && filepath.Base(diskPath) != "index.html":
			w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%d", staticCacheSeconds))
			fileServer.ServeHTTP(w, r)
			return
		case err == nil && info.IsDir() && name != "/":
			// 目录不提供列表，避免暴露目录结构
			http.NotFound(w, r)
			return
		case err != nil && !os.IsNotExist(err):
			utils.Logger.Error("读取前端文件失败: " + err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		case err != nil && path.Ext(name) != "":
			// 带扩展名的资源缺失时直接 404，不回退 index.html
			http.NotFound(w, r)
			return
		}

		// 根路径、index.html 本身以及无扩展名的前端路由：回退到入口文件
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, indexPath)
	})
}
