// Package ui 提供前端管理界面的静态文件服务。
//
// 构建流程：
//  1. 构建前端：cd ../../front && npm run build
//  2. 复制产物：cp -r ../../front/dist ./dist  (或使用 Makefile)
//  3. 构建后端：go build ./cmd/clients
//
// 开发模式：dist/index.html 为占位文件，前端通过 Vite 开发服务器(:5173)独立运行。
package ui

import (
	"embed"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httputil"
	"net/url"
)

//go:embed dist
var distFS embed.FS

// Handler 返回前端静态文件的 HTTP Handler。
// 支持 React Router 的 SPA 路由（所有非 /api 路径返回 index.html）。
func Handler() http.Handler {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		slog.Warn("前端静态文件加载失败，使用开发代理", "proxy", "http://localhost:5173")
		return devProxy()
	}

	fileServer := http.FileServer(http.FS(sub))

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" || path == "" {
			fileServer.ServeHTTP(w, r)
			return
		}

		// 尝试直接提供文件（去掉开头的 /）
		if _, err := sub.Open(path[1:]); err == nil {
			fileServer.ServeHTTP(w, r)
			return
		}

		// 文件不存在 → 返回 index.html（SPA 路由）
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/"
		fileServer.ServeHTTP(w, r2)
	})
}

// devProxy 在开发模式下将请求代理到 Vite 开发服务器。
func devProxy() http.Handler {
	target, _ := url.Parse("http://localhost:5173")
	return httputil.NewSingleHostReverseProxy(target)
}

// HasDist 返回是否存在前端构建产物。
func HasDist() bool {
	_, err := distFS.Open("dist/index.html")
	return err == nil
}
