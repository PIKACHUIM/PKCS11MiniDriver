package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	config "github.com/globaltrusts/client-card/configs"
	"github.com/globaltrusts/client-card/internal/card"
	"github.com/globaltrusts/client-card/internal/storage"
	"github.com/globaltrusts/client-card/ui"
)

// Server 是 REST API 服务。
type Server struct {
	cfg      *config.APIConfig
	mux      *http.ServeMux
	httpSrv  *http.Server
	manager  *card.Manager
	db       *storage.DB
	userRepo *storage.UserRepo
	cardRepo *storage.CardRepo
	certRepo *storage.CertRepo
	logRepo  *storage.LogRepo
}

// NewServer 创建 API 服务实例。
func NewServer(cfg *config.APIConfig, manager *card.Manager, db *storage.DB) *Server {
	s := &Server{
		cfg:      cfg,
		mux:      http.NewServeMux(),
		manager:  manager,
		db:       db,
		userRepo: storage.NewUserRepo(db),
		cardRepo: storage.NewCardRepo(db),
		certRepo: storage.NewCertRepo(db),
		logRepo:  storage.NewLogRepo(db),
	}
	s.registerRoutes()
	return s
}

// registerRoutes 注册所有 API 路由。
// 使用 Go 1.22 ServeMux 的方法+路径模式。
func (s *Server) registerRoutes() {
	// 健康检查
	s.mux.HandleFunc("GET /api/health", s.handleHealth)

	// ---- 用户管理 ----
	s.mux.HandleFunc("GET /api/users", s.handleListUsers)
	s.mux.HandleFunc("POST /api/users", s.handleCreateUser)
	s.mux.HandleFunc("GET /api/users/{uuid}", s.handleGetUser)
	s.mux.HandleFunc("PUT /api/users/{uuid}", s.handleUpdateUser)
	s.mux.HandleFunc("DELETE /api/users/{uuid}", s.handleDeleteUser)

	// ---- 卡片管理 ----
	s.mux.HandleFunc("GET /api/cards", s.handleListCards)
	s.mux.HandleFunc("POST /api/cards", s.handleCreateCard)
	s.mux.HandleFunc("GET /api/cards/{uuid}", s.handleGetCard)
	s.mux.HandleFunc("PUT /api/cards/{uuid}", s.handleUpdateCard)
	s.mux.HandleFunc("DELETE /api/cards/{uuid}", s.handleDeleteCard)

	// ---- 证书管理 ----
	s.mux.HandleFunc("GET /api/cards/{card_uuid}/certs", s.handleListCerts)
	s.mux.HandleFunc("POST /api/cards/{card_uuid}/certs", s.handleCreateCert)
	s.mux.HandleFunc("GET /api/cards/{card_uuid}/certs/{uuid}", s.handleGetCert)
	s.mux.HandleFunc("DELETE /api/cards/{card_uuid}/certs/{uuid}", s.handleDeleteCert)

	// ---- 密钥操作 ----
	s.mux.HandleFunc("POST /api/cards/{card_uuid}/keygen", s.handleKeyGen)

	// ---- 日志查询 ----
	s.mux.HandleFunc("GET /api/logs", s.handleListLogs)

	// ---- Slot 状态 ----
	s.mux.HandleFunc("GET /api/slots", s.handleListSlots)

	// ---- 前端管理界面（静态文件）----
	// 所有非 /api/ 路径的请求都由前端 SPA 处理
	s.mux.Handle("/", ui.Handler())
}

// Handler 返回 HTTP 处理器，供测试使用。
func (s *Server) Handler() http.Handler {
	return corsMiddleware(logMiddleware(s.mux))
}

// Start 启动 HTTP 服务。
func (s *Server) Start() error {
	handler := corsMiddleware(logMiddleware(s.mux))

	s.httpSrv = &http.Server{
		Addr:         s.cfg.Addr(),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	slog.Info("REST API 服务已启动", "addr", s.cfg.Addr())

	go func() {
		var err error
		if s.cfg.TLSEnabled {
			err = s.httpSrv.ListenAndServeTLS(s.cfg.TLSCert, s.cfg.TLSKey)
		} else {
			err = s.httpSrv.ListenAndServe()
		}
		if err != nil && err != http.ErrServerClosed {
			slog.Error("REST API 服务异常退出", "error", err)
		}
	}()

	return nil
}

// Stop 优雅关闭 HTTP 服务。
func (s *Server) Stop(ctx context.Context) error {
	if s.httpSrv == nil {
		return nil
	}
	return s.httpSrv.Shutdown(ctx)
}

// ---- 中间件 ----

// responseWriter 包装 http.ResponseWriter，捕获状态码。
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

// logMiddleware 记录请求日志（含状态码）。
func logMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rw, r)
		slog.Debug("API 请求",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.status,
			"duration", time.Since(start),
			"remote", r.RemoteAddr,
		)
	})
}

// corsMiddleware 添加 CORS 头，允许前端跨域访问。
// 仅允许本地来源（127.0.0.1 / localhost），适合桌面应用场景。
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		// 只允许本地来源
		if isLocalOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		} else if origin == "" {
			// 非浏览器请求（如 curl、测试）直接放行
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// isLocalOrigin 判断是否为本地来源（localhost / 127.0.0.1）。
func isLocalOrigin(origin string) bool {
	if origin == "" {
		return false
	}
	for _, prefix := range []string{
		"http://localhost",
		"https://localhost",
		"http://127.0.0.1",
		"https://127.0.0.1",
	} {
		if len(origin) >= len(prefix) && origin[:len(prefix)] == prefix {
			return true
		}
	}
	return false
}

// ---- 健康检查 ----

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeOK(w, map[string]string{
		"status":  "ok",
		"version": "1.0.0",
		"time":    fmt.Sprintf("%d", time.Now().Unix()),
	})
}

// ---- Slot 状态 ----

func (s *Server) handleListSlots(w http.ResponseWriter, r *http.Request) {
	ids := s.manager.GetSlotList(false)
	type slotItem struct {
		SlotID      uint32 `json:"slot_id"`
		Description string `json:"description"`
		TokenPresent bool  `json:"token_present"`
	}

	items := make([]slotItem, 0, len(ids))
	for _, id := range ids {
		info, err := s.manager.GetSlotInfo(id)
		if err != nil {
			continue
		}
		items = append(items, slotItem{
			SlotID:       uint32(id),
			Description:  info.Description,
			TokenPresent: info.TokenPresent,
		})
	}
	writeOK(w, items)
}
