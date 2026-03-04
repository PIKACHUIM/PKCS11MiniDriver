// Package api 提供 server-card 的 REST API 服务。
package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/globaltrusts/server-card/configs"
	"github.com/globaltrusts/server-card/internal/auth"
	"github.com/globaltrusts/server-card/internal/card"
	"github.com/globaltrusts/server-card/internal/storage"
	"golang.org/x/crypto/bcrypt"
)

// Server 是 REST API 服务器。
type Server struct {
	cfg        *configs.Config
	httpServer *http.Server
	jwtMgr     *auth.Manager
	cardSvc    *card.Service
	userRepo   *storage.UserRepo
	logRepo    *storage.LogRepo
}

// NewServer 创建 API 服务器。
func NewServer(cfg *configs.Config, jwtMgr *auth.Manager, cardSvc *card.Service, userRepo *storage.UserRepo, logRepo *storage.LogRepo) *Server {
	s := &Server{
		cfg:      cfg,
		jwtMgr:   jwtMgr,
		cardSvc:  cardSvc,
		userRepo: userRepo,
		logRepo:  logRepo,
	}

	mux := http.NewServeMux()
	s.registerRoutes(mux)

	s.httpServer = &http.Server{
		Addr:         cfg.API.Addr(),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	return s
}

// registerRoutes 注册所有路由。
func (s *Server) registerRoutes(mux *http.ServeMux) {
	// 健康检查（无需认证）
	mux.HandleFunc("GET /api/health", s.handleHealth)

	// 认证（无需认证）
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/refresh", s.authMiddleware(s.handleRefresh))

	// 卡片管理（需要认证）
	mux.HandleFunc("GET /api/cards", s.authMiddleware(s.handleListCards))
	mux.HandleFunc("POST /api/cards", s.authMiddleware(s.handleCreateCard))
	mux.HandleFunc("GET /api/cards/{uuid}", s.authMiddleware(s.handleGetCard))
	mux.HandleFunc("DELETE /api/cards/{uuid}", s.authMiddleware(s.handleDeleteCard))

	// 证书管理（需要认证）
	mux.HandleFunc("GET /api/cards/{uuid}/certs", s.authMiddleware(s.handleListCerts))
	mux.HandleFunc("POST /api/cards/{uuid}/certs", s.authMiddleware(s.handleImportCert))
	mux.HandleFunc("DELETE /api/cards/{uuid}/certs/{cert_uuid}", s.authMiddleware(s.handleDeleteCert))

	// 密钥生成（需要认证）
	mux.HandleFunc("POST /api/cards/{uuid}/keygen", s.authMiddleware(s.handleKeyGen))

	// 云端签名/解密（需要认证）
	mux.HandleFunc("POST /api/cards/{uuid}/sign", s.authMiddleware(s.handleSign))
	mux.HandleFunc("POST /api/cards/{uuid}/decrypt", s.authMiddleware(s.handleDecrypt))
}

// Start 启动 HTTP 服务。
func (s *Server) Start() error {
	slog.Info("server-card API 服务启动", "addr", s.cfg.API.Addr())
	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("API 服务异常退出", "error", err)
		}
	}()
	return nil
}

// Stop 优雅关闭 HTTP 服务。
func (s *Server) Stop(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

// ---- 认证中间件 ----

// authMiddleware 验证 JWT Token，将 Claims 注入 context。
func (s *Server) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := extractBearerToken(r)
		if token == "" {
			writeError(w, http.StatusUnauthorized, "缺少认证 Token")
			return
		}

		claims, err := s.jwtMgr.Verify(token)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "Token 无效或已过期")
			return
		}

		ctx := context.WithValue(r.Context(), claimsKey{}, claims)
		next(w, r.WithContext(ctx))
	}
}

// ---- 处理器 ----

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "service": "server-card"})
}

// LoginRequest 是登录请求体。
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	if req.Username == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "用户名和密码不能为空")
		return
	}

	user, err := s.userRepo.GetByUsername(r.Context(), req.Username)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	if !user.Enabled {
		writeError(w, http.StatusForbidden, "账号已禁用")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		writeError(w, http.StatusUnauthorized, "用户名或密码错误")
		return
	}

	token, err := s.jwtMgr.Sign(user.UUID, user.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "生成 Token 失败")
		return
	}

	// 记录登录日志
	s.logRepo.Create(r.Context(), &storage.Log{
		UserUUID:  user.UUID,
		Action:    "login",
		IPAddr:    r.RemoteAddr,
		UserAgent: r.UserAgent(),
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token":    token,
		"user_uuid": user.UUID,
		"username": user.Username,
	})
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	token, err := s.jwtMgr.Sign(claims.UserUUID, claims.Username)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "刷新 Token 失败")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"token": token})
}

// ---- 卡片处理器 ----

func (s *Server) handleListCards(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cards, err := s.cardSvc.ListCards(r.Context(), claims.UserUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"cards": cards, "total": len(cards)})
}

// CreateCardRequest 是创建卡片请求体。
type CreateCardRequest struct {
	CardName string `json:"card_name"`
	Remark   string `json:"remark"`
}

func (s *Server) handleCreateCard(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	var req CreateCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.CardName == "" {
		writeError(w, http.StatusBadRequest, "卡片名称不能为空")
		return
	}

	c, err := s.cardSvc.CreateCard(r.Context(), claims.UserUUID, req.CardName, req.Remark)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, c)
}

func (s *Server) handleGetCard(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cardUUID := r.PathValue("uuid")

	c, err := s.cardSvc.GetCard(r.Context(), cardUUID, claims.UserUUID)
	if err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleDeleteCard(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cardUUID := r.PathValue("uuid")

	if err := s.cardSvc.DeleteCard(r.Context(), cardUUID, claims.UserUUID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "卡片已删除"})
}

// ---- 证书处理器 ----

func (s *Server) handleListCerts(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cardUUID := r.PathValue("uuid")

	certs, err := s.cardSvc.ListCerts(r.Context(), cardUUID, claims.UserUUID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"certs": certs, "total": len(certs)})
}

// ImportCertRequest 是导入证书请求体。
type ImportCertRequest struct {
	CertType    string `json:"cert_type"`
	KeyType     string `json:"key_type"`
	CertContent []byte `json:"cert_content"` // Base64 编码的 DER
	Remark      string `json:"remark"`
}

func (s *Server) handleImportCert(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cardUUID := r.PathValue("uuid")

	var req ImportCertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	cert, err := s.cardSvc.ImportCert(r.Context(), cardUUID, claims.UserUUID, req.CertType, req.KeyType, req.Remark, req.CertContent)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cert)
}

func (s *Server) handleDeleteCert(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cardUUID := r.PathValue("uuid")
	certUUID := r.PathValue("cert_uuid")

	if err := s.cardSvc.DeleteCert(r.Context(), certUUID, cardUUID, claims.UserUUID); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "证书已删除"})
}

// KeyGenRequest 是密钥生成请求体。
type KeyGenRequest struct {
	KeyType string `json:"key_type"` // rsa2048/rsa4096/ec256/ec384/ec521
	Remark  string `json:"remark"`
}

func (s *Server) handleKeyGen(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cardUUID := r.PathValue("uuid")

	var req KeyGenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}
	if req.KeyType == "" {
		req.KeyType = "ec256"
	}

	cert, err := s.cardSvc.GenerateKeyPair(r.Context(), cardUUID, claims.UserUUID, req.KeyType, req.Remark)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, cert)
}

// SignRequest 是签名请求体。
type SignRequest struct {
	CertUUID  string `json:"cert_uuid"`
	Mechanism string `json:"mechanism"` // ECDSA_SHA256/SHA256_RSA_PKCS/...
	Data      []byte `json:"data"`      // 待签名数据（Base64）
}

func (s *Server) handleSign(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cardUUID := r.PathValue("uuid")

	var req SignRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	sig, err := s.cardSvc.Sign(r.Context(), req.CertUUID, cardUUID, claims.UserUUID, req.Mechanism, req.Data)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 记录签名日志
	s.logRepo.Create(r.Context(), &storage.Log{
		UserUUID:  claims.UserUUID,
		CardUUID:  cardUUID,
		CertUUID:  req.CertUUID,
		Action:    fmt.Sprintf("sign:%s", req.Mechanism),
		IPAddr:    r.RemoteAddr,
		UserAgent: r.UserAgent(),
	})

	writeJSON(w, http.StatusOK, map[string][]byte{"signature": sig})
}

// DecryptRequest 是解密请求体。
type DecryptRequest struct {
	CertUUID   string `json:"cert_uuid"`
	Mechanism  string `json:"mechanism"`
	Ciphertext []byte `json:"ciphertext"`
}

func (s *Server) handleDecrypt(w http.ResponseWriter, r *http.Request) {
	claims := claimsFromCtx(r.Context())
	cardUUID := r.PathValue("uuid")

	var req DecryptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "请求格式错误")
		return
	}

	plaintext, err := s.cardSvc.Decrypt(r.Context(), req.CertUUID, cardUUID, claims.UserUUID, req.Mechanism, req.Ciphertext)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.logRepo.Create(r.Context(), &storage.Log{
		UserUUID:  claims.UserUUID,
		CardUUID:  cardUUID,
		CertUUID:  req.CertUUID,
		Action:    fmt.Sprintf("decrypt:%s", req.Mechanism),
		IPAddr:    r.RemoteAddr,
		UserAgent: r.UserAgent(),
	})

	writeJSON(w, http.StatusOK, map[string][]byte{"plaintext": plaintext})
}

// ---- 工具函数 ----

type claimsKey struct{}

func claimsFromCtx(ctx context.Context) *auth.Claims {
	return ctx.Value(claimsKey{}).(*auth.Claims)
}

func extractBearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return h[7:]
	}
	return ""
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
